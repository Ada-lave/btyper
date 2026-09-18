package trainer

import (
	"github.com/google/uuid"
	"maps"
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
	finished                              *domain.SessionResult
	activeSince                           time.Time
	activeDays                            map[string]time.Duration
}

func NewEngine(text string, mode domain.Mode, lang string, target rune, now time.Time) *Engine {
	return &Engine{Text: []rune(text), Result: domain.SessionResult{AttemptID: uuid.NewString(), StartedAt: now, Mode: mode, Language: lang, TargetRune: target, Text: text, Chars: map[rune]*domain.CharacterStat{}}}
}

func (e *Engine) Input(r rune, now time.Time) bool {
	if e.Done() || e.IsPaused() || unicode.IsControl(r) {
		return false
	}
	if e.StartedAt.IsZero() {
		e.StartedAt = now
		e.Result.StartedAt = now
		e.activeSince = now
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
	if !from.IsZero() {
		s.LatencyMS += float64(now.Sub(from).Microseconds()) / 1000
		s.LatencySamples++
	}
	e.LastAccepted = now
	e.Pos++
	e.Result.Correct++
	if e.Done() {
		e.closeActive(now)
	}
}

func (e *Engine) Pause(now time.Time) {
	if !e.IsPaused() {
		e.closeActive(now)
		e.PauseStarted = now
	}
}
func (e *Engine) Resume(now time.Time) {
	if e.IsPaused() {
		if !e.StartedAt.IsZero() {
			if !e.Done() {
				e.activeSince = now
			}
			e.Paused += now.Sub(e.PauseStarted)
			if !e.LastAccepted.IsZero() {
				e.LastAccepted = e.LastAccepted.Add(now.Sub(e.PauseStarted))
			}
		}
		e.PauseStarted = time.Time{}
	}
}

func addActiveDays(days map[string]time.Duration, start, end time.Time) {
	if start.IsZero() {
		return
	}
	for start.Before(end) {
		y, m, d := start.Date()
		next := time.Date(y, m, d+1, 0, 0, 0, 0, start.Location())
		until := end
		if next.Before(until) {
			until = next
		}
		days[start.Format("2006-01-02")] += until.Sub(start)
		start = until
	}
}

func (e *Engine) closeActive(now time.Time) {
	if e.activeDays == nil {
		e.activeDays = map[string]time.Duration{}
	}
	addActiveDays(e.activeDays, e.activeSince, now)
	e.activeSince = time.Time{}
}

func (e *Engine) PracticeTimes(now time.Time) []domain.PracticeTime {
	days := maps.Clone(e.activeDays)
	if days == nil {
		days = map[string]time.Duration{}
	}
	addActiveDays(days, e.activeSince, now)
	var out []domain.PracticeTime
	for day, duration := range days {
		out = append(out, domain.PracticeTime{AttemptID: e.Result.AttemptID, Day: day, Duration: duration})
	}
	return out
}
func (e *Engine) IsPaused() bool { return !e.PauseStarted.IsZero() }
func (e *Engine) Done() bool     { return e.Pos >= len(e.Text) }

func (e *Engine) Finish(now time.Time) domain.SessionResult {
	if e.finished != nil {
		return *e.finished
	}
	if e.StartedAt.IsZero() {
		return e.Result
	}
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
	r := e.Result
	e.finished = &r
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
	old = maps.Clone(old)
	if old == nil {
		old = map[rune]domain.CharacterProgress{}
	}
	for r, s := range result.Chars {
		if unicode.IsSpace(r) || s.Samples == 0 {
			continue
		}
		p := old[r]
		p.Language = result.Language
		p.Rune = r
		acc := float64(s.Samples-s.Errors) / float64(s.Samples)
		if p.Samples == 0 {
			p.Accuracy = acc
		} else {
			p.Accuracy = alpha*acc + (1-alpha)*p.Accuracy
		}
		if s.LatencySamples > 0 {
			lat := s.LatencyMS / float64(s.LatencySamples)
			if p.LatencyMS <= 0 {
				p.LatencyMS = lat
			} else {
				p.LatencyMS = alpha*lat + (1-alpha)*p.LatencyMS
			}
		}
		p.Samples += s.Samples
		p.Errors += s.Errors
		targetMS := 12000 / settings.TargetWPM
		speedScore := math.Min(1, targetMS/math.Max(1, p.LatencyMS))
		if p.LatencyMS <= 0 {
			speedScore = 0
		}
		accuracyScore := math.Min(1, p.Accuracy/settings.Accuracy)
		sampleScore := math.Min(1, float64(p.Samples)/30)
		p.Confidence = math.Min(speedScore, math.Min(accuracyScore, sampleScore))
		if r == result.TargetRune {
			if p.Confidence >= .999 {
				p.MasteryStreak++
				if p.MasteryStreak >= 2 {
					p.Mastered = true
				}
			} else {
				p.MasteryStreak = 0
			}
		}
		old[r] = p
	}
	if result.Mode == domain.ModeLearn {
		profile := Profiles()[result.Language]
		unlocked, target := LearningState(profile, old, false)
		if target != 0 {
			unlocked[target] = true
		}
		for r := range unlocked {
			p := old[r]
			p.Rune, p.Language, p.Unlocked = r, result.Language, true
			old[r] = p
		}
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
	for _, r := range p.UnlockOrder {
		if progress[r].Unlocked || progress[r].Mastered {
			unlocked[r] = true
		}
	}
	var target rune
	weakest := 2.0
	for _, r := range p.UnlockOrder {
		v := progress[r]
		if unlocked[r] && (!v.Mastered || v.Confidence < .999) && v.Confidence < weakest {
			weakest, target = v.Confidence, r
		}
	}
	if target != 0 {
		return unlocked, target
	}
	for _, r := range p.UnlockOrder {
		if !unlocked[r] {
			return unlocked, r
		}
	}
	for _, r := range p.UnlockOrder {
		if progress[r].Confidence < weakest {
			weakest, target = progress[r].Confidence, r
		}
	}
	return unlocked, target
}
