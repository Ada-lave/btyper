package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	tea "charm.land/bubbletea/v2"
)

type settingsScreen struct {
	c       *Context
	cursor  int
	confirm bool
}

func newSettingsScreen(c *Context) Screen   { return &settingsScreen{c: c} }
func (s *settingsScreen) Activate() tea.Cmd { return nil }
func (s *settingsScreen) Resize(w, h int)   {}
func (s *settingsScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return Action{}, nil
	}
	if s.confirm {
		if isPlainKey(k, 'y') {
			s.confirm = false
			return Action{}, s.c.work(s.c.service.Reset, func(err error) tea.Cmd {
				if err != nil {
					s.c.setStoreError(err)
				} else {
					s.c.engine = nil
					s.c.dailyTotals = map[string]time.Duration{}
					s.c.goalDays = map[string]time.Duration{}
					s.c.dailySeen = map[string]time.Duration{}
					s.c.dailyPending = map[string]domain.PracticeTime{}
					s.c.status.SetFor(s.c.t(i18n.SettingsResetDone, nil), time.Now().Add(2*time.Second))
				}
				return nil
			})
		}
		s.confirm = false
		return Action{}, nil
	}
	switch {
	case k.Key().Code == tea.KeyEsc:
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	case isUp(k):
		s.cursor = (s.cursor + 12) % 13
	case isDown(k) || k.Key().Code == tea.KeyTab:
		s.cursor = (s.cursor + 1) % 13
	case isLeft(k):
		return Action{}, s.change(-1)
	case isRight(k) || k.Key().Code == tea.KeyEnter:
		if s.cursor == 12 {
			s.confirm = true
		} else {
			return Action{}, s.change(1)
		}
	}
	return Action{}, nil
}
func (s *settingsScreen) change(d int) tea.Cmd {
	v := s.c.settings()
	switch s.cursor {
	case 0:
		if v.UILanguage == "en" {
			v.UILanguage = "ru"
		} else {
			v.UILanguage = "en"
		}
	case 1:
		var languages []string
		for id := range s.c.service.Profiles() {
			languages = append(languages, id)
		}
		sort.Strings(languages)
		for i, id := range languages {
			if id == v.Language {
				v.Language = languages[(i+d+len(languages))%len(languages)]
				break
			}
		}
	case 2:
		v.TargetWPM = max(10, min(150, v.TargetWPM+float64(d*5)))
	case 3:
		v.Accuracy = max(.80, min(1, v.Accuracy+float64(d)*.01))
	case 4:
		v.LessonRunes = max(50, min(500, v.LessonRunes+d*10))
	case 5:
		v.TrainNumbers = !v.TrainNumbers
	case 6:
		v.TrainUppercase = !v.TrainUppercase
	case 7:
		v.TrainPunctuation = !v.TrainPunctuation
	case 8:
		v.ShowKeyboard = !v.ShowKeyboard
	case 9:
		v.ColorTheme = nextColorTheme(v.ColorTheme, d)
	case 10:
		v.Position = nextPosition(v.Position, d)
	case 11:
		v.DailyGoalMinutes = max(1, min(120, v.DailyGoalMinutes+d))
	}
	return s.c.work(func() error { return s.c.service.SaveSettings(v) }, func(err error) tea.Cmd {
		if err != nil {
			s.c.setStoreError(err)
			return nil
		}
		_ = s.c.localizer.SetLanguage(s.c.settings().UILanguage)
		s.c.applyTheme(s.c.settings().ColorTheme)
		s.c.status.Clear()
		return nil
	})
}

func nextPosition(current domain.InterfacePosition, direction int) domain.InterfacePosition {
	positions := []domain.InterfacePosition{
		domain.PositionTopLeft, domain.PositionTopCenter, domain.PositionTopRight,
		domain.PositionCenterLeft, domain.PositionCenter, domain.PositionCenterRight,
		domain.PositionBottomLeft, domain.PositionBottomCenter, domain.PositionBottomRight,
	}
	index := 4
	for i, position := range positions {
		if position == current {
			index = i
			break
		}
	}
	return positions[(index+direction+len(positions))%len(positions)]
}

func nextColorTheme(current domain.ColorTheme, direction int) domain.ColorTheme {
	themes := []domain.ColorTheme{domain.ThemeViolet, domain.ThemeOcean, domain.ThemeSunset, domain.ThemeMono}
	index := 0
	for i, theme := range themes {
		if theme == current {
			index = i
			break
		}
	}
	index = (index + direction + len(themes)) % len(themes)
	return themes[index]
}

func (s *settingsScreen) themeName(theme domain.ColorTheme) string {
	ids := map[domain.ColorTheme]i18n.MessageID{domain.ThemeViolet: i18n.ThemeViolet, domain.ThemeOcean: i18n.ThemeOcean, domain.ThemeSunset: i18n.ThemeSunset, domain.ThemeMono: i18n.ThemeMono}
	id, ok := ids[theme]
	if !ok {
		id = i18n.ThemeViolet
	}
	return s.c.t(id, nil)
}
func (s *settingsScreen) positionName(position domain.InterfacePosition) string {
	ids := map[domain.InterfacePosition]i18n.MessageID{
		domain.PositionTopLeft: i18n.PositionTopLeft, domain.PositionTopCenter: i18n.PositionTopCenter, domain.PositionTopRight: i18n.PositionTopRight,
		domain.PositionCenterLeft: i18n.PositionCenterLeft, domain.PositionCenter: i18n.PositionCenter, domain.PositionCenterRight: i18n.PositionCenterRight,
		domain.PositionBottomLeft: i18n.PositionBottomLeft, domain.PositionBottomCenter: i18n.PositionBottomCenter, domain.PositionBottomRight: i18n.PositionBottomRight,
	}
	id, ok := ids[position]
	if !ok {
		id = i18n.PositionCenter
	}
	return s.c.t(id, nil)
}
func (s *settingsScreen) View() string {
	v := s.c.settings()
	profile := s.c.service.Profile()
	profileName := profile.Name
	if profileName == "" {
		profileName = s.c.t(i18n.MessageID(profile.NameID), nil)
	}
	yesno := s.c.t(i18n.BoolNo, nil)
	if v.ShowKeyboard {
		yesno = s.c.t(i18n.BoolYes, nil)
	}
	length := s.c.localizer.Plural(i18n.CountCharacters, v.LessonRunes, map[string]any{"Count": v.LessonRunes})
	boolValue := func(enabled bool) string {
		if enabled {
			return s.c.t(i18n.BoolYes, nil)
		}
		return s.c.t(i18n.BoolNo, nil)
	}
	vals := []string{
		s.c.t(i18n.SettingsUI, map[string]any{"Value": strings.ToUpper(v.UILanguage)}),
		s.c.t(i18n.SettingsTraining, map[string]any{"Value": profileName}),
		s.c.t(i18n.SettingsSpeed, map[string]any{"Value": fmt.Sprintf("%.0f", v.TargetWPM)}),
		s.c.t(i18n.SettingsAccuracy, map[string]any{"Value": fmt.Sprintf("%.0f", v.Accuracy*100)}),
		s.c.t(i18n.SettingsLength, map[string]any{"Value": length}),
		s.c.t(i18n.SettingsNumbers, map[string]any{"Value": boolValue(v.TrainNumbers)}),
		s.c.t(i18n.SettingsUppercase, map[string]any{"Value": boolValue(v.TrainUppercase)}),
		s.c.t(i18n.SettingsPunctuation, map[string]any{"Value": boolValue(v.TrainPunctuation)}),
		s.c.t(i18n.SettingsKeyboard, map[string]any{"Value": yesno}),
		s.c.t(i18n.SettingsTheme, map[string]any{"Value": s.themeName(v.ColorTheme)}),
		s.c.t(i18n.SettingsPosition, map[string]any{"Value": s.positionName(v.Position)}),
		s.c.t("settings.daily_goal", map[string]any{"Value": v.DailyGoalMinutes}),
		s.c.t(i18n.SettingsReset, nil),
	}
	var b strings.Builder
	b.WriteString(s.c.theme.Title.Render(s.c.t(i18n.Settings, nil)))
	b.WriteString("\n\n")
	for i, line := range vals {
		prefix := "  "
		if i == s.cursor {
			prefix = "› "
			line = s.c.theme.Title.Render(line)
		}
		b.WriteString(prefix + line + "\n")
	}
	if s.confirm {
		b.WriteString("\n" + s.c.theme.Error.Render(s.c.t(i18n.SettingsConfirm, nil)))
	}
	b.WriteString("\n" + Hotkeys(s.c, i18n.HotkeySettings))
	return b.String()
}
