package trainer

import (
	"fmt"
	"strings"
	"unicode"

	"btyper/internal/domain"
)

func Profiles() map[string]domain.LanguageProfile {
	return map[string]domain.LanguageProfile{
		"en": {
			ID: "en", NameID: "language.english",
			UnlockOrder: []rune("enitrlsaudcyghpmobwfvkxzqj"),
			Rows:        []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
			Finger:      fingers([]string{"qaz", "wsx", "edc", "rfvtgb", "yhnujm", "ik", "ol", "p"}),
			Words:       []string{"the", "there", "their", "learn", "letter", "line", "time", "train", "read", "write", "speed", "skill", "type", "quick", "brown", "fox", "jump", "over", "keyboard", "practice", "screen", "focus", "better", "daily", "simple", "word", "sound", "hand", "finger", "home"},
		},
		"ru": {
			ID: "ru", NameID: "language.russian",
			UnlockOrder: []rune("оеаинтсрвлкмдпуяызьбгчйхжюшцщэфъё"),
			Rows:        []string{"ёйцукенгшщзхъ", "фывапролджэ", "ячсмитьбю"},
			Finger:      fingers([]string{"ёйфя", "цыч", "увс", "камепи", "нртьго", "шлб", "щдю", "зхъжэ"}),
			Words:       []string{"это", "как", "она", "они", "его", "для", "слово", "время", "рука", "палец", "строка", "текст", "урок", "скорость", "точность", "навык", "экран", "клавиша", "практика", "работа", "читать", "писать", "лучше", "каждый", "день", "просто", "новый", "звук", "дом", "мир"},
		},
	}
}

func ValidateProfiles(profiles map[string]domain.LanguageProfile) error {
	for id, profile := range profiles {
		if id == "" || profile.ID != id || len(profile.UnlockOrder) < 6 || len(profile.Rows) == 0 || len(profile.Words) == 0 {
			return fmt.Errorf("invalid language profile %q", id)
		}
		seen := map[rune]bool{}
		for _, r := range profile.UnlockOrder {
			if seen[r] || !unicode.IsLetter(r) || profile.Finger[r] == "" {
				return fmt.Errorf("invalid rune %q in profile %s", r, id)
			}
			seen[r] = true
		}
		for _, word := range profile.Words {
			if word != strings.ToLower(word) || strings.TrimSpace(word) != word {
				return fmt.Errorf("invalid word %q in profile %s", word, id)
			}
			for _, r := range word {
				if !seen[r] {
					return fmt.Errorf("word %q contains unknown rune in profile %s", word, id)
				}
			}
		}
	}
	return nil
}

func fingers(groups []string) map[rune]string {
	names := []string{"LP", "LR", "LM", "LI", "RI", "RM", "RR", "RP"}
	m := map[rune]string{}
	for i, group := range groups {
		for _, r := range group {
			m[r] = names[i]
		}
	}
	return m
}
