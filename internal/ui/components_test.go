package ui

import (
	"strings"
	"testing"
	"time"
	"unicode"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/trainer"
	"github.com/charmbracelet/x/ansi"
)

func TestStatusExpires(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	var status Status
	status.SetFor("done", now.Add(2*time.Second))
	if got := status.View(NewTheme(true), now); got == "" {
		t.Fatalf("status disappeared too soon: %q", got)
	}
	status.ClearExpired(now.Add(2 * time.Second))
	if got := status.View(NewTheme(true), now.Add(2*time.Second)); got != "" {
		t.Fatalf("expired status is still visible: %q", got)
	}
}

func TestFingerGuideUsesSelectedLayout(t *testing.T) {
	loc, err := i18n.New("ru")
	if err != nil {
		t.Fatal(err)
	}
	c := &Context{localizer: loc, theme: NewTheme(true)}
	view := FingerGuide{}.View(trainer.Profiles()["ru"], c)
	for _, want := range []string{"А", "О", "левый мизинец", "правый мизинец"} {
		if !strings.Contains(view, want) {
			t.Fatalf("finger guide does not contain %q", want)
		}
	}
}

func TestLearningProgressShowsEveryLetterAndCurrentState(t *testing.T) {
	loc, err := i18n.New("en")
	if err != nil {
		t.Fatal(err)
	}
	profile := trainer.Profiles()["en"]
	progress := map[rune]domain.CharacterProgress{
		'e': {Rune: 'e', Confidence: 1, MasteryStreak: 2, Mastered: true, Unlocked: true},
		'n': {Rune: 'n', Confidence: .5},
		'i': {Rune: 'i', Confidence: .75},
		't': {Rune: 't', Confidence: .75},
		'r': {Rune: 'r', Confidence: .75},
		'l': {Rune: 'l', Confidence: .75},
	}
	c := &Context{localizer: loc, theme: NewTheme(true)}
	view := ansi.Strip(LearningProgress{}.View(profile, progress, c, 70))
	for _, want := range []string{"Letters mastered: 1/26", "current: N", "confidence 50%"} {
		if !strings.Contains(view, want) {
			t.Fatalf("progress view does not contain %q:\n%s", want, view)
		}
	}
	for _, r := range profile.UnlockOrder {
		if !strings.ContainsRune(view, unicode.ToUpper(r)) {
			t.Fatalf("progress view does not contain letter %q", r)
		}
	}
}

func TestKeyboardKeepsHeightAndShowsThumbHintForSpace(t *testing.T) {
	settings := domain.DefaultSettings()
	settings.UILanguage = "en"
	store := &testStore{settings: settings}
	app, err := New(store, settings)
	if err != nil {
		t.Fatal(err)
	}
	profile := trainer.Profiles()["en"]
	letter := Keyboard{}.View(profile, trainer.NewEngine("e", domain.ModeLearn, "en", 'e', time.Time{}), app.ctx)
	space := Keyboard{}.View(profile, trainer.NewEngine(" ", domain.ModeLearn, "en", 'e', time.Time{}), app.ctx)
	if strings.Count(letter, "\n") != strings.Count(space, "\n") {
		t.Fatalf("keyboard height changed for space: letter=%d lines, space=%d lines", strings.Count(letter, "\n")+1, strings.Count(space, "\n")+1)
	}
	if plain := ansi.Strip(space); !strings.Contains(plain, "press with either thumb") {
		t.Fatalf("space hint is missing: %q", plain)
	}
}
