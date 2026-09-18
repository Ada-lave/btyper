package domain

import "time"

type Mode string
type ColorTheme string

const (
	ModeLearn   Mode = "learn"
	ModeImprove Mode = "improve"
	ModeText    Mode = "text"
)

const (
	ThemeViolet ColorTheme = "violet"
	ThemeOcean  ColorTheme = "ocean"
	ThemeSunset ColorTheme = "sunset"
	ThemeMono   ColorTheme = "mono"
)

type Settings struct {
	UILanguage   string
	Language     string
	Mode         Mode
	TargetWPM    float64
	Accuracy     float64
	LessonRunes  int
	ShowKeyboard bool
	ColorTheme   ColorTheme
}

func DefaultSettings() Settings {
	return Settings{UILanguage: "", Language: "en", Mode: ModeLearn, TargetWPM: 35, Accuracy: 0.95, LessonRunes: 140, ShowKeyboard: true, ColorTheme: ThemeViolet}
}

type LanguageProfile struct {
	ID, NameID  string
	UnlockOrder []rune
	Rows        []string
	Finger      map[rune]string
	Words       []string
}

type CharacterProgress struct {
	Unlocked, Mastered              bool
	Language                        string
	Rune                            rune
	Samples, Errors                 int
	LatencyMS, Accuracy, Confidence float64
	MasteryStreak                   int
}

type CharacterStat struct {
	LatencySamples  int
	Rune            rune
	Samples, Errors int
	LatencyMS       float64
}

type SessionResult struct {
	AttemptID                              string
	ID                                     int64
	StartedAt                              time.Time
	Mode                                   Mode
	Language                               string
	TargetRune                             rune
	Text                                   string
	Duration                               time.Duration
	Correct, Attempts, Errors, Corrections int
	WPM, CPM, Accuracy                     float64
	Chars                                  map[rune]*CharacterStat
}

type HistoryEntry struct {
	ID            int64
	StartedAt     time.Time
	Mode          Mode
	Language      string
	WPM, Accuracy float64
	Errors        int
	Duration      time.Duration
}

type HistoryFilter struct {
	Language      string
	Mode          Mode
	Since         time.Time
	Limit, Offset int
}

type HistorySummary struct {
	Sessions      int
	Duration      time.Duration
	WPM, Accuracy float64
}

type PracticeTime struct {
	AttemptID, Day string
	Duration       time.Duration
}

type Store interface {
	LoadSettings() (Settings, error)
	SaveSettings(Settings) error
	LoadProgress(language string) (map[rune]CharacterProgress, error)
	SaveSession(SessionResult, map[rune]CharacterProgress) error
	History(HistoryFilter) ([]HistoryEntry, error)
	Summary(HistoryFilter) (HistorySummary, error)
	SavePracticeTime([]PracticeTime) error
	PracticeTime(day string) (time.Duration, error)
	Reset() error
	Close() error
}
