package application

import (
	"btyper/internal/domain"
	"btyper/internal/storage"
	"btyper/internal/trainer"
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type failingStore struct {
	domain.Store
	failSave, failLoad bool
	saves              int
}

func (s *failingStore) SaveSettings(v domain.Settings) error {
	if s.failSave {
		return errors.New("write failed")
	}
	return s.Store.SaveSettings(v)
}
func (s *failingStore) LoadProgress(lang string) (map[rune]domain.CharacterProgress, error) {
	if s.failLoad {
		return nil, errors.New("read failed")
	}
	return s.Store.LoadProgress(lang)
}
func (s *failingStore) SaveSession(r domain.SessionResult, p map[rune]domain.CharacterProgress) error {
	s.saves++
	if s.failSave {
		return errors.New("write failed")
	}
	return s.Store.SaveSession(r, p)
}

func newTestService(t *testing.T) (*LessonService, *failingStore) {
	t.Helper()
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store := &failingStore{Store: db}
	s, err := NewLessonService(store, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	return s, store
}

func TestSettingsFailurePreservesState(t *testing.T) {
	s, store := newTestService(t)
	before := s.Settings()
	v := before
	v.Language = "ru"
	store.failLoad = true
	if err := s.SaveSettings(v); err == nil || s.Settings() != before {
		t.Fatal("failed read changed settings")
	}
	store.failLoad = false
	store.failSave = true
	if err := s.SaveSettings(v); err == nil || s.Settings() != before {
		t.Fatal("failed write changed settings")
	}
	store.failSave = false
	if err := s.SaveSettings(v); err != nil || s.Settings().Language != "ru" {
		t.Fatal(err)
	}
}

func TestCompleteRetryIsAtomicAndIdempotent(t *testing.T) {
	s, store := newTestService(t)
	e := trainer.NewEngine("ee", domain.ModeLearn, "en", 'e', time.Time{})
	now := time.Now()
	e.Input('e', now)
	e.Input('e', now.Add(time.Second))
	store.failSave = true
	if _, err := s.Complete(e, now.Add(time.Second)); err == nil || s.Progress()['e'].Samples != 0 {
		t.Fatal("failed save changed progress")
	}
	store.failSave = false
	r, err := s.Complete(e, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if r.Duration != time.Second {
		t.Fatal("retry changed duration")
	}
	if _, err = s.Complete(e, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if s.Progress()['e'].Samples != 2 || store.saves != 2 {
		t.Fatal("duplicate progress")
	}
	h, err := s.History(domain.HistoryFilter{})
	if err != nil || len(h) != 1 {
		t.Fatalf("history %v %v", h, err)
	}
}

func TestNormalizationAndFileLimits(t *testing.T) {
	v := domain.DefaultSettings()
	v.TargetWPM = math.NaN()
	v.Accuracy = 0
	v.LessonRunes = -1
	if got := NormalizeSettings(v); got != domain.DefaultSettings() {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		raw  string
		want error
	}{{"", ErrEmptyText}, {"\xff", ErrInvalidUTF8}, {strings.Repeat("a", MaxTextBytes+1), ErrTextTooLarge}} {
		if _, err := NormalizeCustomText(tc.raw); !errors.Is(err, tc.want) {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "text.txt")
	if err := os.WriteFile(path, []byte("  е\u0308\n test \t text "), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadCustomText(path); err != nil || got != "ё test text" {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := ReadCustomText(filepath.Dir(path)); err == nil {
		t.Fatal("accepted directory")
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x", MaxTextBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadCustomText(path); !errors.Is(err, ErrTextTooLarge) {
		t.Fatal(err)
	}
}

func TestCalibrationOnlyTargetsMissingSamples(t *testing.T) {
	s, _ := newTestService(t)
	for _, r := range s.Profile().UnlockOrder {
		s.progress[r] = domain.CharacterProgress{Samples: 12}
	}
	p := s.progress['e']
	p.Samples = 10
	s.progress['e'] = p
	e := s.StartAdaptive(domain.ModeImprove, time.Now())
	if string(e.Text) != "ee" || s.CalibrationRemaining() != 2 {
		t.Fatalf("%q", e.Text)
	}
}

func TestAdaptiveEngineUsesScheduledSkillAsVisibleTarget(t *testing.T) {
	s, _ := newTestService(t)
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	profile := s.Profile()
	for _, candidate := range trainer.CandidateSkills(profile) {
		if candidate.Kind == domain.SkillRune {
			s.skills[trainer.SkillKey(candidate.Kind, candidate.Pattern)] = domain.Skill{
				Language: profile.ID, Kind: candidate.Kind, Pattern: candidate.Pattern,
				Samples: trainer.RuneFoundationSamples, Confidence: 1, DueAt: now.Add(time.Hour),
			}
		}
	}
	want := trainer.SelectSkill(profile, s.Skills(), now)
	if want.Kind != domain.SkillBigram {
		t.Fatalf("test setup selected %v", want)
	}
	e := s.StartAdaptive(domain.ModeAdaptive, now)
	if e.Result.TargetSkill != want.Pattern || e.Result.TargetRune != 0 || !strings.Contains(string(e.Text), want.Pattern) {
		t.Fatalf("scheduled %q, engine target %q, rune %q, text %q", want.Pattern, e.Result.TargetSkill, e.Result.TargetRune, e.Text)
	}
}

func TestCustomTextPersistsNumberUppercaseAndPunctuationSkills(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s, err := NewLessonService(db, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCustomText("A7!"); err != nil {
		t.Fatal(err)
	}
	e, ok := s.NextCustomLesson()
	if !ok {
		t.Fatal("custom lesson missing")
	}
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	for i, r := range e.Text {
		e.Input(r, now.Add(time.Duration(i)*100*time.Millisecond))
	}
	if _, err := s.Complete(e, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	loaded, err := db.LoadSkills("en")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"uppercase:A", "number:7", "punctuation:!"} {
		if loaded[key].Samples != 1 {
			t.Fatalf("%s not persisted: %v", key, loaded[key])
		}
	}
}

func TestDrillRequiresExplicitTargetAndSavesHistory(t *testing.T) {
	db, err := storage.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s, err := NewLessonService(db, domain.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	before := trainer.SelectSkill(s.Profile(), s.Skills(), now)
	if _, err := s.StartDrill("7", now); err == nil {
		t.Fatal("accepted a skill outside letter and bigram Drill")
	}
	e, err := s.StartDrill("zz", now)
	if err != nil || e.Result.Mode != domain.ModeDrill || e.Result.TargetSkill != "zz" || !strings.Contains(string(e.Text), "zz") {
		t.Fatalf("drill setup failed: %v %v", e, err)
	}
	if got := trainer.SelectSkill(s.Profile(), s.Skills(), now); got != before {
		t.Fatalf("choosing Drill changed adaptive queue: %v -> %v", before, got)
	}
	for i, r := range e.Text {
		e.Input(r, now.Add(time.Duration(i+1)*100*time.Millisecond))
	}
	if _, err := s.Complete(e, now.Add(time.Duration(len(e.Text)+2)*100*time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	history, err := s.History(domain.HistoryFilter{Mode: domain.ModeDrill})
	if err != nil || len(history) != 1 || history[0].Mode != domain.ModeDrill {
		t.Fatalf("Drill missing from history: %v %v", history, err)
	}
}

func TestCustomTextAnalysisReportsFrequentAndWeakSkills(t *testing.T) {
	s, _ := newTestService(t)
	s.skills[trainer.SkillKey(domain.SkillRune, "e")] = domain.Skill{Confidence: .9}
	s.skills[trainer.SkillKey(domain.SkillRune, "n")] = domain.Skill{Confidence: .1}
	analysis, err := s.AnalyzeCustomText("en en en 7!")
	if err != nil {
		t.Fatal(err)
	}
	if len(analysis.Frequent) == 0 || analysis.Frequent[0].Occurrences != 3 {
		t.Fatalf("frequent skills missing: %+v", analysis)
	}
	find := func(items []TextSkillPreview, pattern string) bool {
		for _, item := range items {
			if item.Pattern == pattern {
				return true
			}
		}
		return false
	}
	if !find(analysis.Frequent, "en") || !find(analysis.Difficult, "n") {
		t.Fatalf("text skills missing: %+v", analysis)
	}
}

func TestDailyGoalStatusUsesLocalCalendarDays(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 9, 9, 0, 0, 0, loc)
	days := map[string]time.Duration{
		"2026-03-07": 10 * time.Minute,
		"2026-03-08": 10 * time.Minute,
		"2026-03-09": 9 * time.Minute,
	}
	status := DailyGoalStatus(now, days, 10)
	if status.Completed || status.Streak != 2 {
		t.Fatalf("in-progress day broke streak across DST: %+v", status)
	}
	days["2026-03-09"] = 10 * time.Minute
	status = DailyGoalStatus(now, days, 10)
	if !status.Completed || status.Streak != 3 {
		t.Fatalf("completed day did not extend streak: %+v", status)
	}
	status = DailyGoalStatus(now.AddDate(0, 0, 2), days, 10)
	if status.Completed || status.Streak != 0 {
		t.Fatalf("missed day did not end streak: %+v", status)
	}
}

func TestProgressReviewExplainsSkillRetention(t *testing.T) {
	s, _ := newTestService(t)
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	s.skills["rune:e"] = domain.Skill{Samples: 30, Level: 3, DueAt: now.Add(time.Hour)}
	s.skills["rune:n"] = domain.Skill{Samples: 10, Level: 1, DueAt: now.Add(-time.Hour)}
	review, err := s.ProgressReview(now)
	if err != nil || review.ObservedSkills != 2 || review.StableSkills != 1 || review.DueSkills != 1 {
		t.Fatalf("review %+v: %v", review, err)
	}
}

func TestImportedProfileCanBeSelectedAfterRestart(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	profile := trainer.Profiles()["en"]
	profile.ID, profile.Name, profile.NameID = "en_alt", "English alternative", ""
	data, err := trainer.MarshalProfileJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ImportUserProfile(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
	settings := domain.DefaultSettings()
	settings.Language = "en_alt"
	if err := db.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = storage.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	loaded, err := db.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewLessonService(db, loaded)
	if err != nil || s.Profile().ID != "en_alt" || s.StartAdaptive(domain.ModeAdaptive, time.Now()).Result.Language != "en_alt" {
		t.Fatalf("imported profile cannot be used after restart: %v", err)
	}
}
