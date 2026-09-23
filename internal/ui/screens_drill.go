package ui

import (
	"strings"
	"time"

	"btyper/internal/domain"
	"btyper/internal/trainer"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type drillScreen struct {
	c     *Context
	input textinput.Model
}

func newDrillScreen(c *Context) Screen {
	input := textinput.New()
	input.Placeholder = c.t("drill.placeholder", nil)
	input.CharLimit = 2
	input.SetWidth(16)
	return &drillScreen{c: c, input: input}
}

func (s *drillScreen) Activate() tea.Cmd { return s.input.Focus() }
func (s *drillScreen) Resize(int, int)   {}

func (s *drillScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if k.Key().Code == tea.KeyEsc {
			return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
		}
		if k.Key().Code == tea.KeyEnter {
			engine, err := s.c.service.StartDrill(s.input.Value(), time.Now())
			if err != nil {
				s.c.status.Set(s.c.t("drill.invalid", nil))
				return Action{}, nil
			}
			s.c.engine = engine
			s.c.status.Clear()
			return Action{Kind: ActionNavigate, Route: RoutePractice}, nil
		}
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return Action{}, cmd
}

func (s *drillScreen) View() string {
	profile := s.c.service.Profile()
	var pairs []string
	for _, candidate := range trainer.CandidateSkills(profile) {
		if candidate.Kind == domain.SkillBigram {
			pairs = append(pairs, candidate.Pattern)
			if len(pairs) == 8 {
				break
			}
		}
	}
	return s.c.theme.Title.Render(s.c.t("drill.title", nil)) + "\n\n" +
		s.c.t("drill.instructions", map[string]any{"Pairs": strings.Join(pairs, ", ")}) + "\n\n" +
		s.input.View() + "\n\n" + s.c.t("drill.hotkeys", nil)
}
