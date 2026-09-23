package ui

import (
	"errors"
	"fmt"
	"strings"

	"btyper/internal/application"
	"btyper/internal/i18n"
	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

type textScreen struct {
	c       *Context
	area    textarea.Model
	preview *application.TextAnalysis
}

func newTextScreen(c *Context, payload any) Screen {
	a := textarea.New()
	a.Placeholder = c.t(i18n.TextEmpty, nil)
	a.ShowLineNumbers = false
	a.CharLimit = application.MaxTextBytes
	a.SetWidth(70)
	a.SetHeight(10)
	if v, ok := payload.(string); ok {
		a.SetValue(v)
	}
	return &textScreen{c: c, area: a}
}
func (s *textScreen) Activate() tea.Cmd { return s.area.Focus() }
func (s *textScreen) Resize(w, h int) {
	s.area.SetWidth(min(90, max(40, w)))
	s.area.SetHeight(max(5, min(14, h-10)))
}
func (s *textScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if s.preview != nil {
			switch {
			case k.Key().Code == tea.KeyEsc:
				s.preview = nil
				return Action{}, nil
			case k.Key().Code == tea.KeyEnter || isCtrlKey(k, 's'):
				if err := s.c.service.SetCustomText(s.area.Value()); err != nil {
					s.c.status.Set(err.Error())
					return Action{}, nil
				}
				s.c.engine, _ = s.c.service.NextCustomLesson()
				s.c.status.Clear()
				return Action{Kind: ActionNavigate, Route: RoutePractice}, nil
			}
			return Action{}, nil
		}
		switch {
		case k.Key().Code == tea.KeyEsc:
			s.area.Blur()
			return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
		case isCtrlKey(k, 'o'):
			s.area.Blur()
			return Action{Kind: ActionNavigate, Route: RoutePicker, Payload: s.area.Value()}, nil
		case isCtrlKey(k, 's'):
			analysis, err := s.c.service.AnalyzeCustomText(s.area.Value())
			if err != nil {
				if errors.Is(err, application.ErrInvalidUTF8) {
					s.c.status.Set(s.c.t(i18n.InvalidUTF8, nil))
				} else if errors.Is(err, application.ErrTextTooLarge) {
					s.c.status.Set(s.c.t("text.too_large", nil))
				} else {
					s.c.status.Set(s.c.t(i18n.TextEmpty, nil))
				}
				return Action{}, nil
			}
			s.preview = &analysis
			s.c.status.Clear()
			return Action{}, nil
		}
	}
	var cmd tea.Cmd
	s.area, cmd = s.area.Update(msg)
	return Action{}, cmd
}
func (s *textScreen) View() string {
	if s.preview != nil {
		format := func(items []application.TextSkillPreview) string {
			var labels []string
			for _, item := range items {
				labels = append(labels, fmt.Sprintf("%s ×%d (%.0f%%)", item.Pattern, item.Occurrences, item.Confidence*100))
			}
			if len(labels) == 0 {
				return "—"
			}
			return strings.Join(labels, ", ")
		}
		return s.c.theme.Title.Render(s.c.t("text.analysis_title", nil)) + "\n\n" +
			s.c.t("text.analysis_frequent", nil) + "\n" + format(s.preview.Frequent) + "\n\n" +
			s.c.t("text.analysis_difficult", nil) + "\n" + format(s.preview.Difficult) + "\n\n" +
			s.c.t("text.analysis_help", nil) + "\n\n" + s.c.t("text.analysis_hotkeys", nil)
	}
	return s.c.theme.Title.Render(s.c.t(i18n.CustomText, nil)) + "\n\n" + s.area.View() + "\n\n" + Hotkeys(s.c, i18n.HotkeyText)
}

type pickerScreen struct {
	c        *Context
	picker   filepicker.Model
	previous string
}

type textLoadedMsg struct{ text string }

func newPickerScreen(c *Context, payload any) Screen {
	p := filepicker.New()
	p.FileAllowed = true
	p.DirAllowed = false
	p.SetHeight(14)
	p.KeyMap.Down.SetKeys("j", "о", "down", "ctrl+n")
	p.KeyMap.Up.SetKeys("k", "л", "up", "ctrl+p")
	p.KeyMap.Back.SetKeys("h", "р", "backspace", "left", "esc")
	p.KeyMap.Open.SetKeys("l", "д", "right", "enter")
	previous, _ := payload.(string)
	return &pickerScreen{c: c, picker: p, previous: previous}
}
func (s *pickerScreen) Activate() tea.Cmd { return s.picker.Init() }
func (s *pickerScreen) Resize(w, h int)   { s.picker.SetHeight(max(5, min(14, h-8))) }
func (s *pickerScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && k.Key().Code == tea.KeyEsc {
		return Action{Kind: ActionNavigate, Route: RouteText, Payload: s.previous}, nil
	}
	var cmd tea.Cmd
	s.picker, cmd = s.picker.Update(msg)
	if ok, path := s.picker.DidSelectFile(msg); ok {
		var text string
		return Action{}, s.c.work(func() error { var err error; text, err = application.ReadCustomText(path); return err }, func(err error) tea.Cmd {
			if err != nil {
				if errors.Is(err, application.ErrTextTooLarge) {
					s.c.status.Set(s.c.t("text.too_large", nil))
				} else {
					s.c.status.Set(s.c.t(i18n.FileError, map[string]any{"Error": err}))
				}
				return nil
			}
			return func() tea.Msg { return textLoadedMsg{text} }
		})
	}
	return Action{}, cmd
}
func (s *pickerScreen) View() string {
	return s.c.theme.Title.Render(s.c.t(i18n.TextChoose, nil)) + "\n\n" + s.picker.View() + "\n\n" + Hotkeys(s.c, i18n.HotkeyBack)
}
