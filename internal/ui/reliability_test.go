package ui

import (
	"btyper/internal/domain"
	"btyper/internal/storage"
	"btyper/internal/trainer"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type faultStore struct {
	domain.Store
	fail bool
}

func (s *faultStore) SaveSession(r domain.SessionResult, p map[rune]domain.CharacterProgress) error {
	if s.fail {
		return errors.New("disk unavailable")
	}
	return s.Store.SaveSession(r, p)
}
func (s *faultStore) SaveSettings(v domain.Settings) error {
	if s.fail {
		return errors.New("disk unavailable")
	}
	return s.Store.SaveSettings(v)
}

func drainCommand(t *testing.T, a *App, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, child := range batch {
			drainCommand(t, a, child)
		}
		return
	}
	switch msg.(type) {
	case workDone, dailySavedMsg, navigateMsg, textLoadedMsg:
	default:
		return
	}
	_, next := a.Update(msg)
	drainCommand(t, a, next)
}

func realApp(t *testing.T, lang string) (*App, *faultStore) {
	t.Helper()
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := &faultStore{Store: db}
	settings := domain.DefaultSettings()
	settings.UILanguage = lang
	settings.Language = lang
	a, err := New(store, settings)
	if err != nil {
		t.Fatal(err)
	}
	return a, store
}

func TestResultFailureBlocksNextLessonAndRetries(t *testing.T) {
	a, store := realApp(t, "en")
	a.ctx.engine = trainer.NewEngine("ee", domain.ModeLearn, "en", 'e', time.Time{})
	a.navigate(RoutePractice, nil)
	store.fail = true
	_, cmd := a.Update(key('e', 0, 0))
	drainCommand(t, a, cmd)
	_, cmd = a.Update(key('e', 0, 0))
	drainCommand(t, a, cmd)
	if a.route != RouteResult || !a.ctx.unsaved {
		t.Fatal("failed result not retained")
	}
	if !strings.Contains(a.View().Content, "retry saving") {
		t.Fatal("missing recovery instructions")
	}
	engine := a.ctx.engine
	_, cmd = a.Update(key('n', 0, 0))
	drainCommand(t, a, cmd)
	if a.ctx.engine != engine || a.route != RouteResult {
		t.Fatal("advanced despite failed save")
	}
	store.fail = false
	_, cmd = a.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	drainCommand(t, a, cmd)
	if a.ctx.unsaved || a.ctx.service.Progress()['e'].Samples != 2 {
		t.Fatal("retry did not save")
	}
	h, err := store.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatal(h, err)
	}
}

func TestFailedSettingsKeepLanguageAndTheme(t *testing.T) {
	a, store := realApp(t, "en")
	a.navigate(RouteSettings, nil)
	store.fail = true
	_, cmd := a.Update(key('l', 0, 0))
	drainCommand(t, a, cmd)
	if a.ctx.localizer.Language() != "en" || a.ctx.settings().UILanguage != "en" {
		t.Fatal("failed setting changed locale")
	}
	s := a.screen.(*settingsScreen)
	s.cursor = 9
	_, cmd = a.Update(key('l', 0, 0))
	drainCommand(t, a, cmd)
	if a.ctx.settings().ColorTheme != domain.ThemeViolet {
		t.Fatal("failed setting changed theme")
	}
}

func TestSmallTerminalRequiresExplicitResume(t *testing.T) {
	a, _ := realApp(t, "en")
	a.Start(domain.ModeLearn)
	a.Update(tea.WindowSizeMsg{Width: 59, Height: 15})
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a.Update(tea.FocusMsg{})
	if !a.ctx.engine.IsPaused() {
		t.Fatal("resize auto-resumed")
	}
	a.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if a.ctx.engine.IsPaused() {
		t.Fatal("explicit resume did not work")
	}
}

func TestScreensFitSupportedSizes(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, size := range [][2]int{{60, 16}, {80, 24}, {120, 40}} {
			t.Run(fmt.Sprintf("%s-%dx%d", lang, size[0], size[1]), func(t *testing.T) {
				a, _ := realApp(t, lang)
				a.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
				for _, route := range []Route{RouteMenu, RoutePractice, RouteResult, RouteSettings, RouteHelp, RouteStatistics, RouteText, RouteDrill} {
					a.ctx.engine = a.ctx.service.StartAdaptive(domain.ModeLearn, time.Now())
					a.ctx.result = domain.SessionResult{Mode: domain.ModeLearn, Language: lang, TargetRune: a.ctx.engine.Result.TargetRune}
					drainCommand(t, a, a.navigate(route, nil))
					view := a.View().Content
					if lipgloss.Width(view) > size[0] || lipgloss.Height(view) > size[1] {
						t.Fatalf("route %d: %dx%d exceeds %dx%d\n%s", route, lipgloss.Width(view), lipgloss.Height(view), size[0], size[1], view)
					}
					if strings.Contains(view, "!lesson.") || strings.Contains(view, "!today.") {
						t.Fatal("missing translation")
					}
				}
			})
		}
	}
}

func TestDailyCounterIncludesAbandonedTimeAndSaveDoesNotBlockInput(t *testing.T) {
	a, store := realApp(t, "en")
	now := time.Now()
	a.ctx.now = now
	e := trainer.NewEngine("abc", domain.ModeText, "en", 0, time.Time{})
	a.ctx.engine = e
	a.navigate(RoutePractice, nil)
	e.Input('a', now.Add(-2*time.Second))
	a.ctx.captureTime(now)
	cmd := a.ctx.saveDaily()
	if a.ctx.busy {
		t.Fatal("daily write blocks typing")
	}
	e.Input('b', now.Add(time.Second))
	a.ctx.captureTime(now.Add(time.Second))
	drainCommand(t, a, cmd)
	if len(a.ctx.dailyPending) == 0 {
		t.Fatal("older save erased newer time")
	}
	e.Pause(now.Add(time.Second))
	a.ctx.captureTime(now.Add(time.Second))
	a.ctx.engine = nil
	drainCommand(t, a, a.ctx.flushTime(nil))
	d, err := store.PracticeTime(now.Format("2006-01-02"))
	if err != nil || d != 3*time.Second {
		t.Fatal(d, err)
	}
	if !strings.Contains(a.ctx.todayView(false), "00:03") {
		t.Fatal(a.ctx.todayView(false))
	}
}

func TestWeeklyReviewIsAvailableFromStatistics(t *testing.T) {
	a, _ := realApp(t, "en")
	a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	drainCommand(t, a, a.navigate(RouteStatistics, nil))
	_, cmd := a.Update(key('4', 0, 0))
	drainCommand(t, a, cmd)
	s, ok := a.screen.(*statisticsScreen)
	if !ok || s.tab != 3 || !strings.Contains(a.View().Content, "Personal baseline") {
		t.Fatal("weekly review is not visible from statistics")
	}
}
