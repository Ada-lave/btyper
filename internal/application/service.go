package application

import (
	"errors"
	"io"
	"maps"
	"math"
	"os"
	"sort"
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
	if v.DailyGoalMinutes < 1 || v.DailyGoalMinutes > 120 {
		v.DailyGoalMinutes = d.DailyGoalMinutes
	}
	if v.Language == "" {
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
	if provider, ok := store.(interface {
		UserProfiles() (map[string]domain.LanguageProfile, error)
	}); ok {
		userProfiles, err := provider.UserProfiles()
		if err != nil {
			return nil, err
		}
		for id, profile := range userProfiles {
			profiles[id] = profile
		}
	}
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

func (s *LessonService) StartDrill(pattern string, now time.Time) (*trainer.Engine, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.profiles[s.settings.Language]
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	var target domain.Skill
	rs := []rune(pattern)
	if len(rs) == 1 || len(rs) == 2 {
		allowed := map[rune]bool{}
		for _, r := range p.UnlockOrder {
			allowed[r] = true
		}
		valid := true
		for _, r := range rs {
			valid = valid && allowed[r]
		}
		if valid {
			kind := domain.SkillRune
			if len(rs) == 2 {
				kind = domain.SkillBigram
			}
			target = domain.Skill{Language: p.ID, Kind: kind, Pattern: pattern}
		}
	}
	if target.Pattern == "" {
		return nil, errors.New("choose a letter or bigram from the current language")
	}
	unlocked := trainer.InitialUnlocked(p)
	for _, skill := range s.skills {
		if skill.Kind == domain.SkillRune && skill.Samples >= 6 {
			if rs := []rune(skill.Pattern); len(rs) == 1 {
				unlocked[rs[0]] = true
			}
		}
	}
	for _, r := range target.Pattern {
		unlocked[r] = true
	}
	weak := make(map[rune]float64, len(s.progress))
	for r, v := range s.progress {
		weak[r] = v.Confidence
	}
	text := trainer.NewGenerator(now.UnixNano()).AdaptiveLesson(p, unlocked, target, weak, s.settings.LessonRunes)
	return trainer.NewFocusedEngine(text, domain.ModeDrill, p.ID, target.Pattern, time.Time{}), nil
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

func (s *LessonService) ProgressReview(now time.Time) (domain.ProgressReview, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var review domain.ProgressReview
	if store, ok := s.store.(domain.ReviewStore); ok {
		var err error
		review, err = store.ProgressReview(now, s.settings.Language)
		if err != nil {
			return review, err
		}
	}
	for _, skill := range s.skills {
		if skill.Samples == 0 {
			continue
		}
		review.ObservedSkills++
		if skill.Level >= 3 {
			review.StableSkills++
		}
		if !skill.DueAt.IsZero() && !skill.DueAt.After(now) {
			review.DueSkills++
		}
	}
	return review, nil
}

func (s *LessonService) SavePracticeTime(entries []domain.PracticeTime) error {
	return s.store.SavePracticeTime(entries)
}
func (s *LessonService) PracticeTime(day string) (time.Duration, error) {
	return s.store.PracticeTime(day)
}

func (s *LessonService) PracticeDays() (map[string]time.Duration, error) {
	if daily, ok := s.store.(domain.DailyStore); ok {
		return daily.PracticeDays()
	}
	return map[string]time.Duration{}, nil
}

type GoalStatus struct {
	Today     time.Duration
	Goal      time.Duration
	Completed bool
	Streak    int
}

func DailyGoalStatus(now time.Time, days map[string]time.Duration, minutes int) GoalStatus {
	goal := time.Duration(minutes) * time.Minute
	today := days[now.Format("2006-01-02")]
	completed := today >= goal
	streak := 0
	year, month, date := now.Date()
	day := time.Date(year, month, date, 12, 0, 0, 0, now.Location())
	if !completed {
		day = day.AddDate(0, 0, -1)
	}
	for days[day.Format("2006-01-02")] >= goal {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return GoalStatus{Today: today, Goal: goal, Completed: completed, Streak: streak}
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

type TextSkillPreview struct {
	Kind        domain.SkillKind
	Pattern     string
	Occurrences int
	Confidence  float64
}

type TextAnalysis struct {
	Frequent  []TextSkillPreview
	Difficult []TextSkillPreview
}

func (s *LessonService) AnalyzeCustomText(raw string) (TextAnalysis, error) {
	text, err := NormalizeCustomText(raw)
	if err != nil {
		return TextAnalysis{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	candidates := map[string]bool{}
	profile := s.profiles[s.settings.Language]
	allowedLetters := map[rune]bool{}
	for _, r := range profile.UnlockOrder {
		allowedLetters[r] = true
	}
	for _, skill := range trainer.CandidateSkills(profile) {
		candidates[trainer.SkillKey(skill.Kind, skill.Pattern)] = true
	}
	counts := map[string]*TextSkillPreview{}
	add := func(kind domain.SkillKind, pattern string) {
		key := trainer.SkillKey(kind, pattern)
		if !candidates[key] {
			return
		}
		if counts[key] == nil {
			counts[key] = &TextSkillPreview{Kind: kind, Pattern: pattern, Confidence: s.skills[key].Confidence}
		}
		counts[key].Occurrences++
	}
	for i, r := range text {
		if kind := trainer.SkillKindForPattern(string(r)); kind != "" {
			add(kind, string(r))
		}
		if i > 0 && allowedLetters[text[i-1]] && allowedLetters[r] {
			candidates[trainer.SkillKey(domain.SkillBigram, string([]rune{text[i-1], r}))] = true
			add(domain.SkillBigram, string([]rune{text[i-1], r}))
		}
	}
	var all []TextSkillPreview
	for _, value := range counts {
		all = append(all, *value)
	}
	frequent := append([]TextSkillPreview(nil), all...)
	sort.Slice(frequent, func(i, j int) bool {
		if frequent[i].Occurrences != frequent[j].Occurrences {
			return frequent[i].Occurrences > frequent[j].Occurrences
		}
		return frequent[i].Pattern < frequent[j].Pattern
	})
	difficult := append([]TextSkillPreview(nil), all...)
	sort.Slice(difficult, func(i, j int) bool {
		left := float64(difficult[i].Occurrences) * (1 - difficult[i].Confidence)
		right := float64(difficult[j].Occurrences) * (1 - difficult[j].Confidence)
		if left != right {
			return left > right
		}
		return difficult[i].Pattern < difficult[j].Pattern
	})
	return TextAnalysis{Frequent: frequent[:min(5, len(frequent))], Difficult: difficult[:min(5, len(difficult))]}, nil
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
