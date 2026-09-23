package ui

import (
	"testing"
	"time"

	"btyper/internal/domain"
	"btyper/internal/trainer"
	tea "charm.land/bubbletea/v2"
)

func TestPracticePauseMenu(t *testing.T) {
	engine := trainer.NewEngine("test", domain.ModeLearn, "en", 't', time.Time{})
	c := &Context{engine: engine}
	s := newPracticeScreen(c).(*practiceScreen)

	action, _ := s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if action.Kind != ActionNone || !s.manualPause || !engine.IsPaused() {
		t.Fatal("Esc did not open the pause menu")
	}
	s.Update(key('j', 0, 0))
	if s.pauseCursor != 1 {
		t.Fatal("j did not select restart")
	}
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if s.manualPause || c.engine == engine || c.engine.Pos != 0 {
		t.Fatal("restart did not replace the paused lesson")
	}

	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	s.Update(key('о', 0, 0))
	s.Update(key('о', 0, 0))
	action, _ = s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if action.Kind != ActionNavigate || action.Route != RouteMenu || c.engine != nil {
		t.Fatal("exit did not discard the lesson and return to the menu")
	}
}

func TestFocusDoesNotResumeManualPause(t *testing.T) {
	engine := trainer.NewEngine("test", domain.ModeLearn, "en", 0, time.Time{})
	c := &Context{engine: engine}
	s := newPracticeScreen(c).(*practiceScreen)
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	s.Update(tea.FocusMsg{})
	if !engine.IsPaused() || !s.manualPause {
		t.Fatal("focus resumed a manually paused lesson")
	}
}

func TestCtrlKTogglesKeyboardWithoutTyping(t *testing.T) {
	engine := trainer.NewEngine("test", domain.ModeLearn, "en", 't', time.Time{})
	s := newPracticeScreen(&Context{engine: engine}).(*practiceScreen)

	s.Update(key('k', tea.ModCtrl, 0))
	if !s.hideKeyboard || engine.Pos != 0 {
		t.Fatal("Ctrl+K did not hide the keyboard without entering text")
	}
	s.Update(key('л', tea.ModCtrl, 0))
	if s.hideKeyboard || engine.Pos != 0 {
		t.Fatal("Ctrl+K with Russian layout did not restore the keyboard")
	}
}

func TestRestartPreservesAdaptivePairTarget(t *testing.T) {
	e := trainer.NewAdaptiveEngine("br brown", "en", "br", time.Time{})
	c := &Context{engine: e}
	s := newPracticeScreen(c).(*practiceScreen)
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	s.Update(key('j', 0, 0))
	s.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if c.engine == e || c.engine.Result.TargetSkill != "br" || string(c.engine.Text) != "br brown" {
		t.Fatalf("restart changed scheduled target: %+v", c.engine.Result)
	}
}
