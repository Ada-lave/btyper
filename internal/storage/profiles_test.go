package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"btyper/internal/trainer"
)

func TestUserProfileImportExportAndInvalidRollback(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	profile := trainer.Profiles()["en"]
	profile.ID, profile.Name, profile.NameID = "en_custom", "English custom", ""
	data, err := trainer.MarshalProfileJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportUserProfile(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ImportUserProfile(bytes.NewReader(data)); err == nil {
		t.Fatal("duplicate import overwrote existing profile")
	}
	invalid := bytes.Replace(data, []byte(`"vowels": "aeiouy"`), []byte(`"vowels": "a7"`), 1)
	if bytes.Equal(invalid, data) {
		t.Fatal("invalid fixture did not change profile")
	}
	if _, err := s.ImportUserProfile(bytes.NewReader(invalid)); err == nil || !strings.Contains(err.Error(), "vowels") {
		t.Fatalf("invalid profile was accepted: %v", err)
	}
	var exported bytes.Buffer
	if err := s.ExportUserProfile("en_custom", &exported); err != nil {
		t.Fatal(err)
	}
	if loaded, err := trainer.LoadProfileJSON(exported.Bytes()); err != nil || loaded.ID != "en_custom" {
		t.Fatalf("exported profile changed: %v %v", loaded.ID, err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	profiles, err := s.UserProfiles()
	if err != nil || profiles["en_custom"].Name != "English custom" {
		t.Fatalf("profile did not survive restart: %v %v", profiles, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profiles", "broken.json"), invalid, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserProfiles(); err == nil || !strings.Contains(err.Error(), "vowels") {
		t.Fatalf("invalid installed profile did not identify its field: %v", err)
	}
}
