package main

import (
	"testing"

	"btyper/internal/domain"
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
