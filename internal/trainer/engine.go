package trainer

import (
	"math"
	"time"
	"unicode"

	"btyper/internal/domain"
)

type Engine struct {
	Text                                  []rune
	Pos                                   int
	Pending                               rune
	StartedAt, LastAccepted, PauseStarted time.Time
	Paused                                time.Duration
	Result                                domain.SessionResult
}

func NewEngine(text string, mode domain.Mode, lang string, target rune, now time.Time) *Engine {
	return &Engine{Text: []rune(text), Result: domain.SessionResult{StartedAt: now, Mode: mode, Language: lang, TargetRune: target, Text: text, Chars: map[rune]*domain.CharacterStat{}}}
}

func (e *Engine) Input(r rune, now time.Time) bool {
	if e.Done() || e.IsPaused() || unicode.IsControl(r) {
		return false
	}
	if e.StartedAt.IsZero() {
		e.StartedAt = now
		e.Result.StartedAt = now
	}
	if e.Pending != 0 {
		return false
	}
	e.Result.Attempts++
	expected := e.Text[e.Pos]
	stat := e.stat(expected)
	stat.Samples++
	if r != expected {
		e.Pending = r
		e.Result.Errors++
		stat.Errors++
		return false
	}
	e.accept(expected, now, stat)
	return true
}

func (e *Engine) Backspace() {
	if e.Pending != 0 {
		e.Pending = 0
		e.Result.Corrections++
	}
}

func (e *Engine) accept(r rune, now time.Time, s *domain.CharacterStat) {
	from := e.LastAccepted
	if from.IsZero() {
		from = e.StartedAt
	}
	s.LatencyMS += float64(now.Sub(from).Milliseconds())
	e.LastAccepted = now
	e.Pos++
	e.Result.Correct++
}

func (e *Engine) Pause(now time.Time) {
	if !e.IsPaused() {
		e.PauseStarted = now
	}
}
func (e *Engine) Resume(now time.Time) {
	if e.IsPaused() {
		if !e.StartedAt.IsZero() {
			e.Paused += now.Sub(e.PauseStarted)
			e.LastAccepted = now
		}
		e.PauseStarted = time.Time{}
	}
}
func (e *Engine) IsPaused() bool { return !e.PauseStarted.IsZero() }
func (e *Engine) Done() bool     { return e.Pos >= len(e.Text) }

func (e *Engine) Finish(now time.Time) domain.SessionResult {
	d := now.Sub(e.StartedAt) - e.Paused
	if e.IsPaused() {
		d -= now.Sub(e.PauseStarted)
	}
	if d < time.Millisecond {
		d = time.Millisecond
	}
	e.Result.Duration = d
	mins := d.Minutes()
	e.Result.CPM = float64(e.Result.Correct) / mins
	e.Result.WPM = e.Result.CPM / 5
	if e.Result.Attempts > 0 {
		e.Result.Accuracy = float64(e.Result.Attempts-e.Result.Errors) / float64(e.Result.Attempts)
	}
	return e.Result
}

func (e *Engine) Live(now time.Time) (float64, float64, time.Duration) {
	if e.StartedAt.IsZero() {
		return 0, 1, 0
	}
	d := now.Sub(e.StartedAt) - e.Paused
	if e.IsPaused() {
		d -= now.Sub(e.PauseStarted)
	}
	wpm := 0.0
	if d > 0 {
		wpm = float64(e.Result.Correct) / 5 / d.Minutes()
	}
	acc := 1.0
	if e.Result.Attempts > 0 {
		acc = float64(e.Result.Attempts-e.Result.Errors) / float64(e.Result.Attempts)
	}
	return wpm, acc, d
}

func (e *Engine) stat(r rune) *domain.CharacterStat {
	s := e.Result.Chars[r]
	if s == nil {
		s = &domain.CharacterStat{Rune: r}
		e.Result.Chars[r] = s
	}
	return s
}

func UpdateProgress(old map[rune]domain.CharacterProgress, result domain.SessionResult, settings domain.Settings) map[rune]domain.CharacterProgress {
	const alpha = 0.2
	for r, s := range result.Chars {
		if unicode.IsSpace(r) || s.Samples == 0 {
			continue
		}
		p := old[r]
		p.Language = result.Language
		p.Rune = r
		lat := s.LatencyMS / float64(max(1, s.Samples-s.Errors))
		acc := float64(s.Samples-s.Errors) / float64(s.Samples)
		if p.Samples == 0 {
			p.LatencyMS, p.Accuracy = lat, acc
		} else {
			p.LatencyMS = alpha*lat + (1-alpha)*p.LatencyMS
			p.Accuracy = alpha*acc + (1-alpha)*p.Accuracy
		}
		p.Samples += s.Samples
		p.Errors += s.Errors
		targetMS := 12000 / settings.TargetWPM
		speedScore := math.Min(1, targetMS/math.Max(1, p.LatencyMS))
		accuracyScore := math.Min(1, p.Accuracy/settings.Accuracy)
		sampleScore := math.Min(1, float64(p.Samples)/30)
		p.Confidence = math.Min(speedScore, math.Min(accuracyScore, sampleScore))
		if r == result.TargetRune {
			if p.Confidence >= .999 {
				p.MasteryStreak++
			} else {
				p.MasteryStreak = 0
			}
		}
		old[r] = p
	}
	return old
}

func LearningState(p domain.LanguageProfile, progress map[rune]domain.CharacterProgress, improve bool) (map[rune]bool, rune) {
	if improve {
		unlocked := map[rune]bool{}
		target := rune(0)
		weakest := 2.0
		for _, r := range p.UnlockOrder {
			unlocked[r] = true
			c := progress[r].Confidence
			if c < weakest {
				weakest, target = c, r
			}
		}
		return unlocked, target
	}
	unlocked := InitialUnlocked(p)
	target := rune(0)
	weakest := 2.0
	for _, r := range p.UnlockOrder[:min(6, len(p.UnlockOrder))] {
		if c := progress[r].Confidence; c < .999 && c < weakest {
			weakest, target = c, r
		}
	}
	if target != 0 {
		return unlocked, target
	}
	for i, r := range p.UnlockOrder {
		if i < 6 {
			continue
		}
		if progress[r].MasteryStreak >= 2 {
			unlocked[r] = true
		} else {
			return unlocked, r
		}
	}
	return unlocked, p.UnlockOrder[len(p.UnlockOrder)-1]
}
