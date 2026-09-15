package ui

import (
	"strings"
	"time"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func isBack(msg tea.Msg) bool {
	k, ok := msg.(tea.KeyPressMsg)
	return ok && (k.String() == "esc" || k.String() == "q")
}

type menuItem struct {
	route       Route
	mode        domain.Mode
	title, desc string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type onboardingScreen struct{ c *Context }

func newOnboardingScreen(c *Context) Screen   { return &onboardingScreen{c} }
func (s *onboardingScreen) Activate() tea.Cmd { return nil }
func (s *onboardingScreen) Resize(w, h int)   {}
func (s *onboardingScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return Action{}, nil
	}
	settings := s.c.settings()
	switch k.String() {
	case "left", "right", "tab":
		if settings.Language == "en" {
			settings.Language = "ru"
		} else {
			settings.Language = "en"
		}
		s.c.setStoreError(s.c.service.SaveSettings(settings))
	case "1":
		settings.Mode = domain.ModeLearn
		s.c.setStoreError(s.c.service.SaveSettings(settings))
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	case "2":
		settings.Mode = domain.ModeImprove
		s.c.setStoreError(s.c.service.SaveSettings(settings))
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	case "esc":
		return Action{Kind: ActionQuit}, nil
	}
	return Action{}, nil
}
func (s *onboardingScreen) View() string {
	p := s.c.service.Profile()
	name := s.c.t(i18n.MessageID(p.NameID), nil)
	return s.c.theme.Title.Render(s.c.t(i18n.OnboardTitle, nil)) + "\n\n" + s.c.t(i18n.OnboardLanguage, map[string]any{"Language": s.c.theme.Title.Render(name)}) + "\n\n" + s.c.t(i18n.OnboardLearn, nil) + "\n" + s.c.t(i18n.OnboardImprove, nil) + "\n\n" + Hotkeys(s.c, i18n.HotkeyQuit)
}

type menuScreen struct {
	c    *Context
	menu list.Model
}

func newMenuScreen(c *Context) Screen { s := &menuScreen{c: c}; s.rebuild(); return s }
func (s *menuScreen) rebuild() {
	c := s.c
	items := []list.Item{menuItem{RoutePractice, domain.ModeLearn, c.t(i18n.Learn, nil), c.t(i18n.LearnDesc, nil)}, menuItem{RoutePractice, domain.ModeImprove, c.t(i18n.Improve, nil), c.t(i18n.ImproveDesc, nil)}, menuItem{RouteText, domain.ModeText, c.t(i18n.CustomText, nil), c.t(i18n.TextDesc, nil)}, menuItem{RouteStatistics, "", c.t(i18n.History, nil), c.t(i18n.HistoryDesc, nil)}, menuItem{RouteSettings, "", c.t(i18n.Settings, nil), c.t(i18n.SettingsDesc, nil)}, menuItem{RouteHelp, "", c.t(i18n.Help, nil), c.t(i18n.HelpDesc, nil)}, menuItem{-1, "", c.t(i18n.Quit, nil), c.t(i18n.QuitDesc, nil)}}
	s.menu = list.New(items, list.NewDefaultDelegate(), 72, 22)
	s.menu.Title = c.t(i18n.Title, nil)
	s.menu.SetShowStatusBar(false)
	s.menu.SetFilteringEnabled(false)
}
func (s *menuScreen) Activate() tea.Cmd { return nil }
func (s *menuScreen) Resize(w, h int)   { s.menu.SetSize(min(76, max(40, w-8)), max(10, h-6)) }
func (s *menuScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	var cmd tea.Cmd
	s.menu, cmd = s.menu.Update(msg)
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" {
		item, ok := s.menu.SelectedItem().(menuItem)
		if !ok {
			return Action{}, cmd
		}
		if item.route < 0 {
			return Action{Kind: ActionQuit}, cmd
		}
		if item.mode == domain.ModeLearn || item.mode == domain.ModeImprove {
			s.c.engine = s.c.service.StartAdaptive(item.mode, time.Now())
			settings := s.c.settings()
			settings.Mode = item.mode
			s.c.setStoreError(s.c.service.SaveSettings(settings))
			s.c.status.Clear()
		}
		return Action{Kind: ActionNavigate, Route: item.route}, cmd
	}
	return Action{}, cmd
}
func (s *menuScreen) View() string { return s.menu.View() }

type helpScreen struct{ c *Context }

func newHelpScreen(c *Context) Screen   { return &helpScreen{c} }
func (s *helpScreen) Activate() tea.Cmd { return nil }
func (s *helpScreen) Resize(w, h int)   {}
func (s *helpScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	if isBack(msg) {
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	return Action{}, nil
}
func (s *helpScreen) View() string {
	p := s.c.service.Profile()
	return s.c.theme.Title.Render(s.c.t(i18n.Help, nil)) + "\n\n" + s.c.t(i18n.HelpBody, nil) + "\n\n" + strings.Join(p.Rows, "\n") + "\n\n" + s.c.t(i18n.HelpFingers, nil) + "\n\n" + Hotkeys(s.c, i18n.HotkeyPractice) + "\n" + Hotkeys(s.c, i18n.HotkeyBack)
}
