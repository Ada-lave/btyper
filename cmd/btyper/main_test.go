package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"btyper/internal/domain"
	"btyper/internal/trainer"
)

func TestApplyLocaleDefaults(t *testing.T) {
	tests := []struct {
		name, detected, cliLanguage string
		settings                    domain.Settings
		wantUI, wantTraining        string
	}{
		{name: "new Russian install", detected: "ru", settings: domain.DefaultSettings(), wantUI: "ru", wantTraining: "ru"},
		{name: "CLI language wins", detected: "ru", cliLanguage: "en", settings: domain.DefaultSettings(), wantUI: "ru", wantTraining: "en"},
		{name: "saved settings stay unchanged", detected: "ru", settings: domain.Settings{UILanguage: "en", Language: "en"}, wantUI: "en", wantTraining: "en"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyLocaleDefaults(tt.settings, tt.detected, tt.cliLanguage)
			if got.UILanguage != tt.wantUI || got.Language != tt.wantTraining {
				t.Fatalf("got UI/training %s/%s, want %s/%s", got.UILanguage, got.Language, tt.wantUI, tt.wantTraining)
			}
		})
	}
}

func TestProfileCommandsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	profile := trainer.Profiles()["en"]
	profile.ID, profile.Name, profile.NameID = "en_alt", "Alternative English", ""
	data, err := trainer.MarshalProfileJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(dir, "input.json")
	output := filepath.Join(dir, "output.json")
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := runProfileCommand([]string{"import", "--data-dir", dir, "--input", input}); err != nil {
		t.Fatal(err)
	}
	if err := runProfileCommand([]string{"export", "--data-dir", dir, "--id", "en_alt", "--output", output}); err != nil {
		t.Fatal(err)
	}
	exported, err := os.ReadFile(output)
	if err != nil || !bytes.Equal(bytes.TrimSpace(exported), data) {
		t.Fatalf("profile export changed data: %v", err)
	}
}
