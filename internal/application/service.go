package application

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"btyper/internal/domain"
	"btyper/internal/trainer"
	"golang.org/x/text/unicode/norm"
)

var (
	ErrInvalidUTF8 = errors.New("invalid UTF-8")
	ErrEmptyText   = errors.New("empty text")
)

type LessonService struct {
	store        domain.Store
	profiles     map[string]domain.LanguageProfile
	settings     domain.Settings
	progress     map[rune]domain.CharacterProgress
	customText   []rune
	customOffset int
}

func LoadSettings(store domain.Store) (domain.Settings, error) { return store.LoadSettings() }

func NewLessonService(store domain.Store, settings domain.Settings) (*LessonService, error) {
	profiles := trainer.Profiles()
	if _, ok := profiles[settings.Language]; !ok {
		settings.Language = "en"
	}
	progress, err := store.LoadProgress(settings.Language)
	if err != nil {
		return nil, err
	}
	return &LessonService{store: store, profiles: profiles, settings: settings, progress: progress}, nil
}

func (s *LessonService) Settings() domain.Settings                   { return s.settings }
func (s *LessonService) Profiles() map[string]domain.LanguageProfile { return s.profiles }
func (s *LessonService) Profile() domain.LanguageProfile             { return s.profiles[s.settings.Language] }
func (s *LessonService) Progress() map[rune]domain.CharacterProgress { return s.progress }

func (s *LessonService) SaveSettings(v domain.Settings) error {
	if _, ok := s.profiles[v.Language]; !ok {
		v.Language = "en"
	}
	changed := v.Language != s.settings.Language
	s.settings = v
	if changed {
		p, err := s.store.LoadProgress(v.Language)
		if err != nil {
			return err
		}
		s.progress = p
	}
	return s.store.SaveSettings(v)
}

func (s *LessonService) StartAdaptive(mode domain.Mode, now time.Time) *trainer.Engine {
	s.settings.Mode = mode
	p := s.Profile()
	improve := mode == domain.ModeImprove
	unlocked, target := trainer.LearningState(p, s.progress, improve)
	weak := make(map[rune]float64, len(s.progress))
	for r, v := range s.progress {
		weak[r] = v.Confidence
	}
	text := ""
	if improve && s.needsCalibration(p) {
		text = s.calibrationText(p)
	} else {
		text = trainer.NewGenerator(now.UnixNano()).Lesson(p, unlocked, target, weak, s.settings.LessonRunes, mode)
	}
	return trainer.NewEngine(text, mode, s.settings.Language, target, time.Time{})
}

func (s *LessonService) needsCalibration(p domain.LanguageProfile) bool {
	for _, r := range p.UnlockOrder {
		if s.progress[r].Samples < 12 {
			return true
		}
	}
	return false
}

func (s *LessonService) calibrationText(p domain.LanguageProfile) string {
	var b strings.Builder
	for len([]rune(b.String())) < s.settings.LessonRunes {
		for i, r := range p.UnlockOrder {
			b.WriteRune(r)
			if i%6 == 5 {
				b.WriteByte(' ')
			}
		}
	}
	rs := []rune(b.String())
	if len(rs) > s.settings.LessonRunes {
		rs = rs[:s.settings.LessonRunes]
	}
	return strings.TrimSpace(string(rs))
}

func (s *LessonService) Complete(engine *trainer.Engine, now time.Time) (domain.SessionResult, error) {
	r := engine.Finish(now)
	s.progress = trainer.UpdateProgress(s.progress, r, s.settings)
	return r, s.store.SaveSession(r, s.progress)
}

func (s *LessonService) History(limit int) ([]domain.HistoryEntry, error) {
	return s.store.History(limit)
}

func (s *LessonService) Reset() error {
	if err := s.store.Reset(); err != nil {
		return err
	}
	s.progress = map[rune]domain.CharacterProgress{}
	return nil
}

func NormalizeCustomText(raw string) ([]rune, error) {
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
