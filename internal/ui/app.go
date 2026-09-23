package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"btyper/internal/application"
	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/trainer"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Route int

const (
	RouteMenu Route = iota
	RoutePractice
	RouteResult
	RouteText
	RoutePicker
	RouteStatistics
	RouteSettings
	RouteHelp
	RouteDrill
)

type ActionKind int

const (
	ActionNone ActionKind = iota
	ActionNavigate
	ActionQuit
)

type Action struct {
	Kind    ActionKind
	Route   Route
	Payload any
}
type Screen interface {
	Activate() tea.Cmd
	Update(tea.Msg) (Action, tea.Cmd)
	View() string
	Resize(width, height int)
}
type tickMsg time.Time
type navigateMsg struct{ route Route }

type Context struct {
	service            *application.LessonService
	localizer          *i18n.Localizer
	theme              Theme
	dark               bool
	status             Status
	width, height      int
	now                time.Time
	engine             *trainer.Engine
	result             domain.SessionResult
	busy               bool
	unsaved            bool
	previousConfidence float64
	busyView           string
	dailyTotals        map[string]time.Duration
	goalDays           map[string]time.Duration
	dailySeen          map[string]time.Duration
	dailyPending       map[string]domain.PracticeTime
	lastDailySave      time.Time
	dailyInFlight      bool
	queuedWork         tea.Cmd
}

func dayStart(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func (c *Context) todayView(includeCurrent bool) string {
	d := c.dailyTotals[c.now.Format("2006-01-02")]
	status := application.DailyGoalStatus(c.now, c.goalDays, c.settings().DailyGoalMinutes)
	state := c.t("goal.in_progress", nil)
	if status.Completed {
		state = c.t("goal.complete", nil)
	}
	return c.t("today.time", map[string]any{"Time": formatDuration(d)}) + "  ·  " +
		c.t("goal.status", map[string]any{"Goal": c.settings().DailyGoalMinutes, "State": state, "Streak": status.Streak})
}

func (c *Context) captureTime(now time.Time) {
	if c.engine == nil {
		return
	}
	if c.dailyTotals == nil {
		c.dailyTotals = map[string]time.Duration{}
		c.dailySeen = map[string]time.Duration{}
		c.dailyPending = map[string]domain.PracticeTime{}
	}
	if c.goalDays == nil {
		c.goalDays = map[string]time.Duration{}
	}
	for _, entry := range c.engine.PracticeTimes(now) {
		key := entry.AttemptID + "/" + entry.Day
		if delta := entry.Duration - c.dailySeen[key]; delta > 0 {
			c.dailyTotals[entry.Day] += delta
			c.goalDays[entry.Day] += delta
			c.dailySeen[key] = entry.Duration
			c.dailyPending[key] = entry
		}
	}
}

func (c *Context) flushTime(done func(error) tea.Cmd) tea.Cmd {
	entries := make([]domain.PracticeTime, 0, len(c.dailyPending))
	for _, entry := range c.dailyPending {
		entries = append(entries, entry)
	}
	c.lastDailySave = c.now
	return c.work(func() error { return c.service.SavePracticeTime(entries) }, func(err error) tea.Cmd {
		if err == nil {
			for _, entry := range entries {
				delete(c.dailyPending, entry.AttemptID+"/"+entry.Day)
			}
		} else {
			c.setStoreError(err)
		}
		if done != nil {
			return done(err)
		}
		return nil
	})
}

type workDone struct {
	err  error
	done func(error) tea.Cmd
}

type dailySavedMsg struct {
	entries []domain.PracticeTime
	err     error
}

func (c *Context) saveDaily() tea.Cmd {
	entries := make([]domain.PracticeTime, 0, len(c.dailyPending))
	for _, entry := range c.dailyPending {
		entries = append(entries, entry)
	}
	c.lastDailySave, c.dailyInFlight = c.now, true
	return func() tea.Msg { return dailySavedMsg{entries: entries, err: c.service.SavePracticeTime(entries)} }
}

func (c *Context) work(work func() error, done func(error) tea.Cmd) tea.Cmd {
	c.busy = true
	c.busyView = c.t("work.wait", nil)
	cmd := func() tea.Msg { return workDone{err: work(), done: done} }
	if c.dailyInFlight {
		c.queuedWork = cmd
		return nil
	}
	return cmd
}

func (c *Context) t(id i18n.MessageID, data map[string]any) string { return c.localizer.Text(id, data) }
func (c *Context) settings() domain.Settings                       { return c.service.Settings() }
func (c *Context) applyTheme(theme domain.ColorTheme)              { c.theme = NewColorTheme(c.dark, theme) }
func (c *Context) setStoreError(err error) {
	if err != nil {
		c.status.Set(c.t(i18n.StoreError, map[string]any{"Error": err}))
	}
}
func (c *Context) modeName(mode domain.Mode) string {
	ids := map[domain.Mode]i18n.MessageID{domain.ModeAdaptive: i18n.ModeLearn, domain.ModeLearn: i18n.ModeLearn, domain.ModeImprove: i18n.ModeImprove, domain.ModeText: i18n.ModeText, domain.ModeDrill: "mode.drill"}
	return c.t(ids[mode], nil)
}

type App struct {
	ctx                           *Context
	route                         Route
	screen                        Screen
	terminalWidth, terminalHeight int
	scroll                        int
}

func New(store domain.Store, initial domain.Settings) (*App, error) {
	repaired := application.NormalizeSettings(initial) != initial
	if initial.UILanguage == "" {
		initial.UILanguage = i18n.Detect()
	}
	loc, err := i18n.New(initial.UILanguage)
	if err != nil {
		return nil, err
	}
	initial.UILanguage = loc.Language()
	service, err := application.NewLessonService(store, initial)
	if err != nil {
		return nil, err
	}
	if err := service.SaveSettings(service.Settings()); err != nil {
		return nil, err
	}
	ctx := &Context{service: service, localizer: loc, dark: true, now: time.Now()}
	ctx.applyTheme(service.Settings().ColorTheme)
	day := ctx.now.Format("2006-01-02")
	duration, err := service.PracticeTime(day)
	if err != nil {
		ctx.setStoreError(err)
	}
	ctx.dailyTotals = map[string]time.Duration{day: duration}
	ctx.goalDays, err = service.PracticeDays()
	if err != nil {
		ctx.setStoreError(err)
		ctx.goalDays = map[string]time.Duration{}
	}
	if ctx.goalDays[day] < duration {
		ctx.goalDays[day] = duration
	}
	ctx.dailySeen = map[string]time.Duration{}
	ctx.dailyPending = map[string]domain.PracticeTime{}
	ctx.lastDailySave = ctx.now
	if repaired {
		ctx.status.Set(ctx.t("settings.repaired", nil))
	}
	a := &App{ctx: ctx, route: RouteMenu}
	a.screen = a.newScreen(a.route, nil)
	return a, nil
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(tea.RequestBackgroundColor, tick(), a.screen.Activate())
}
func tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !a.ctx.busy {
		a.ctx.captureTime(time.Now())
	}
	switch x := msg.(type) {
	case dailySavedMsg:
		a.ctx.dailyInFlight = false
		if x.err != nil {
			a.ctx.setStoreError(x.err)
		} else {
			for _, entry := range x.entries {
				key := entry.AttemptID + "/" + entry.Day
				if a.ctx.dailyPending[key].Duration <= entry.Duration {
					delete(a.ctx.dailyPending, key)
				}
			}
		}
		cmd := a.ctx.queuedWork
		a.ctx.queuedWork = nil
		return a, cmd
	case navigateMsg:
		return a, a.navigate(x.route, nil)
	case textLoadedMsg:
		return a, a.navigate(RouteText, x.text)
	case workDone:
		a.ctx.busy = false
		if x.done != nil {
			return a, x.done(x.err)
		}
		a.ctx.setStoreError(x.err)
		return a, nil
	case tea.WindowSizeMsg:
		a.terminalWidth, a.terminalHeight = x.Width, x.Height
		a.ctx.width, a.ctx.height = layoutSize(x.Width, x.Height)
		a.screen.Resize(a.ctx.width, a.ctx.height)
		if (x.Width < 60 || x.Height < 16) && a.route == RoutePractice {
			if s, ok := a.screen.(*practiceScreen); ok && a.ctx.engine != nil {
				a.ctx.engine.Pause(time.Now())
				s.manualPause = true
			}
		}
	case tea.BackgroundColorMsg:
		a.ctx.dark = x.IsDark()
		a.ctx.applyTheme(a.ctx.settings().ColorTheme)
	case tea.BlurMsg:
		if a.ctx.engine != nil && a.route == RoutePractice {
			a.ctx.engine.Pause(time.Now())
		}
	case tickMsg:
		a.ctx.now = time.Time(x)
		a.ctx.status.ClearExpired(a.ctx.now)
		if !a.ctx.busy && !a.ctx.dailyInFlight && len(a.ctx.dailyPending) > 0 && a.ctx.now.Sub(a.ctx.lastDailySave) >= 5*time.Second {
			return a, tea.Batch(tick(), a.ctx.saveDaily())
		}
		return a, tick()
	case tea.KeyPressMsg:
		if a.ctx.busy {
			return a, nil
		}
		if x.Key().Code == tea.KeyPgDown {
			a.scroll += max(1, a.terminalHeight-3)
			return a, nil
		}
		if x.Key().Code == tea.KeyPgUp {
			a.scroll = max(0, a.scroll-max(1, a.terminalHeight-3))
			return a, nil
		}
		if a.route == RoutePractice && a.terminalWidth > 0 && (a.terminalWidth < 60 || a.terminalHeight < 16) {
			return a, nil
		}
		if isCtrlKey(x, 'c') {
			if a.ctx.unsaved {
				return a, nil
			}
			return a, a.quit()
		}
	}
	if a.ctx.busy {
		return a, nil
	}
	action, cmd := a.screen.Update(msg)
	if !a.ctx.busy {
		a.ctx.captureTime(time.Now())
	}
	if action.Kind == ActionQuit {
		return a, a.quit()
	}
	if action.Kind == ActionNavigate {
		cmd = tea.Batch(cmd, a.navigate(action.Route, action.Payload))
	}
	return a, cmd
}

func (a *App) quit() tea.Cmd {
	if a.ctx.engine != nil {
		a.ctx.engine.Pause(time.Now())
		a.ctx.captureTime(time.Now())
	}
	if len(a.ctx.dailyPending) == 0 {
		return tea.Quit
	}
	return a.ctx.flushTime(func(err error) tea.Cmd {
		if err != nil {
			return nil
		}
		return tea.Quit
	})
}
func (a *App) navigate(route Route, payload any) tea.Cmd {
	a.scroll = 0
	a.route = route
	a.screen = a.newScreen(route, payload)
	a.screen.Resize(a.ctx.width, a.ctx.height)
	return a.screen.Activate()
}
func (a *App) newScreen(route Route, payload any) Screen {
	switch route {
	case RouteMenu:
		return newMenuScreen(a.ctx)
	case RoutePractice:
		return newPracticeScreen(a.ctx)
	case RouteResult:
		return newResultScreen(a.ctx)
	case RouteText:
		return newTextScreen(a.ctx, payload)
	case RoutePicker:
		return newPickerScreen(a.ctx, payload)
	case RouteStatistics:
		return newStatisticsScreen(a.ctx)
	case RouteSettings:
		return newSettingsScreen(a.ctx)
	case RouteHelp:
		return newHelpScreen(a.ctx)
	case RouteDrill:
		return newDrillScreen(a.ctx)
	}
	panic("unknown route")
}
func (a *App) View() tea.View {
	body := ""
	if a.ctx.busy {
		body = a.ctx.busyView
	} else if a.terminalWidth > 0 && (a.terminalWidth < 60 || a.terminalHeight < 16) {
		body = a.ctx.theme.Border.Render(a.ctx.t(i18n.TerminalSmall, map[string]any{"Width": a.terminalWidth, "Height": a.terminalHeight}))
	} else {
		body = a.screen.View()
	}
	if s := a.ctx.status.View(a.ctx.theme, a.ctx.now); s != "" {
		body += "\n\n" + s
	}
	if a.terminalWidth > 0 && a.terminalHeight > 0 {
		body = lipgloss.NewStyle().MaxWidth(a.terminalWidth).Render(body)
		lines := strings.Split(body, "\n")
		if len(lines) > a.terminalHeight {
			visible := max(1, a.terminalHeight-1)
			start := min(a.scroll, len(lines)-visible)
			body = strings.Join(lines[start:start+visible], "\n") + "\n" + lipgloss.NewStyle().MaxWidth(a.terminalWidth).Render(a.ctx.t("view.scroll", nil))
		}
		horizontal, vertical := positionAlignment(a.ctx.settings().Position)
		body = lipgloss.Place(
			a.terminalWidth,
			a.terminalHeight,
			horizontal,
			vertical,
			body,
			lipgloss.WithWhitespaceChars(" "),
		)
	}
	v := tea.NewView(body)
	v.AltScreen = true
	v.ReportFocus = true
	v.KeyboardEnhancements.ReportAlternateKeys = true
	v.WindowTitle = "btyper"
	return v
}

func positionAlignment(position domain.InterfacePosition) (lipgloss.Position, lipgloss.Position) {
	horizontal, vertical := lipgloss.Center, lipgloss.Center
	switch position {
	case domain.PositionTopLeft, domain.PositionCenterLeft, domain.PositionBottomLeft:
		horizontal = lipgloss.Left
	case domain.PositionTopRight, domain.PositionCenterRight, domain.PositionBottomRight:
		horizontal = lipgloss.Right
	}
	switch position {
	case domain.PositionTopLeft, domain.PositionTopCenter, domain.PositionTopRight:
		vertical = lipgloss.Top
	case domain.PositionBottomLeft, domain.PositionBottomCenter, domain.PositionBottomRight:
		vertical = lipgloss.Bottom
	}
	return horizontal, vertical
}

func layoutSize(terminalWidth, terminalHeight int) (int, int) {
	return max(1, min(104, terminalWidth-4)), max(1, min(36, terminalHeight-2))
}

func (a *App) Start(mode domain.Mode) {
	s := a.ctx.settings()
	s.Mode = mode
	a.ctx.setStoreError(a.ctx.service.SaveSettings(s))
	if mode == domain.ModeText {
		a.navigate(RouteText, nil)
	} else {
		a.ctx.engine = a.ctx.service.StartAdaptive(mode, time.Now())
		a.navigate(RoutePractice, nil)
	}
}
func (a *App) StartCustomText(raw string) error {
	err := a.ctx.service.SetCustomText(raw)
	if err != nil {
		if errors.Is(err, application.ErrInvalidUTF8) {
			return fmt.Errorf("%s", a.ctx.t(i18n.InvalidUTF8, nil))
		}
		if errors.Is(err, application.ErrTextTooLarge) {
			return fmt.Errorf("%s", a.ctx.t("text.too_large", nil))
		}
		return fmt.Errorf("%s", a.ctx.t(i18n.TextEmpty, nil))
	}
	a.ctx.engine, _ = a.ctx.service.NextCustomLesson()
	a.navigate(RoutePractice, nil)
	return nil
}
