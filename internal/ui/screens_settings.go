package ui

import (
	"fmt"
	"strings"
	"time"

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
			if err := s.c.service.Reset(); err != nil {
				s.c.setStoreError(err)
			} else {
				s.c.status.SetFor(s.c.t(i18n.SettingsResetDone, nil), time.Now().Add(2*time.Second))
			}
		}
		s.confirm = false
		return Action{}, nil
	}
	switch {
	case k.Key().Code == tea.KeyEsc:
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	case isUp(k):
		s.cursor = (s.cursor + 6) % 7
	case isDown(k) || k.Key().Code == tea.KeyTab:
		s.cursor = (s.cursor + 1) % 7
	case isLeft(k):
		s.change(-1)
	case isRight(k) || k.Key().Code == tea.KeyEnter:
		if s.cursor == 6 {
			s.confirm = true
		} else {
			s.change(1)
		}
	}
	return Action{}, nil
}
func (s *settingsScreen) change(d int) {
	v := s.c.settings()
	switch s.cursor {
	case 0:
		if v.UILanguage == "en" {
			v.UILanguage = "ru"
		} else {
			v.UILanguage = "en"
		}
		_ = s.c.localizer.SetLanguage(v.UILanguage)
		v.UILanguage = s.c.localizer.Language()
	case 1:
		if v.Language == "en" {
			v.Language = "ru"
		} else {
			v.Language = "en"
		}
	case 2:
		v.TargetWPM = max(10, min(150, v.TargetWPM+float64(d*5)))
	case 3:
		v.Accuracy = max(.80, min(1, v.Accuracy+float64(d)*.01))
	case 4:
		v.LessonRunes = max(50, min(500, v.LessonRunes+d*10))
	case 5:
		v.ShowKeyboard = !v.ShowKeyboard
	}
	s.c.setStoreError(s.c.service.SaveSettings(v))
}
func (s *settingsScreen) View() string {
	v := s.c.settings()
	profile := s.c.service.Profile()
	yesno := s.c.t(i18n.BoolNo, nil)
	if v.ShowKeyboard {
		yesno = s.c.t(i18n.BoolYes, nil)
	}
	length := s.c.localizer.Plural(i18n.CountCharacters, v.LessonRunes, map[string]any{"Count": v.LessonRunes})
	vals := []string{s.c.t(i18n.SettingsUI, map[string]any{"Value": strings.ToUpper(v.UILanguage)}), s.c.t(i18n.SettingsTraining, map[string]any{"Value": s.c.t(i18n.MessageID(profile.NameID), nil)}), s.c.t(i18n.SettingsSpeed, map[string]any{"Value": fmt.Sprintf("%.0f", v.TargetWPM)}), s.c.t(i18n.SettingsAccuracy, map[string]any{"Value": fmt.Sprintf("%.0f", v.Accuracy*100)}), s.c.t(i18n.SettingsLength, map[string]any{"Value": length}), s.c.t(i18n.SettingsKeyboard, map[string]any{"Value": yesno}), s.c.t(i18n.SettingsReset, nil)}
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
