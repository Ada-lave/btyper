package trainer

import (
	"btyper/internal/domain"
	"math"
	"testing"
	"time"
)

func TestTimingErrorsAndPause(t *testing.T) {
	start := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	e := NewEngine("ab", domain.ModeLearn, "en", 'b', time.Time{})
	e.Input('a', start)
	if e.Result.Chars['a'].LatencySamples != 0 {
		t.Fatal("first key must not supply a zero-latency sample")
	}
	e.Input('x', start.Add(100*time.Millisecond))
	e.Pause(start.Add(200 * time.Millisecond))
	e.Resume(start.Add(2200 * time.Millisecond))
	e.Backspace()
	e.Input('b', start.Add(2500*time.Millisecond))
	r := e.Finish(start.Add(2500 * time.Millisecond))
	if r.Duration != 500*time.Millisecond || r.Chars['b'].LatencyMS != 500 || r.Chars['b'].LatencySamples != 1 {
		t.Fatalf("timing: %+v / %+v", r, r.Chars['b'])
	}
	if r.Attempts != 3 || r.Corrections != 1 || math.Abs(r.Accuracy-2.0/3) > 1e-9 {
		t.Fatalf("incorrect metrics: %+v", r)
	}
	if later := e.Finish(start.Add(time.Hour)); later.Duration != r.Duration {
		t.Fatal("retry changed frozen result")
	}
}

func TestPauseBeforeFirstInput(t *testing.T) {
	now := time.Now()
	e := NewEngine("ab", domain.ModeLearn, "en", 'a', time.Time{})
	e.Pause(now)
	e.Resume(now.Add(time.Hour))
	e.Input('a', now.Add(2*time.Hour))
	e.Input('b', now.Add(2*time.Hour+time.Second))
	if r := e.Finish(now.Add(2*time.Hour + time.Second)); r.Duration != time.Second {
		t.Fatal(r.Duration)
	}
}

func TestDailyTimeSplitsAtMidnightAndExcludesPauses(t *testing.T) {
	zone := time.FixedZone("UTC+5", 5*60*60)
	start := time.Date(2026, 9, 18, 23, 59, 58, 0, zone)
	e := NewEngine("abc", domain.ModeText, "en", 0, time.Time{})
	e.Input('a', start)
	e.Pause(start.Add(time.Second))
	e.Resume(start.Add(3 * time.Second))
	e.Input('b', start.Add(4*time.Second))
	e.Input('c', start.Add(5*time.Second))
	got := map[string]time.Duration{}
	for _, v := range e.PracticeTimes(start.Add(time.Hour)) {
		got[v.Day] = v.Duration
	}
	if got["2026-09-18"] != time.Second || got["2026-09-19"] != 2*time.Second {
		t.Fatal(got)
	}
}

func TestLearningKeepsUnlockedLettersAndReviewsWeakest(t *testing.T) {
	profile := Profiles()["en"]
	p := map[rune]domain.CharacterProgress{}
	for _, r := range profile.UnlockOrder[:10] {
		p[r] = domain.CharacterProgress{Unlocked: true, Mastered: true, Confidence: 1}
	}
	v := p['e']
	v.Confidence = .4
	p['e'] = v
	unlocked, target := LearningState(profile, p, false)
	if target != 'e' || len(unlocked) != 10 {
		t.Fatalf("target %c, unlocked %v", target, unlocked)
	}
	v.Confidence = 1
	p['e'] = v
	_, target = LearningState(profile, p, false)
	if target != profile.UnlockOrder[10] {
		t.Fatal("did not advance")
	}
	for _, r := range profile.UnlockOrder {
		p[r] = domain.CharacterProgress{Unlocked: true, Mastered: true, Confidence: 1}
	}
	v = p['n']
	v.Confidence = .5
	p['n'] = v
	_, target = LearningState(profile, p, false)
	if target != 'n' {
		t.Fatal("completed alphabet did not select weakest")
	}
}

func TestMasteryRequiresTwoLessonsAndDoesNotMutateInput(t *testing.T) {
	p := map[rune]domain.CharacterProgress{}
	r := domain.SessionResult{Language: "en", Mode: domain.ModeLearn, TargetRune: 'e', Chars: map[rune]*domain.CharacterStat{'e': {Rune: 'e', Samples: 30, LatencySamples: 29, LatencyMS: 2900}}}
	first := UpdateProgress(p, r, domain.DefaultSettings())
	if len(p) != 0 || first['e'].Mastered || first['e'].MasteryStreak != 1 {
		t.Fatal("invalid first lesson")
	}
	second := UpdateProgress(first, r, domain.DefaultSettings())
	if !second['e'].Mastered || first['e'].Mastered {
		t.Fatal("invalid mastery transition")
	}
	r.Chars['e'].Errors = 25
	third := UpdateProgress(second, r, domain.DefaultSettings())
	if !third['e'].Mastered || !third['e'].Unlocked || third['e'].MasteryStreak != 0 {
		t.Fatal("regression removed mastery")
	}
}
