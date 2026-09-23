package ui

import (
	"testing"

	"btyper/internal/domain"
	tea "charm.land/bubbletea/v2"
)

func TestDrillSelectionStartsFocusedLesson(t *testing.T) {
	a, _ := realApp(t, "en")
	s := newDrillScreen(a.ctx).(*drillScreen)
	s.input.SetValue("zz")
	action, _ := s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if action.Kind != ActionNavigate || action.Route != RoutePractice || a.ctx.engine.Result.Mode != domain.ModeDrill || a.ctx.engine.Result.TargetSkill != "zz" {
		t.Fatalf("Drill target was not selected: %+v", action)
	}
}

func TestCustomTextShowsAnalysisBeforePractice(t *testing.T) {
	a, _ := realApp(t, "en")
	s := newTextScreen(a.ctx, nil).(*textScreen)
	s.area.SetValue("en en 7!")
	action, _ := s.Update(tea.KeyPressMsg(tea.Key{Code: 's', Mod: tea.ModCtrl}))
	if action.Kind != ActionNone || s.preview == nil || a.ctx.engine != nil {
		t.Fatal("custom text started without analysis preview")
	}
	action, _ = s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if action.Kind != ActionNavigate || action.Route != RoutePractice || a.ctx.engine == nil {
		t.Fatal("analysis confirmation did not start practice")
	}
}
