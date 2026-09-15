package ui

import "charm.land/lipgloss/v2"

type Theme struct{ Title, Text, Muted, Success, Error, Selection, Border lipgloss.Style }

func NewTheme(dark bool) Theme {
	ld := lipgloss.LightDark(dark)
	fg := ld(lipgloss.Color("#202020"), lipgloss.Color("#E6E6E6"))
	muted := ld(lipgloss.Color("#666666"), lipgloss.Color("#888888"))
	accent := ld(lipgloss.Color("#5B21B6"), lipgloss.Color("#A78BFA"))
	return Theme{Title: lipgloss.NewStyle().Bold(true).Foreground(accent), Text: lipgloss.NewStyle().Foreground(fg), Muted: lipgloss.NewStyle().Foreground(muted), Success: lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")), Error: lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Underline(true), Selection: lipgloss.NewStyle().Foreground(fg).Reverse(true), Border: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(muted).Padding(1, 2)}
}
