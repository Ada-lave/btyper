package ui

import (
	"fmt"
	"strings"
	"time"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/trainer"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
)

type practiceScreen struct {
	c        *Context
	bar      progress.Model
	lesson   LessonRenderer
	keyboard Keyboard
}

func newPracticeScreen(c *Context) Screen {
	return &practiceScreen{c: c, bar: progress.New(progress.WithDefaultBlend())}
}
func (s *practiceScreen) Activate() tea.Cmd { return nil }
func (s *practiceScreen) Resize(w, h int)   { s.bar.SetWidth(max(10, min(70, max(40, w-8)-20))) }
func (s *practiceScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	e := s.c.engine
	if e == nil {
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	now := time.Now()
	switch x := msg.(type) {
	case tea.FocusMsg:
		if e.IsPaused() {
			e.Resume(now)
		}
	case tea.KeyPressMsg:
		s.c.now = now
		switch x.String() {
		case "esc":
			if e.IsPaused() {
				e.Resume(now)
			} else {
				e.Pause(now)
			}
			return Action{}, nil
		case "ctrl+r":
			s.c.engine = trainer.NewEngine(string(e.Text), e.Result.Mode, e.Result.Language, e.Result.TargetRune, time.Time{})
			s.c.status.Clear()
			return Action{}, nil
		case "backspace":
			e.Backspace()
			return Action{}, nil
		}
		if e.IsPaused() {
			return Action{}, nil
		}
		rs := []rune(x.Key().Text)
		if len(rs) != 1 {
			if len(rs) > 1 {
				s.c.status.Set(s.c.t(i18n.PasteDisabled, nil))
			}
			return Action{}, nil
		}
		e.Input(rs[0], now)
		if e.Done() {
			result, err := s.c.service.Complete(e, now)
			s.c.result = result
			s.c.setStoreError(err)
			return Action{Kind: ActionNavigate, Route: RouteResult}, nil
		}
	}
	return Action{}, nil
}
func (s *practiceScreen) View() string {
	e := s.c.engine
	if e == nil {
		return ""
	}
	if e.IsPaused() {
		return s.c.theme.Border.Render(s.c.theme.Title.Render(s.c.t(i18n.Paused, nil)) + "\n\n" + Hotkeys(s.c, i18n.HotkeyContinue))
	}
	wpm, acc, d := e.Live(s.c.now)
	data := map[string]any{"Language": strings.ToUpper(e.Result.Language), "Mode": s.c.modeName(e.Result.Mode), "WPM": fmt.Sprintf("%.1f", wpm), "CPM": fmt.Sprintf("%.0f", wpm*5), "Accuracy": fmt.Sprintf("%.1f", acc*100), "Duration": formatDuration(d)}
	head := s.c.t(i18n.PracticeHeader, data)
	pct := float64(e.Pos) / float64(max(1, len(e.Text)))
	footer := fmt.Sprintf("%s  %s   %s", s.bar.ViewAs(pct), s.c.theme.Title.Render(fmt.Sprintf("%d/%d", e.Pos, len(e.Text))), Hotkeys(s.c, i18n.HotkeyPractice))
	out := s.c.theme.Title.Render(head) + "\n\n" + s.c.theme.Border.Width(max(48, min(100, s.c.width-10))).Render(s.lesson.View(e, s.c.theme, s.c.width)) + "\n\n" + footer
	if s.c.settings().ShowKeyboard && s.c.width >= 80 {
		out += "\n\n" + s.keyboard.View(s.c.service.Profile(), e, s.c)
	}
	return out
}

type resultScreen struct{ c *Context }

func newResultScreen(c *Context) Screen   { return &resultScreen{c} }
func (s *resultScreen) Activate() tea.Cmd { return nil }
func (s *resultScreen) Resize(w, h int)   {}
func (s *resultScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return Action{}, nil
	}
	switch k.String() {
	case "enter", "n":
		if s.c.result.Mode == domain.ModeText {
			if s.c.service.HasMoreCustomText() {
				s.c.engine, _ = s.c.service.NextCustomLesson()
				return Action{Kind: ActionNavigate, Route: RoutePractice}, nil
			}
			return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
		}
		s.c.engine = s.c.service.StartAdaptive(s.c.result.Mode, time.Now())
		return Action{Kind: ActionNavigate, Route: RoutePractice}, nil
	case "esc", "q":
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	return Action{}, nil
}
func (s *resultScreen) View() string {
	r := s.c.result
	target := ""
	if r.TargetRune != 0 {
		target = s.c.t(i18n.ResultTarget, map[string]any{"Key": string(r.TargetRune), "Confidence": fmt.Sprintf("%.0f", s.c.service.Progress()[r.TargetRune].Confidence*100)})
	}
	stats := s.c.t(i18n.ResultStats, map[string]any{"WPM": fmt.Sprintf("%6.1f", r.WPM), "CPM": fmt.Sprintf("%6.1f", r.CPM), "Accuracy": fmt.Sprintf("%6.1f", r.Accuracy*100), "Errors": fmt.Sprintf("%6d", r.Errors), "Duration": formatDuration(r.Duration)})
	return s.c.theme.Title.Render(s.c.t(i18n.ResultTitle, nil)) + target + "\n\n" + s.c.theme.Border.Render(stats) + "\n\n" + Hotkeys(s.c, i18n.HotkeyResult)
}
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}
