package ui

import (
	"strings"
	"time"
	"unicode"

	"btyper/internal/domain"
	"btyper/internal/i18n"
	"btyper/internal/trainer"
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

type Status struct {
	text      string
	expiresAt time.Time
}

func (s *Status) Set(v string)                     { s.text, s.expiresAt = v, time.Time{} }
func (s *Status) SetFor(v string, until time.Time) { s.text, s.expiresAt = v, until }
func (s *Status) Clear()                           { s.text, s.expiresAt = "", time.Time{} }
func (s *Status) ClearExpired(now time.Time) {
	if !s.expiresAt.IsZero() && !now.Before(s.expiresAt) {
		s.Clear()
	}
}
func (s Status) View(t Theme, now time.Time) string {
	if s.text == "" || (!s.expiresAt.IsZero() && !now.Before(s.expiresAt)) {
		return ""
	}
	return t.Error.Render(s.text)
}
func Hotkeys(c *Context, id i18n.MessageID) string { return c.theme.Muted.Render(c.t(id, nil)) }

type LessonRenderer struct{}

func (LessonRenderer) View(e *trainer.Engine, t Theme, width int) string {
	var b strings.Builder
	start := max(0, e.Pos-35)
	end := min(len(e.Text), e.Pos+100)
	if start > 0 {
		b.WriteString(t.Muted.Render("…"))
	}
	for i := start; i < end; i++ {
		r := e.Text[i]
		switch {
		case i < e.Pos:
			b.WriteString(t.Success.Render(string(r)))
		case i == e.Pos && e.Pending != 0:
			b.WriteString(t.Error.Render(string(e.Pending)))
		case i == e.Pos:
			b.WriteString(t.Selection.Render(string(r)))
		default:
			b.WriteRune(r)
		}
	}
	if end < len(e.Text) {
		b.WriteString(t.Muted.Render("…"))
	}
	return lipgloss.NewStyle().Width(max(44, min(94, width-16))).Render(b.String())
}

type Keyboard struct{}

func (Keyboard) View(p domain.LanguageProfile, e *trainer.Engine, c *Context) string {
	expected := rune(0)
	if e.Pos < len(e.Text) {
		expected = unicode.ToLower(e.Text[e.Pos])
	}
	var lines []string
	for i, row := range p.Rows {
		var b strings.Builder
		b.WriteString(strings.Repeat(" ", i*2))
		for _, r := range row {
			s := " " + string(r) + " "
			if r == expected {
				s = c.theme.Selection.Render("[" + string(r) + "]")
			} else {
				s = c.theme.Muted.Render(s)
			}
			b.WriteString(s)
		}
		lines = append(lines, b.String())
	}
	finger := p.Finger[expected]
	if finger != "" {
		ids := map[string]i18n.MessageID{"LP": i18n.FingerLP, "LR": i18n.FingerLR, "LM": i18n.FingerLM, "LI": i18n.FingerLI, "RI": i18n.FingerRI, "RM": i18n.FingerRM, "RR": i18n.FingerRR, "RP": i18n.FingerRP}
		lines = append(lines, "", c.theme.Title.Render(c.t(i18n.FingerHint, map[string]any{"Finger": c.t(ids[finger], nil)})))
	}
	return strings.Join(lines, "\n")
}

type FingerGuide struct{}

func (FingerGuide) View(p domain.LanguageProfile, c *Context) string {
	home := map[string]map[rune]bool{
		"en": {'f': true, 'j': true},
		"ru": {'а': true, 'о': true},
	}[p.ID]
	var lines []string
	for i, row := range p.Rows {
		var b strings.Builder
		b.WriteString(strings.Repeat(" ", i*2))
		for _, r := range row {
			style := c.theme.Finger[p.Finger[r]]
			if home[r] {
				style = style.Bold(true).Underline(true)
			}
			b.WriteString(style.Render("[" + strings.ToUpper(string(r)) + "]"))
		}
		lines = append(lines, b.String())
	}
	keyboard := strings.Join(lines, "\n")
	order := []string{"LP", "LR", "LM", "LI", "RI", "RM", "RR", "RP"}
	ids := fingerMessageIDs()
	legend := make(map[string]string, len(order))
	for _, finger := range order {
		legend[finger] = c.theme.Finger[finger].Render(c.t(ids[finger], nil))
	}
	legendLines := make([]string, 0, 4)
	for i := 0; i < 4; i++ {
		legendLines = append(legendLines, legend[order[i]]+"  ·  "+legend[order[7-i]])
	}
	return keyboard + "\n\n" + strings.Join(legendLines, "\n")
}

func fingerMessageIDs() map[string]i18n.MessageID {
	return map[string]i18n.MessageID{"LP": i18n.FingerLP, "LR": i18n.FingerLR, "LM": i18n.FingerLM, "LI": i18n.FingerLI, "RI": i18n.FingerRI, "RM": i18n.FingerRM, "RR": i18n.FingerRR, "RP": i18n.FingerRP}
}

type StatisticsTable struct{ Model table.Model }

func (s StatisticsTable) View(t Theme, height int) string {
	rows, columns := s.Model.Rows(), s.Model.Columns()
	var b strings.Builder
	write := func(row table.Row, header, selected bool) {
		var cells []string
		for i, col := range columns {
			v := ""
			if i < len(row) {
				v = row[i]
			}
			cell := lipgloss.NewStyle().Width(col.Width).MaxWidth(col.Width).Render(v)
			if header {
				cell = lipgloss.NewStyle().Bold(true).Render(cell)
			}
			cells = append(cells, cell)
		}
		line := lipgloss.JoinHorizontal(lipgloss.Top, cells...)
		if selected {
			line = t.Selection.Render(line)
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	h := make(table.Row, len(columns))
	for i, col := range columns {
		h[i] = col.Title
	}
	write(h, true, false)
	visible := max(4, height-11)
	start := max(0, s.Model.Cursor()-visible+1)
	end := min(len(rows), start+visible)
	for i := start; i < end; i++ {
		write(rows[i], false, i == s.Model.Cursor())
	}
	return strings.TrimRight(b.String(), "\n")
}
