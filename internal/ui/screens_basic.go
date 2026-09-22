package ui

import (
	"time"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func isBack(msg tea.Msg) bool {
	k, ok := msg.(tea.KeyPressMsg)
	return ok && (k.Key().Code == tea.KeyEsc || isPlainKey(k, 'q'))
}

type menuItem struct {
	route       Route
	mode        domain.Mode
	title, desc string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type menuScreen struct {
	c    *Context
	menu list.Model
}

func newMenuScreen(c *Context) Screen { s := &menuScreen{c: c}; s.rebuild(); return s }
func (s *menuScreen) rebuild() {
	c := s.c
	items := []list.Item{menuItem{RoutePractice, domain.ModeAdaptive, c.t(i18n.Learn, nil), c.t(i18n.LearnDesc, nil)}, menuItem{RouteText, domain.ModeText, c.t(i18n.CustomText, nil), c.t(i18n.TextDesc, nil)}, menuItem{RouteStatistics, "", c.t(i18n.History, nil), c.t(i18n.HistoryDesc, nil)}, menuItem{RouteSettings, "", c.t(i18n.Settings, nil), c.t(i18n.SettingsDesc, nil)}, menuItem{RouteHelp, "", c.t(i18n.Help, nil), c.t(i18n.HelpDesc, nil)}, menuItem{-1, "", c.t(i18n.Quit, nil), c.t(i18n.QuitDesc, nil)}}
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = c.theme.Text.PaddingLeft(2)
	delegate.Styles.NormalDesc = c.theme.Muted.PaddingLeft(2)
	delegate.Styles.SelectedTitle = c.theme.Title.Border(lipgloss.NormalBorder(), false, false, false, true).PaddingLeft(1)
	delegate.Styles.SelectedDesc = c.theme.Title.Border(lipgloss.NormalBorder(), false, false, false, true).PaddingLeft(1)
	s.menu = list.New(items, delegate, 72, 22)
	s.menu.Title = c.t(i18n.Title, nil)
	s.menu.Styles.Title = c.theme.Title
	s.menu.Styles.HelpStyle = c.theme.Muted
	s.menu.SetShowStatusBar(false)
	s.menu.SetFilteringEnabled(false)
}
func (s *menuScreen) Activate() tea.Cmd { return nil }
func (s *menuScreen) Resize(w, h int) {
	s.menu.SetSize(min(76, max(40, w)), max(10, min(22, h-4)))
}
func (s *menuScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if isUp(k) {
			s.menu.CursorUp()
			return Action{}, nil
		}
		if isDown(k) {
			s.menu.CursorDown()
			return Action{}, nil
		}
	}
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
		if item.mode == domain.ModeAdaptive {
			settings := s.c.settings()
			settings.Mode = item.mode
			return Action{}, s.c.work(func() error { return s.c.service.SaveSettings(settings) }, func(err error) tea.Cmd {
				if err != nil {
					s.c.setStoreError(err)
					return nil
				}
				s.c.engine = s.c.service.StartAdaptive(item.mode, time.Now())
				s.c.status.Clear()
				return func() tea.Msg { return navigateMsg{RoutePractice} }
			})
		}
		return Action{Kind: ActionNavigate, Route: item.route}, cmd
	}
	return Action{}, cmd
}
func (s *menuScreen) View() string { return s.menu.View() + "\n" + s.c.todayView(false) }

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
	home := map[string][]string{"en": {"F", "J"}, "ru": {"А", "О"}}[p.ID]
	guide := FingerGuide{}.View(p, s.c)
	return s.c.theme.Title.Render(s.c.t(i18n.Help, nil)) + "\n\n" + s.c.t(i18n.HelpBody, map[string]any{"Left": home[0], "Right": home[1]}) + "\n\n" + s.c.theme.Border.Render(guide) + "\n\n" + s.c.t(i18n.HelpFingers, nil) + "\n\n" + lipgloss.NewStyle().Width(max(40, s.c.width)).Render(s.c.t("help.metrics", nil)) + "\n\n" + Hotkeys(s.c, i18n.HotkeyBack)
}
