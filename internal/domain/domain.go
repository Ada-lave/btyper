package domain

import "time"

type Mode string
type ColorTheme string
type InterfacePosition string

const (
	ModeAdaptive Mode = "adaptive"
	// ModeLearn and ModeImprove are retained for reading historical sessions.
	ModeLearn   Mode = "learn"
	ModeImprove Mode = "improve"
	ModeText    Mode = "text"
)

const (
	PositionTopLeft      InterfacePosition = "top-left"
	PositionTopCenter    InterfacePosition = "top-center"
	PositionTopRight     InterfacePosition = "top-right"
	PositionCenterLeft   InterfacePosition = "center-left"
	PositionCenter       InterfacePosition = "center"
	PositionCenterRight  InterfacePosition = "center-right"
	PositionBottomLeft   InterfacePosition = "bottom-left"
	PositionBottomCenter InterfacePosition = "bottom-center"
	PositionBottomRight  InterfacePosition = "bottom-right"
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
	Position     InterfacePosition
}

func DefaultSettings() Settings {
	return Settings{UILanguage: "", Language: "en", Mode: ModeAdaptive, TargetWPM: 35, Accuracy: 0.95, LessonRunes: 140, ShowKeyboard: true, ColorTheme: ThemeViolet, Position: PositionCenter}
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

type SkillKind string

const (
	SkillRune        SkillKind = "rune"
	SkillBigram      SkillKind = "bigram"
	SkillNumber      SkillKind = "number"
	SkillUppercase   SkillKind = "uppercase"
	SkillPunctuation SkillKind = "punctuation"
)

// Skill is the durable state used by the adaptive scheduler. Pattern contains
// one rune for single-key skills and two runes for SkillBigram.
type Skill struct {
	Language, Pattern               string
	Kind                            SkillKind
	Samples, Errors, LatencySamples int
	LatencyMS, Accuracy, Confidence float64
	Level                           int
	LastPracticed, DueAt            time.Time
}

type SkillStat struct {
	Kind                            SkillKind
	Pattern                         string
	Samples, Errors, LatencySamples int
	LatencyMS                       float64
}

type SessionResult struct {
	AttemptID                              string
	ID                                     int64
	StartedAt                              time.Time
	Mode                                   Mode
	Language                               string
	TargetRune                             rune
	TargetSkill                            string
	Text                                   string
	Duration                               time.Duration
	Correct, Attempts, Errors, Corrections int
	WPM, CPM, Accuracy                     float64
	Chars                                  map[rune]*CharacterStat
	Skills                                 map[string]*SkillStat
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

type TrendPoint struct {
	Day                    string
	Duration               time.Duration
	Sessions               int
	WPM, Accuracy, Latency float64
}

type AdaptiveStore interface {
	LoadSkills(language string) (map[string]Skill, error)
	SaveAdaptiveSession(SessionResult, map[rune]CharacterProgress, map[string]Skill) error
	Trends(language string, since time.Time) ([]TrendPoint, error)
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
