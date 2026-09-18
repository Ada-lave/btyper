package ui

import (
	"btyper/internal/domain"
	"fmt"
	"strings"
	"time"

	"btyper/internal/i18n"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

type statisticsScreen struct {
	c                                     *Context
	tab                                   int
	table                                 StatisticsTable
	filter                                domain.HistoryFilter
	languageIndex, modeIndex, periodIndex int
	summary                               domain.HistorySummary
}

func newStatisticsScreen(c *Context) Screen {
	s := &statisticsScreen{c: c, filter: domain.HistoryFilter{Limit: 50}}
	return s
}
func (s *statisticsScreen) Activate() tea.Cmd { return s.loadHistory() }
func (s *statisticsScreen) Resize(w, h int) {
	s.table.Model.SetWidth(max(40, w))
	s.table.Model.SetHeight(max(5, min(16, h-9)))
}
func (s *statisticsScreen) Update(msg tea.Msg) (Action, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok && s.tab == 0 {
		changed := true
		switch {
		case isPlainKey(k, 'l'):
			s.languageIndex = (s.languageIndex + 1) % 3
			s.filter.Language = []string{"", "en", "ru"}[s.languageIndex]
			s.filter.Offset = 0
		case isPlainKey(k, 'm'):
			s.modeIndex = (s.modeIndex + 1) % 4
			s.filter.Mode = []domain.Mode{"", domain.ModeLearn, domain.ModeImprove, domain.ModeText}[s.modeIndex]
			s.filter.Offset = 0
		case isPlainKey(k, 'p'):
			s.periodIndex = (s.periodIndex + 1) % 3
			s.filter.Since = time.Time{}
			if s.periodIndex > 0 {
				days := []int{0, 7, 30}[s.periodIndex]
				s.filter.Since = dayStart(s.c.now).AddDate(0, 0, -days+1)
			}
			s.filter.Offset = 0
		case k.Key().Code == tea.KeyRight:
			if s.filter.Offset+50 < s.summary.Sessions {
				s.filter.Offset += 50
			}
		case k.Key().Code == tea.KeyLeft:
			s.filter.Offset = max(0, s.filter.Offset-50)
		default:
			changed = false
		}
		if changed {
			return Action{}, s.loadHistory()
		}
	}
	if k, ok := msg.(tea.KeyPressMsg); ok && (k.String() == "tab" || k.String() == "1" || k.String() == "2") {
		if k.String() == "1" {
			s.tab = 0
		} else if k.String() == "2" {
			s.tab = 1
		} else {
			s.tab = (s.tab + 1) % 2
		}
		if s.tab == 0 {
			return Action{}, s.loadHistory()
		} else {
			s.rebuildKeys()
			s.Resize(s.c.width, s.c.height)
		}
		return Action{}, nil
	}
	if isBack(msg) {
		return Action{Kind: ActionNavigate, Route: RouteMenu}, nil
	}
	if k, ok := msg.(tea.KeyPressMsg); ok {
		if isUp(k) {
			s.table.Model.MoveUp(1)
			return Action{}, nil
		}
		if isDown(k) {
			s.table.Model.MoveDown(1)
			return Action{}, nil
		}
	}
	var cmd tea.Cmd
	s.table.Model, cmd = s.table.Model.Update(msg)
	return Action{}, cmd
}
func (s *statisticsScreen) loadHistory() tea.Cmd {
	var history []domain.HistoryEntry
	var summary domain.HistorySummary
	filter := s.filter
	return s.c.work(func() error {
		var err error
		history, err = s.c.service.History(filter)
		if err != nil {
			return err
		}
		summary, err = s.c.service.Summary(filter)
		return err
	}, func(err error) tea.Cmd {
		if err != nil {
			s.c.setStoreError(err)
			return nil
		}
		s.summary = summary
		s.rebuildHistory(history)
		s.Resize(s.c.width, s.c.height)
		return nil
	})
}

func (s *statisticsScreen) rebuildHistory(h []domain.HistoryEntry) {
	rows := make([]table.Row, 0, len(h))
	for _, v := range h {
		rows = append(rows, table.Row{v.StartedAt.Local().Format("2006-01-02 15:04"), s.c.modeName(v.Mode), strings.ToUpper(v.Language), fmt.Sprintf("%.1f", v.WPM), fmt.Sprintf("%.1f%%", v.Accuracy*100), fmt.Sprint(v.Errors)})
	}
	s.table.Model = table.New(table.WithColumns([]table.Column{{Title: s.c.t(i18n.ColDate, nil), Width: 17}, {Title: s.c.t(i18n.ColMode, nil), Width: 11}, {Title: s.c.t(i18n.ColLang, nil), Width: 6}, {Title: s.c.t(i18n.ColWPM, nil), Width: 8}, {Title: s.c.t(i18n.ColAccuracy, nil), Width: 11}, {Title: s.c.t(i18n.ColErrors, nil), Width: 8}}), table.WithRows(rows), table.WithHeight(12), table.WithFocused(true))
}
func (s *statisticsScreen) rebuildKeys() {
	p := s.c.service.Profile()
	progress := s.c.service.Progress()
	rows := make([]table.Row, 0, len(p.UnlockOrder))
	for _, r := range p.UnlockOrder {
		v := progress[r]
		latency, accuracy := "—", "—"
		if v.Samples > 0 {
			latency = fmt.Sprintf("%.0f ms", v.LatencyMS)
			accuracy = fmt.Sprintf("%.1f%%", v.Accuracy*100)
		}
		rows = append(rows, table.Row{string(r), fmt.Sprint(v.Samples), fmt.Sprint(v.Errors), latency, accuracy, fmt.Sprintf("%.0f%%", v.Confidence*100)})
	}
	s.table.Model = table.New(table.WithColumns([]table.Column{{Title: s.c.t(i18n.ColKey, nil), Width: 8}, {Title: s.c.t(i18n.ColSamples, nil), Width: 10}, {Title: s.c.t(i18n.ColErrors, nil), Width: 9}, {Title: s.c.t(i18n.ColLatency, nil), Width: 12}, {Title: s.c.t(i18n.ColAccuracy, nil), Width: 12}, {Title: s.c.t(i18n.ColConfidence, nil), Width: 13}}), table.WithRows(rows), table.WithHeight(12), table.WithFocused(true))
}
func (s *statisticsScreen) View() string {
	a, b := "[1] "+s.c.t(i18n.HistorySessions, nil), "[2] "+s.c.t(i18n.HistoryKeys, nil)
	if s.tab == 0 {
		a = s.c.theme.Title.Render(a)
	} else {
		b = s.c.theme.Title.Render(b)
	}
	out := s.c.theme.Title.Render(s.c.t(i18n.History, nil)) + "\n" + a + "   " + b + "\n\n"
	if s.tab == 0 {
		language := strings.ToUpper(s.filter.Language)
		if language == "" {
			language = s.c.t("filter.all", nil)
		}
		mode := s.c.t("filter.all", nil)
		if s.filter.Mode != "" {
			mode = s.c.modeName(s.filter.Mode)
		}
		period := []string{s.c.t("filter.all", nil), "7d", "30d"}[s.periodIndex]
		out += s.c.t("history.filters", map[string]any{"Language": language, "Mode": mode, "Period": period}) + "\n"
		out += s.c.t("history.summary", map[string]any{"Count": s.summary.Sessions, "Time": formatDuration(s.summary.Duration), "WPM": fmt.Sprintf("%.1f", s.summary.WPM), "Accuracy": fmt.Sprintf("%.1f", s.summary.Accuracy*100)}) + "\n"
		out += s.c.t("history.page", map[string]any{"Page": s.filter.Offset/50 + 1, "Pages": max(1, (s.summary.Sessions+49)/50)}) + "\n\n"
	}
	if len(s.table.Model.Rows()) == 0 {
		out += s.c.t(i18n.NoHistory, nil)
	} else {
		height := s.c.height
		if s.tab == 0 {
			height -= 4
		}
		out += s.table.View(s.c.theme, height)
	}
	return out + "\n" + s.c.todayView(false) + "\n" + Hotkeys(s.c, i18n.HotkeyHistory)
}
