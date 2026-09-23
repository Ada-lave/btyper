package trainer

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"btyper/internal/domain"
)

//go:embed profiles/*.json
var builtInProfiles embed.FS

type ProfileDocument struct {
	ID            string   `json:"id"`
	Name          string   `json:"name,omitempty"`
	NameID        string   `json:"name_id,omitempty"`
	UnlockOrder   string   `json:"unlock_order"`
	Vowels        string   `json:"vowels"`
	Rows          []string `json:"rows"`
	FingerGroups  []string `json:"finger_groups"`
	Words         []string `json:"words"`
	FrequentPairs []string `json:"frequent_pairs"`
}

func Profiles() map[string]domain.LanguageProfile {
	profiles := make(map[string]domain.LanguageProfile, 2)
	for _, id := range []string{"en", "ru"} {
		data, err := builtInProfiles.ReadFile("profiles/" + id + ".json")
		if err != nil {
			panic(err)
		}
		profile, err := LoadProfileJSON(data)
		if err != nil {
			panic(err)
		}
		profiles[id] = profile
	}
	return profiles
}

func LoadProfileJSON(data []byte) (domain.LanguageProfile, error) {
	if !utf8.Valid(data) {
		return domain.LanguageProfile{}, errors.New("profile JSON: invalid UTF-8")
	}
	var doc ProfileDocument
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return domain.LanguageProfile{}, fmt.Errorf("profile JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return domain.LanguageProfile{}, errors.New("profile JSON: multiple documents")
	} else if !errors.Is(err, io.EOF) {
		return domain.LanguageProfile{}, fmt.Errorf("profile JSON: %w", err)
	}
	profile := domain.LanguageProfile{
		ID: doc.ID, Name: doc.Name, NameID: doc.NameID,
		UnlockOrder: []rune(doc.UnlockOrder), Vowels: []rune(doc.Vowels), Rows: doc.Rows,
		FingerGroups: doc.FingerGroups, Finger: fingers(doc.FingerGroups),
		Words: doc.Words, FrequentPairs: doc.FrequentPairs,
	}
	if err := ValidateProfile(profile); err != nil {
		return domain.LanguageProfile{}, err
	}
	return profile, nil
}

func MarshalProfileJSON(profile domain.LanguageProfile) ([]byte, error) {
	if err := ValidateProfile(profile); err != nil {
		return nil, err
	}
	doc := ProfileDocument{
		ID: profile.ID, Name: profile.Name, NameID: profile.NameID,
		UnlockOrder: string(profile.UnlockOrder), Vowels: string(profile.Vowels), Rows: profile.Rows,
		FingerGroups: profile.FingerGroups, Words: profile.Words,
		FrequentPairs: profile.FrequentPairs,
	}
	return json.MarshalIndent(doc, "", "  ")
}

func ValidateProfiles(profiles map[string]domain.LanguageProfile) error {
	for id, profile := range profiles {
		if profile.ID != id {
			return fmt.Errorf("profile %q: id does not match map key %q", profile.ID, id)
		}
		if err := ValidateProfile(profile); err != nil {
			return err
		}
	}
	return nil
}

func ValidateProfile(p domain.LanguageProfile) error {
	if p.ID == "" || len(p.ID) > 32 {
		return errors.New("profile.id: expected 1–32 ASCII letters, digits, hyphens, or underscores")
	}
	for _, r := range p.ID {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' {
			return errors.New("profile.id: expected lowercase ASCII letters, digits, hyphens, or underscores")
		}
	}
	if p.Name == "" && p.NameID == "" {
		return errors.New("profile.name: required")
	}
	if len(p.Name) > 80 || strings.TrimSpace(p.Name) != p.Name {
		return errors.New("profile.name: expected a trimmed name of up to 80 bytes")
	}
	for _, r := range p.Name {
		if unicode.IsControl(r) {
			return errors.New("profile.name: control characters are not allowed")
		}
	}
	if len(p.UnlockOrder) < 6 || len(p.UnlockOrder) > 128 {
		return errors.New("profile.unlock_order: expected 6–128 letters")
	}
	seen := map[rune]bool{}
	for i, r := range p.UnlockOrder {
		if seen[r] || !unicode.IsLetter(r) || !unicode.IsLower(r) {
			return fmt.Errorf("profile.unlock_order[%d]: expected a unique lowercase letter", i)
		}
		seen[r] = true
	}
	if len(p.Vowels) == 0 {
		return errors.New("profile.vowels: required")
	}
	vowels := map[rune]bool{}
	for i, r := range p.Vowels {
		if !seen[r] || vowels[r] {
			return fmt.Errorf("profile.vowels[%d]: expected a unique profile letter", i)
		}
		vowels[r] = true
	}
	if len(p.Rows) == 0 || len(p.Rows) > 8 {
		return errors.New("profile.rows: expected 1–8 rows")
	}
	rowSeen := map[rune]bool{}
	for i, row := range p.Rows {
		for _, r := range row {
			if !seen[r] || rowSeen[r] {
				return fmt.Errorf("profile.rows[%d]: unknown or duplicate letter %q", i, r)
			}
			rowSeen[r] = true
		}
	}
	if len(rowSeen) != len(seen) {
		return errors.New("profile.rows: every unlock letter must appear once")
	}
	if len(p.FingerGroups) != 8 {
		return errors.New("profile.finger_groups: expected eight finger groups")
	}
	fingerSeen := map[rune]bool{}
	for i, group := range p.FingerGroups {
		for _, r := range group {
			if !seen[r] || fingerSeen[r] {
				return fmt.Errorf("profile.finger_groups[%d]: unknown or duplicate letter %q", i, r)
			}
			fingerSeen[r] = true
		}
	}
	if len(fingerSeen) != len(seen) {
		return errors.New("profile.finger_groups: every unlock letter needs a finger")
	}
	if len(p.Words) == 0 || len(p.Words) > 10000 {
		return errors.New("profile.words: expected 1–10000 words")
	}
	for i, word := range p.Words {
		if word == "" || word != strings.ToLower(word) || strings.TrimSpace(word) != word || utf8.RuneCountInString(word) > 32 {
			return fmt.Errorf("profile.words[%d]: expected a lowercase word of up to 32 letters", i)
		}
		for _, r := range word {
			if !seen[r] {
				return fmt.Errorf("profile.words[%d]: unknown letter %q", i, r)
			}
		}
	}
	if len(p.FrequentPairs) == 0 || len(p.FrequentPairs) > 64 {
		return errors.New("profile.frequent_pairs: expected 1–64 pairs")
	}
	pairs := map[string]bool{}
	for i, pair := range p.FrequentPairs {
		rs := []rune(pair)
		if len(rs) != 2 || !seen[rs[0]] || !seen[rs[1]] || pairs[pair] {
			return fmt.Errorf("profile.frequent_pairs[%d]: expected a unique pair of profile letters", i)
		}
		pairs[pair] = true
	}
	return nil
}

func fingers(groups []string) map[rune]string {
	names := []string{"LP", "LR", "LM", "LI", "RI", "RM", "RR", "RP"}
	m := map[rune]string{}
	for i, group := range groups[:min(len(groups), len(names))] {
		for _, r := range group {
			m[r] = names[i]
		}
	}
	return m
}
