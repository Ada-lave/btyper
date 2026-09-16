package ui

import (
	"strings"
	"testing"
	"time"

	"btyper/internal/i18n"
	"btyper/internal/trainer"
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
