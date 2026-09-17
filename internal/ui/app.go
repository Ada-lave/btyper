package ui

import (
	"errors"
	"fmt"
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

type Context struct {
	service       *application.LessonService
	localizer     *i18n.Localizer
	theme         Theme
	dark          bool
	status        Status
	width, height int
	now           time.Time
	engine        *trainer.Engine
	result        domain.SessionResult
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
	ids := map[domain.Mode]i18n.MessageID{domain.ModeLearn: i18n.ModeLearn, domain.ModeImprove: i18n.ModeImprove, domain.ModeText: i18n.ModeText}
	return c.t(ids[mode], nil)
}

type App struct {
	ctx                           *Context
	route                         Route
	screen                        Screen
	terminalWidth, terminalHeight int
}

func New(store domain.Store, initial domain.Settings) (*App, error) {
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
	ctx := &Context{service: service, localizer: loc, dark: true, now: time.Now()}
	ctx.applyTheme(service.Settings().ColorTheme)
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
	switch x := msg.(type) {
	case tea.WindowSizeMsg:
		a.terminalWidth, a.terminalHeight = x.Width, x.Height
		a.ctx.width, a.ctx.height = layoutSize(x.Width, x.Height)
		a.screen.Resize(a.ctx.width, a.ctx.height)
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
		return a, tick()
	case tea.KeyPressMsg:
		if isCtrlKey(x, 'c') {
			return a, tea.Quit
		}
	}
	action, cmd := a.screen.Update(msg)
	if action.Kind == ActionQuit {
		return a, tea.Quit
	}
	if action.Kind == ActionNavigate {
		cmd = tea.Batch(cmd, a.navigate(action.Route, action.Payload))
	}
	return a, cmd
}
func (a *App) navigate(route Route, payload any) tea.Cmd {
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
	}
	panic("unknown route")
}
func (a *App) View() tea.View {
	body := ""
	if a.terminalWidth > 0 && (a.terminalWidth < 60 || a.terminalHeight < 16) {
		body = a.ctx.theme.Border.Render(a.ctx.t(i18n.TerminalSmall, map[string]any{"Width": a.terminalWidth, "Height": a.terminalHeight}))
	} else {
		body = a.screen.View()
	}
	if s := a.ctx.status.View(a.ctx.theme, a.ctx.now); s != "" {
		body += "\n\n" + s
	}
	if a.terminalWidth > 0 && a.terminalHeight > 0 {
		body = lipgloss.Place(
			a.terminalWidth,
			a.terminalHeight,
			lipgloss.Center,
			lipgloss.Center,
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
		return fmt.Errorf("%s", a.ctx.t(i18n.TextEmpty, nil))
	}
	a.ctx.engine, _ = a.ctx.service.NextCustomLesson()
	a.navigate(RoutePractice, nil)
	return nil
}
