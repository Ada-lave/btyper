package ui

import (
	"errors"
	"testing"

	"btyper/internal/domain"
)

type testStore struct {
	settings domain.Settings
	progress map[rune]domain.CharacterProgress
}

func (s *testStore) LoadSettings() (domain.Settings, error) { return s.settings, nil }
func (s *testStore) SaveSettings(v domain.Settings) error   { s.settings = v; return nil }
func (s *testStore) LoadProgress(string) (map[rune]domain.CharacterProgress, error) {
	if s.progress == nil {
		s.progress = map[rune]domain.CharacterProgress{}
	}
	return s.progress, nil
}
func (*testStore) SaveSession(domain.SessionResult, map[rune]domain.CharacterProgress) error {
	return nil
}
func (*testStore) History(int) ([]domain.HistoryEntry, error) {
	return nil, errors.New("history must not be read during startup")
}
func (s *testStore) Reset() error { s.progress = map[rune]domain.CharacterProgress{}; return nil }
func (*testStore) Close() error   { return nil }

func TestNewStartsAtMainMenuWithoutReadingHistory(t *testing.T) {
	store := &testStore{settings: domain.DefaultSettings()}
	app, err := New(store, store.settings)
	if err != nil {
		t.Fatal(err)
	}
	if app.route != RouteMenu {
		t.Fatalf("started at route %d, want main menu", app.route)
	}
}

func TestLayoutUsesBoundedCenteredWorkspace(t *testing.T) {
	for _, tc := range []struct {
		terminalWidth, terminalHeight int
		wantWidth, wantHeight         int
	}{
		{80, 24, 76, 22},
		{160, 50, 104, 36},
		{64, 18, 60, 16},
	} {
		width, height := layoutSize(tc.terminalWidth, tc.terminalHeight)
		if width != tc.wantWidth || height != tc.wantHeight {
			t.Fatalf("layoutSize(%d, %d) = %dx%d, want %dx%d", tc.terminalWidth, tc.terminalHeight, width, height, tc.wantWidth, tc.wantHeight)
		}
	}
}

func TestSettingsVimKeysInRussianLayout(t *testing.T) {
	store := &testStore{settings: domain.DefaultSettings()}
	app, err := New(store, store.settings)
	if err != nil {
		t.Fatal(err)
	}
	s := newSettingsScreen(app.ctx).(*settingsScreen)
	s.Update(key('о', 0, 0))
	if s.cursor != 1 {
		t.Fatal("Russian physical j did not move down")
	}
	s.Update(key('о', 0, 0))
	before := app.ctx.settings().TargetWPM
	s.Update(key('р', 0, 0))
	if got := app.ctx.settings().TargetWPM; got != before-5 {
		t.Fatalf("Russian physical h changed speed to %v, want %v", got, before-5)
	}
	s.Update(key('д', 0, 0))
	if got := app.ctx.settings().TargetWPM; got != before {
		t.Fatalf("Russian physical l changed speed to %v, want %v", got, before)
	}
}

func TestLegacySettingsDefaultToVioletTheme(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.ColorTheme = ""
	store := &testStore{settings: settings}
	app, err := New(store, settings)
	if err != nil {
		t.Fatal(err)
	}
	if got := app.ctx.settings().ColorTheme; got != domain.ThemeViolet {
		t.Fatalf("legacy theme normalized to %q, want %q", got, domain.ThemeViolet)
	}
}

func TestThemeSettingAppliesAndPersists(t *testing.T) {
	store := &testStore{settings: domain.DefaultSettings()}
	app, err := New(store, store.settings)
	if err != nil {
		t.Fatal(err)
	}
	s := newSettingsScreen(app.ctx).(*settingsScreen)
	s.cursor = 6
	before := app.ctx.theme.Progress(.5).Render("x")
	s.Update(key('l', 0, 0))
	if got := app.ctx.settings().ColorTheme; got != domain.ThemeOcean {
		t.Fatalf("active theme is %q, want %q", got, domain.ThemeOcean)
	}
	if got := store.settings.ColorTheme; got != domain.ThemeOcean {
		t.Fatalf("saved theme is %q, want %q", got, domain.ThemeOcean)
	}
	if after := app.ctx.theme.Progress(.5).Render("x"); after == before {
		t.Fatal("theme change did not rebuild progress palette")
	}
}

var _ domain.Store = (*testStore)(nil)
