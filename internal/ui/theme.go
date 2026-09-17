package ui

import (
	"btyper/internal/domain"
	"charm.land/lipgloss/v2"
)

type Theme struct {
	Title, Text, Muted, Success, Error, Selection, Border lipgloss.Style
	Finger                                                map[string]lipgloss.Style
	ProgressTone                                          []lipgloss.Style
}

type palette struct {
	accent   string
	progress []string
}

func NewTheme(dark bool) Theme { return NewColorTheme(dark, domain.ThemeViolet) }

func NewColorTheme(dark bool, id domain.ColorTheme) Theme {
	ld := lipgloss.LightDark(dark)
	fg := ld(lipgloss.Color("#202020"), lipgloss.Color("#E6E6E6"))
	muted := ld(lipgloss.Color("#666666"), lipgloss.Color("#888888"))
	palettes := map[domain.ColorTheme]palette{
		domain.ThemeViolet: {"#A78BFA", []string{"#EF4444", "#F0523F", "#F26339", "#F57931", "#F29127", "#EAB308", "#C5B91A", "#9CBD2A", "#70C03B", "#48C34C", "#22C55E"}},
		domain.ThemeOcean:  {"#22D3EE", []string{"#F43F5E", "#E94E70", "#D55E82", "#BD7196", "#9985AA", "#38BDF8", "#35C0EA", "#32C4DC", "#30C8CE", "#2ECFC2", "#2DD4BF"}},
		domain.ThemeSunset: {"#FB923C", []string{"#DC2626", "#E33725", "#E94924", "#EF5A22", "#F56B20", "#F97316", "#FB8818", "#FB9D19", "#FBB21B", "#FBC51D", "#FACC15"}},
		domain.ThemeMono:   {"#D4D4D4", []string{"#525252", "#626262", "#737373", "#838383", "#939393", "#A3A3A3", "#B3B3B3", "#C3C3C3", "#D4D4D4", "#E5E5E5", "#FAFAFA"}},
	}
	p, ok := palettes[id]
	if !ok {
		p = palettes[domain.ThemeViolet]
	}
	accent := lipgloss.Color(p.accent)
	if !dark && id == domain.ThemeMono {
		accent = lipgloss.Color("#404040")
	} else if !dark && id == domain.ThemeViolet {
		accent = lipgloss.Color("#5B21B6")
	}
	colors := map[string]string{"LP": "#EF4444", "LR": "#F97316", "LM": "#EAB308", "LI": "#22C55E", "RI": "#14B8A6", "RM": "#3B82F6", "RR": "#8B5CF6", "RP": "#EC4899"}
	fingers := make(map[string]lipgloss.Style, len(colors))
	for name, color := range colors {
		fingers[name] = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
	progressTone := make([]lipgloss.Style, len(p.progress))
	for i, color := range p.progress {
		progressTone[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(color))
	}
	return Theme{Title: lipgloss.NewStyle().Bold(true).Foreground(accent), Text: lipgloss.NewStyle().Foreground(fg), Muted: lipgloss.NewStyle().Foreground(muted), Success: lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")), Error: lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Underline(true), Selection: lipgloss.NewStyle().Foreground(fg).Reverse(true), Border: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(muted).Padding(1, 2), Finger: fingers, ProgressTone: progressTone}
}

func (t Theme) Progress(confidence float64) lipgloss.Style {
	confidence = max(0, min(1, confidence))
	index := int(confidence * float64(len(t.ProgressTone)-1))
	return t.ProgressTone[index]
}
