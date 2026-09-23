package application

import (
	"btyper/internal/domain"
	"btyper/internal/storage"
	"btyper/internal/trainer"
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
