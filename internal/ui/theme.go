package ui

import "charm.land/lipgloss/v2"

type Theme struct {
	Title, Text, Muted, Success, Error, Selection, Border lipgloss.Style
	Finger                                                map[string]lipgloss.Style
	ProgressTone                                          []lipgloss.Style
}

func NewTheme(dark bool) Theme {
	ld := lipgloss.LightDark(dark)
	fg := ld(lipgloss.Color("#202020"), lipgloss.Color("#E6E6E6"))
	muted := ld(lipgloss.Color("#666666"), lipgloss.Color("#888888"))
	accent := ld(lipgloss.Color("#5B21B6"), lipgloss.Color("#A78BFA"))
	colors := map[string]string{"LP": "#EF4444", "LR": "#F97316", "LM": "#EAB308", "LI": "#22C55E", "RI": "#14B8A6", "RM": "#3B82F6", "RR": "#8B5CF6", "RP": "#EC4899"}
	fingers := make(map[string]lipgloss.Style, len(colors))
	for name, color := range colors {
		fingers[name] = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
	progressColors := []string{"#EF4444", "#F0523F", "#F26339", "#F57931", "#F29127", "#EAB308", "#C5B91A", "#9CBD2A", "#70C03B", "#48C34C", "#22C55E"}
	progressTone := make([]lipgloss.Style, len(progressColors))
	for i, color := range progressColors {
		progressTone[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
	return Theme{Title: lipgloss.NewStyle().Bold(true).Foreground(accent), Text: lipgloss.NewStyle().Foreground(fg), Muted: lipgloss.NewStyle().Foreground(muted), Success: lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")), Error: lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Underline(true), Selection: lipgloss.NewStyle().Foreground(fg).Reverse(true), Border: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(muted).Padding(1, 2), Finger: fingers, ProgressTone: progressTone}
}

func (t Theme) Progress(confidence float64) lipgloss.Style {
	confidence = max(0, min(1, confidence))
	index := int(confidence * float64(len(t.ProgressTone)-1))
	return t.ProgressTone[index]
}
