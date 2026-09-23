package storage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"btyper/internal/domain"
	"btyper/internal/trainer"
)

const maxProfileBytes = 1 << 20

func (s *SQLite) profilesDir() string { return filepath.Join(s.dir, "profiles") }

func (s *SQLite) UserProfiles() (map[string]domain.LanguageProfile, error) {
	out := map[string]domain.LanguageProfile{}
	entries, err := os.ReadDir(s.profilesDir())
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	if len(entries) > 256 {
		return nil, errors.New("too many user profiles")
	}
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Size() > maxProfileBytes {
			return nil, fmt.Errorf("profile %s: expected a regular JSON file up to 1 MiB", entry.Name())
		}
		data, err := os.ReadFile(filepath.Join(s.profilesDir(), entry.Name()))
		if err != nil {
			return nil, err
		}
		profile, err := trainer.LoadProfileJSON(data)
		if err != nil {
			return nil, fmt.Errorf("profile %s: %w", entry.Name(), err)
		}
		if profile.ID == "en" || profile.ID == "ru" || entry.Name() != profile.ID+".json" {
			return nil, fmt.Errorf("profile %s: filename must match a non-built-in ID", entry.Name())
		}
		out[profile.ID] = profile
	}
	return out, nil
}

func (s *SQLite) ImportUserProfile(reader io.Reader) (domain.LanguageProfile, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxProfileBytes+1))
	if err != nil {
		return domain.LanguageProfile{}, err
	}
	if len(data) > maxProfileBytes {
		return domain.LanguageProfile{}, errors.New("profile exceeds 1 MiB")
	}
	profile, err := trainer.LoadProfileJSON(data)
	if err != nil {
		return domain.LanguageProfile{}, err
	}
	if profile.ID == "en" || profile.ID == "ru" {
		return domain.LanguageProfile{}, errors.New("built-in profile IDs are reserved")
	}
	if profile.Name == "" {
		return domain.LanguageProfile{}, errors.New("profile.name: required for user profiles")
	}
	if err := os.MkdirAll(s.profilesDir(), 0o700); err != nil {
		return domain.LanguageProfile{}, err
	}
	file, err := os.CreateTemp(s.profilesDir(), ".import-*.tmp")
	if err != nil {
		return domain.LanguageProfile{}, err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return domain.LanguageProfile{}, err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return domain.LanguageProfile{}, err
	}
	if err = file.Close(); err != nil {
		return domain.LanguageProfile{}, err
	}
	target := filepath.Join(s.profilesDir(), profile.ID+".json")
	if err = os.Link(file.Name(), target); err != nil {
		return domain.LanguageProfile{}, fmt.Errorf("install profile %q: %w", profile.ID, err)
	}
	return profile, nil
}

func (s *SQLite) ExportUserProfile(id string, writer io.Writer) error {
	profile, ok := trainer.Profiles()[id]
	if !ok {
		profiles, err := s.UserProfiles()
		if err != nil {
			return err
		}
		profile, ok = profiles[id]
	}
	if !ok {
		return fmt.Errorf("profile %q not found", id)
	}
	data, err := trainer.MarshalProfileJSON(profile)
	if err != nil {
		return err
	}
	_, err = writer.Write(append(data, '\n'))
	return err
}
