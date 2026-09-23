package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/trainer"
	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type practiceScreen struct {
	c            *Context
	bar          progress.Model
	lesson       LessonRenderer
	keyboard     Keyboard
	pauseCursor  int
	manualPause  bool
	hideKeyboard bool
}

func newPracticeScreen(c *Context) Screen {
	return &practiceScreen{c: c, bar: progress.New(progress.WithDefaultBlend())}
}
func (s *practiceScreen) Activate() tea.Cmd {
	s.hideKeyboard = false
	return nil
}
func (s *practiceScreen) Resize(w, h int) { s.bar.SetWidth(max(10, min(70, max(40, w-8)-20))) }
func (s *practiceScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	e := s.c.engine
	if e == nil {
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	now := time.Now()
	switch x := msg.(type) {
	case tea.FocusMsg:
		if e.IsPaused() && !s.manualPause {
			e.Resume(now)
		}
	case tea.KeyPressMsg:
		s.c.now = now
		if s.manualPause {
			return s.updatePause(x, now)
		}
		switch {
		case x.Key().Code == tea.KeyEsc:
			e.Pause(now)
			s.manualPause = true
			s.pauseCursor = 0
			return Action{}, nil
		case isCtrlKey(x, 'r'):
			s.c.engine = restartEngine(e)
			s.c.status.Clear()
			return Action{}, nil
		case isCtrlKey(x, 'k'):
			s.hideKeyboard = !s.hideKeyboard
			return Action{}, nil
		case x.Key().Code == tea.KeyBackspace:
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
			if e.Result.TargetSkill != "" {
				kind := trainer.SkillKindForPattern(e.Result.TargetSkill)
				s.c.previousConfidence = s.c.service.Skills()[trainer.SkillKey(kind, e.Result.TargetSkill)].Confidence
			} else {
				s.c.previousConfidence = s.c.service.Progress()[e.Result.TargetRune].Confidence
			}
			return Action{}, saveResult(s.c, now)
		}
	}
	return Action{}, nil
}

func (s *practiceScreen) updatePause(k tea.KeyPressMsg, now time.Time) (Action, tea.Cmd) {
	if k.Key().Code == tea.KeyEsc {
		s.c.engine.Resume(now)
		s.manualPause = false
		return Action{}, nil
	}
	if isUp(k) {
		s.pauseCursor = (s.pauseCursor + 2) % 3
		return Action{}, nil
	}
	if isDown(k) {
		s.pauseCursor = (s.pauseCursor + 1) % 3
		return Action{}, nil
	}
	if k.Key().Code != tea.KeyEnter {
		return Action{}, nil
	}
	switch s.pauseCursor {
	case 0:
		s.c.engine.Resume(now)
		s.manualPause = false
	case 1:
		e := s.c.engine
		s.c.engine = restartEngine(e)
		s.manualPause = false
		s.c.status.Clear()
	case 2:
		s.c.engine = nil
		s.manualPause = false
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	return Action{}, nil
}

func restartEngine(e *trainer.Engine) *trainer.Engine {
	if e.Result.Mode == domain.ModeAdaptive || e.Result.Mode == domain.ModeDrill {
		return trainer.NewFocusedEngine(string(e.Text), e.Result.Mode, e.Result.Language, e.Result.TargetSkill, time.Time{})
	}
	return trainer.NewEngine(string(e.Text), e.Result.Mode, e.Result.Language, e.Result.TargetRune, time.Time{})
}
func (s *practiceScreen) View() string {
	e := s.c.engine
	if e == nil {
		return ""
	}
	if s.manualPause {
		items := []string{s.c.t(i18n.PauseContinue, nil), s.c.t(i18n.PauseRestart, nil), s.c.t(i18n.PauseExit, nil)}
		var lines []string
		for i, item := range items {
			prefix := "  "
			if i == s.pauseCursor {
				prefix, item = "› ", s.c.theme.Title.Render(item)
			}
			lines = append(lines, prefix+item)
		}
		return s.c.theme.Border.Render(s.c.theme.Title.Render(s.c.t(i18n.Paused, nil)) + "\n\n" + strings.Join(lines, "\n") + "\n\n" + Hotkeys(s.c, i18n.HotkeyPause))
	}
	wpm, acc, d := e.Live(s.c.now)
	data := map[string]any{"Language": strings.ToUpper(e.Result.Language), "Mode": s.c.modeName(e.Result.Mode), "WPM": fmt.Sprintf("%.1f", wpm), "CPM": fmt.Sprintf("%.0f", wpm*5), "Accuracy": fmt.Sprintf("%.1f", acc*100), "Duration": formatDuration(d)}
	head := s.c.t(i18n.PracticeHeader, data)
	pct := float64(e.Pos) / float64(max(1, len(e.Text)))
	footer := fmt.Sprintf("%s  %s\n%s", s.bar.ViewAs(pct), s.c.theme.Title.Render(fmt.Sprintf("%d/%d", e.Pos, len(e.Text))), Hotkeys(s.c, i18n.HotkeyPractice))
	header := strings.Join([]string{s.c.theme.Title.Render(head), s.c.todayView(true), lessonPurpose(s.c, e)}, "\n")
	blocks := []string{centerBlock(header, s.c.width)}
	if (e.Result.Mode == domain.ModeAdaptive || e.Result.Mode == domain.ModeDrill) && s.c.height >= 22 {
		blocks = append(blocks, centerBlock(LearningProgress{}.View(s.c.service.Profile(), s.c.service.Progress(), e.Result.TargetSkill, s.c.service.Skills(), s.c, min(100, s.c.width-8)), s.c.width))
	}
	blocks = append(blocks,
		centerBlock(s.c.theme.Border.Width(max(48, min(100, s.c.width-10))).Render(s.lesson.View(e, s.c.theme, s.c.width)), s.c.width),
		centerBlock(footer, s.c.width),
	)
	out := strings.Join(blocks, "\n\n")
	if s.c.settings().ShowKeyboard && !s.hideKeyboard && s.c.width >= 80 && s.c.height >= 24 {
		keyboard := centerBlock(s.keyboard.View(s.c.service.Profile(), e, s.c), s.c.width)
		withKeyboard := out + "\n\n" + keyboard
		if lipgloss.Height(lipgloss.NewStyle().Width(s.c.width).Render(withKeyboard)) <= s.c.height {
			out = withKeyboard
		}
	}
	return out
}

func centerBlock(view string, width int) string {
	return lipgloss.NewStyle().Width(max(1, width)).Align(lipgloss.Center).Render(view)
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
	if s.c.unsaved {
		if k.Key().Code == tea.KeyEnter {
			return Action{}, saveResult(s.c, time.Now())
		}
		if k.Key().Code == tea.KeyEsc {
			s.c.unsaved = false
			s.c.engine = nil
			s.c.status.Clear()
			return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
		}
		return Action{}, nil
	}
	switch {
	case k.Key().Code == tea.KeyEnter || isPlainKey(k, 'n'):
		if s.c.result.Mode == domain.ModeText {
			if s.c.service.HasMoreCustomText() {
				s.c.engine, _ = s.c.service.NextCustomLesson()
				return Action{Kind: ActionNavigate, Route: RoutePractice}, nil
			}
			return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
		}
		if s.c.result.Mode == domain.ModeDrill {
			return Action{Kind: ActionNavigate, Route: RouteDrill}, nil
		}
		s.c.engine = s.c.service.StartAdaptive(s.c.result.Mode, time.Now())
		return Action{Kind: ActionNavigate, Route: RoutePractice}, nil
	case k.Key().Code == tea.KeyEsc || isPlainKey(k, 'q'):
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	return Action{}, nil
}
func (s *resultScreen) View() string {
	r := s.c.result
	target := ""
	if r.TargetSkill != "" {
		kind := trainer.SkillKindForPattern(r.TargetSkill)
		target = s.c.t(i18n.ResultTarget, map[string]any{"Key": r.TargetSkill, "Confidence": fmt.Sprintf("%.0f", s.c.service.Skills()[trainer.SkillKey(kind, r.TargetSkill)].Confidence*100)})
	}
	stats := s.c.t(i18n.ResultStats, map[string]any{"WPM": fmt.Sprintf("%6.1f", r.WPM), "CPM": fmt.Sprintf("%6.1f", r.CPM), "Accuracy": fmt.Sprintf("%6.1f", r.Accuracy*100), "Errors": fmt.Sprintf("%6d", r.Errors), "Duration": formatDuration(r.Duration)})
	out := s.c.theme.Title.Render(s.c.t(i18n.ResultTitle, nil)) + target
	if r.Mode == domain.ModeAdaptive || r.Mode == domain.ModeDrill {
		out += "\n\n" + LearningProgress{}.View(s.c.service.Profile(), s.c.service.Progress(), r.TargetSkill, s.c.service.Skills(), s.c, min(100, s.c.width-8))
	}
	footer := Hotkeys(s.c, i18n.HotkeyResult)
	if s.c.unsaved {
		footer = s.c.t("result.unsaved", nil)
	} else {
		out += "\n" + resultFeedback(s.c)
	}
	return out + "\n\n" + s.c.theme.Border.Render(stats) + "\n" + s.c.todayView(false) + "\n\n" + footer
}

func saveResult(c *Context, now time.Time) tea.Cmd {
	var result domain.SessionResult
	c.engine.Finish(now)
	c.captureTime(now)
	return c.work(func() error {
		var err error
		result, err = c.service.Complete(c.engine, now)
		return err
	}, func(err error) tea.Cmd {
		c.result, c.unsaved = result, err != nil
		if err != nil {
			c.setStoreError(err)
		} else {
			c.status.Clear()
		}
		return c.flushTime(func(error) tea.Cmd { return func() tea.Msg { return navigateMsg{RouteResult} } })
	})
}

func lessonPurpose(c *Context, e *trainer.Engine) string {
	if e.Result.Mode == domain.ModeText {
		return c.t("lesson.custom", nil)
	}
	if e.Result.Mode == domain.ModeAdaptive && c.service.CalibrationRemaining() > 0 {
		return c.t("lesson.calibration", map[string]any{"Remaining": c.service.CalibrationRemaining()})
	}
	v := c.service.Progress()[e.Result.TargetRune]
	id := i18n.MessageID("lesson.repeat")
	key, streak := string(e.Result.TargetRune), v.MasteryStreak
	if e.Result.TargetSkill != "" {
		key = e.Result.TargetSkill
		kind := trainer.SkillKindForPattern(key)
		skill := c.service.Skills()[trainer.SkillKey(kind, key)]
		streak = skill.Level
	}
	if e.Result.Mode == domain.ModeAdaptive && streak == 0 {
		id = "lesson.learn"
	}
	return c.t(id, map[string]any{"Key": key, "Streak": streak})
}

func resultFeedback(c *Context) string {
	r := c.result
	var keys []rune
	p := c.service.Progress()
	for _, key := range c.service.Profile().UnlockOrder {
		if stat := r.Chars[key]; stat != nil && stat.Samples > 0 {
			keys = append(keys, key)
		}
	}
	sort.SliceStable(keys, func(i, j int) bool { return p[keys[i]].Confidence < p[keys[j]].Confidence })
	var labels []string
	for _, key := range keys[:min(3, len(keys))] {
		labels = append(labels, fmt.Sprintf("%c %.0f%%", key, p[key].Confidence*100))
	}
	out := ""
	if len(labels) > 0 {
		out = c.t("result.weak", map[string]any{"Keys": strings.Join(labels, ", ")})
	}
	if r.TargetSkill != "" {
		kind := trainer.SkillKindForPattern(r.TargetSkill)
		skill := c.service.Skills()[trainer.SkillKey(kind, r.TargetSkill)]
		after := skill.Confidence
		out += "\n" + c.t("result.change", map[string]any{"Key": r.TargetSkill, "Before": fmt.Sprintf("%.0f", c.previousConfidence*100), "After": fmt.Sprintf("%.0f", after*100)})
		due := "—"
		if !skill.DueAt.IsZero() {
			due = skill.DueAt.Local().Format("2006-01-02 15:04")
		}
		out += "\n" + c.t("result.schedule", map[string]any{"Level": skill.Level, "Due": due})
	}
	if r.Mode == domain.ModeAdaptive || r.Mode == domain.ModeLearn || r.Mode == domain.ModeImprove {
		next := c.service.StartAdaptive(domain.ModeAdaptive, time.Now())
		out += "\n" + c.t("result.next", nil) + " " + lessonPurpose(c, next)
	}
	return out
}
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}
