package application

import (
	"errors"
	"io"
	"maps"
	"math"
	"os"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"btyper/internal/domain"
	"btyper/internal/trainer"
	"golang.org/x/text/unicode/norm"
)

var (
	ErrInvalidUTF8  = errors.New("invalid UTF-8")
	ErrEmptyText    = errors.New("empty text")
	ErrTextTooLarge = errors.New("text exceeds 1 MiB")
)

const MaxTextBytes = 1 << 20

func ReadCustomText(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("expected a regular text file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err = f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("expected a regular text file")
	}
	b, err := io.ReadAll(io.LimitReader(f, MaxTextBytes+1))
	if err != nil {
		return "", err
	}
	runes, err := NormalizeCustomText(string(b))
	return string(runes), err
}

func NormalizeSettings(v domain.Settings) domain.Settings {
	d := domain.DefaultSettings()
	if math.IsNaN(v.TargetWPM) || math.IsInf(v.TargetWPM, 0) || v.TargetWPM < 10 || v.TargetWPM > 150 {
		v.TargetWPM = d.TargetWPM
	}
	if math.IsNaN(v.Accuracy) || math.IsInf(v.Accuracy, 0) || v.Accuracy < .8 || v.Accuracy > 1 {
		v.Accuracy = d.Accuracy
	}
	if v.LessonRunes < 50 || v.LessonRunes > 500 {
		v.LessonRunes = d.LessonRunes
	}
	if v.Language != "en" && v.Language != "ru" {
		v.Language = d.Language
	}
	if v.UILanguage != "" && v.UILanguage != "en" && v.UILanguage != "ru" {
		v.UILanguage = "en"
	}
	if v.Mode == domain.ModeLearn || v.Mode == domain.ModeImprove {
		v.Mode = domain.ModeAdaptive
	}
	if v.Mode != domain.ModeAdaptive && v.Mode != domain.ModeText {
		v.Mode = d.Mode
	}
	v.ColorTheme = normalizeColorTheme(v.ColorTheme)
	v.Position = normalizePosition(v.Position)
	return v
}

type LessonService struct {
	mu           sync.Mutex
	store        domain.Store
	profiles     map[string]domain.LanguageProfile
	settings     domain.Settings
	progress     map[rune]domain.CharacterProgress
	skills       map[string]domain.Skill
	customText   []rune
	customOffset int
	completed    map[string]bool
}

func LoadSettings(store domain.Store) (domain.Settings, error) { return store.LoadSettings() }

func NewLessonService(store domain.Store, settings domain.Settings) (*LessonService, error) {
	settings = NormalizeSettings(settings)
	profiles := trainer.Profiles()
	if err := trainer.ValidateProfiles(profiles); err != nil {
		return nil, err
	}
	if _, ok := profiles[settings.Language]; !ok {
		settings.Language = "en"
	}
	settings.ColorTheme = normalizeColorTheme(settings.ColorTheme)
	progress, err := store.LoadProgress(settings.Language)
	if err != nil {
		return nil, err
	}
	skills := map[string]domain.Skill{}
	if adaptive, ok := store.(domain.AdaptiveStore); ok {
		skills, err = adaptive.LoadSkills(settings.Language)
		if err != nil {
			return nil, err
		}
	}
	return &LessonService{store: store, profiles: profiles, settings: settings, progress: progress, skills: skills, completed: map[string]bool{}}, nil
}

func (s *LessonService) Settings() domain.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}
func (s *LessonService) Profiles() map[string]domain.LanguageProfile { return s.profiles }
func (s *LessonService) Profile() domain.LanguageProfile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.profiles[s.settings.Language]
}
func (s *LessonService) Progress() map[rune]domain.CharacterProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.progress)
}
func (s *LessonService) Skills() map[string]domain.Skill {
	s.mu.Lock()
	defer s.mu.Unlock()
	return maps.Clone(s.skills)
}

func (s *LessonService) SaveSettings(v domain.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	v = NormalizeSettings(v)
	if _, ok := s.profiles[v.Language]; !ok {
		v.Language = "en"
	}
	v.ColorTheme = normalizeColorTheme(v.ColorTheme)
	changed := v.Language != s.settings.Language
	progress := s.progress
	skills := s.skills
	if changed {
		p, err := s.store.LoadProgress(v.Language)
		if err != nil {
			return err
		}
		progress = p
		if adaptive, ok := s.store.(domain.AdaptiveStore); ok {
			loaded, loadErr := adaptive.LoadSkills(v.Language)
			if loadErr != nil {
				return loadErr
			}
			skills = loaded
		} else {
			skills = map[string]domain.Skill{}
		}
	}
	if err := s.store.SaveSettings(v); err != nil {
		return err
	}
	s.settings, s.progress, s.skills = v, progress, skills
	return nil
}

func normalizeColorTheme(theme domain.ColorTheme) domain.ColorTheme {
	switch theme {
	case domain.ThemeViolet, domain.ThemeOcean, domain.ThemeSunset, domain.ThemeMono:
		return theme
	default:
		return domain.ThemeViolet
	}
}

func normalizePosition(position domain.InterfacePosition) domain.InterfacePosition {
	switch position {
	case domain.PositionTopLeft, domain.PositionTopCenter, domain.PositionTopRight,
		domain.PositionCenterLeft, domain.PositionCenter, domain.PositionCenterRight,
		domain.PositionBottomLeft, domain.PositionBottomCenter, domain.PositionBottomRight:
		return position
	default:
		return domain.PositionCenter
	}
}

func (s *LessonService) StartAdaptive(mode domain.Mode, now time.Time) *trainer.Engine {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.profiles[s.settings.Language]
	if mode == domain.ModeImprove && len(s.skills) == 0 && s.needsLegacyCalibration(p) {
		return trainer.NewEngine(s.legacyCalibrationText(p), mode, s.settings.Language, 0, time.Time{})
	}
	unlocked := trainer.InitialUnlocked(p)
	for _, skill := range s.skills {
		if skill.Kind == domain.SkillRune && skill.Samples >= 6 {
			if rs := []rune(skill.Pattern); len(rs) == 1 {
				unlocked[rs[0]] = true
			}
		}
	}
	target := trainer.SelectSkill(p, s.skills, now)
	for _, r := range target.Pattern {
		unlocked[r] = true
	}
	weak := make(map[rune]float64, len(s.progress))
	for r, v := range s.progress {
		weak[r] = v.Confidence
	}
	text := trainer.NewGenerator(now.UnixNano()).AdaptiveLesson(p, unlocked, target, weak, s.settings.LessonRunes)
	return trainer.NewAdaptiveEngine(text, s.settings.Language, target.Pattern, time.Time{})
}

func (s *LessonService) needsLegacyCalibration(p domain.LanguageProfile) bool {
	for _, r := range p.UnlockOrder {
		if s.progress[r].Samples < 12 {
			return true
		}
	}
	return false
}

func (s *LessonService) legacyCalibrationText(p domain.LanguageProfile) string {
	var b strings.Builder
	for _, r := range p.UnlockOrder {
		for range max(0, 12-s.progress[r].Samples) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (s *LessonService) Complete(engine *trainer.Engine, now time.Time) (domain.SessionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !engine.Done() || engine.StartedAt.IsZero() {
		return domain.SessionResult{}, errors.New("lesson is not complete")
	}
	r := engine.Finish(now)
	if s.completed[r.AttemptID] {
		return r, nil
	}
	progress := trainer.UpdateProgress(s.progress, r, s.settings)
	skills := trainer.UpdateSkills(s.skills, r, s.settings, now)
	var err error
	if adaptive, ok := s.store.(domain.AdaptiveStore); ok {
		err = adaptive.SaveAdaptiveSession(r, progress, skills)
	} else {
		err = s.store.SaveSession(r, progress)
	}
	if err != nil {
		return r, err
	}
	s.progress, s.skills = progress, skills
	s.completed[r.AttemptID] = true
	return r, nil
}

func (s *LessonService) History(filter domain.HistoryFilter) ([]domain.HistoryEntry, error) {
	return s.store.History(filter)
}

func (s *LessonService) Summary(filter domain.HistoryFilter) (domain.HistorySummary, error) {
	return s.store.Summary(filter)
}
func (s *LessonService) Trends(since time.Time) ([]domain.TrendPoint, error) {
	adaptive, ok := s.store.(domain.AdaptiveStore)
	if !ok {
		return nil, nil
	}
	return adaptive.Trends(s.Settings().Language, since)
}

func (s *LessonService) SavePracticeTime(entries []domain.PracticeTime) error {
	return s.store.SavePracticeTime(entries)
}
func (s *LessonService) PracticeTime(day string) (time.Duration, error) {
	return s.store.PracticeTime(day)
}

func (s *LessonService) CalibrationRemaining() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	if len(s.skills) == 0 {
		for _, r := range s.profiles[s.settings.Language].UnlockOrder {
			n += max(0, 12-s.progress[r].Samples)
		}
		return n
	}
	for _, r := range s.profiles[s.settings.Language].UnlockOrder {
		n += max(0, trainer.RuneFoundationSamples-s.skills[trainer.SkillKey(domain.SkillRune, string(r))].Samples)
	}
	return n
}

func (s *LessonService) Reset() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.store.Reset(); err != nil {
		return err
	}
	s.progress = map[rune]domain.CharacterProgress{}
	s.skills = map[string]domain.Skill{}
	s.completed = map[string]bool{}
	return nil
}

func NormalizeCustomText(raw string) ([]rune, error) {
	if len(raw) > MaxTextBytes {
		return nil, ErrTextTooLarge
	}
	if !utf8.ValidString(raw) {
		return nil, ErrInvalidUTF8
	}
	clean := strings.Join(strings.Fields(norm.NFC.String(raw)), " ")
	if clean == "" {
		return nil, ErrEmptyText
	}
	return []rune(clean), nil
}

func CustomChunk(text []rune, offset, limit int) (string, int) {
	offset = max(0, min(offset, len(text)))
	limit = max(1, limit)
	end := min(len(text), offset+limit)
	if end < len(text) {
		for end > offset && !unicode.IsSpace(text[end-1]) {
			end--
		}
		if end == offset {
			end = min(len(text), offset+limit)
		}
	}
	chunk := strings.TrimSpace(string(text[offset:end]))
	for end < len(text) && unicode.IsSpace(text[end]) {
		end++
	}
	return chunk, end
}

func (s *LessonService) SetCustomText(raw string) error {
	text, err := NormalizeCustomText(raw)
	if err != nil {
		return err
	}
	s.customText, s.customOffset = text, 0
	return nil
}

func (s *LessonService) HasMoreCustomText() bool { return s.customOffset < len(s.customText) }
func (s *LessonService) NextCustomLesson() (*trainer.Engine, bool) {
	if !s.HasMoreCustomText() {
		return nil, false
	}
	chunk, next := CustomChunk(s.customText, s.customOffset, s.settings.LessonRunes)
	s.customOffset = next
	return trainer.NewEngine(chunk, domain.ModeText, s.settings.Language, 0, time.Time{}), true
}
