package trainer

import (
	"testing"
	"time"
	"unicode"

	"btyper/internal/domain"
)

func TestSchedulerCalibratesRunesInOrderThenSelectsOverdue(t *testing.T) {
	p := Profiles()["en"]
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	skills := map[string]domain.Skill{}
	if got := SelectSkill(p, skills, now); got.Pattern != string(p.UnlockOrder[0]) {
		t.Fatal(got)
	}
	for _, r := range p.UnlockOrder {
		skills[SkillKey(domain.SkillRune, string(r))] = domain.Skill{Language: "en", Kind: domain.SkillRune, Pattern: string(r), Samples: 6, Confidence: .2, DueAt: now.Add(time.Hour)}
	}
	if got := SelectSkill(p, skills, now); got.Kind != domain.SkillRune {
		t.Fatalf("bigram selected before rune foundation was ready: %v", got)
	}
	for _, r := range p.UnlockOrder {
		skill := skills[SkillKey(domain.SkillRune, string(r))]
		skill.Samples = RuneFoundationSamples
		skill.Confidence = 1
		skills[SkillKey(domain.SkillRune, string(r))] = skill
	}
	candidates := CandidateSkills(p)
	var bigram domain.Skill
	for _, skill := range candidates {
		if skill.Kind == domain.SkillBigram {
			bigram = skill
			break
		}
	}
	if got := SelectSkill(p, skills, now); got.Pattern != bigram.Pattern {
		t.Fatalf("got %v, want first bigram %v", got, bigram)
	}
	for _, skill := range candidates {
		if skill.Kind != domain.SkillRune {
			skill.Samples = 10
			skill.Confidence = 1
			skill.DueAt = now.Add(time.Hour)
			skills[SkillKey(skill.Kind, skill.Pattern)] = skill
		}
	}
	weak := skills[SkillKey(domain.SkillRune, string(p.UnlockOrder[3]))]
	weak.Confidence = .2
	weak.DueAt = now.Add(-48 * time.Hour)
	skills[SkillKey(weak.Kind, weak.Pattern)] = weak
	if got := SelectSkill(p, skills, now); got.Pattern != weak.Pattern {
		t.Fatalf("overdue weak skill not selected: %v", got)
	}
}

func TestRuneFoundationBlocksBigramsAtBoundary(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		p := Profiles()[language]
		now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
		skills := map[string]domain.Skill{}
		for _, r := range p.UnlockOrder {
			pattern := string(r)
			skills[SkillKey(domain.SkillRune, pattern)] = domain.Skill{Samples: RuneFoundationSamples, Confidence: 1, DueAt: now.Add(time.Hour)}
		}
		last := string(p.UnlockOrder[len(p.UnlockOrder)-1])
		skill := skills[SkillKey(domain.SkillRune, last)]
		skill.Samples--
		skills[SkillKey(domain.SkillRune, last)] = skill
		if got := SelectSkill(p, skills, now); got.Kind != domain.SkillRune || got.Pattern != last {
			t.Fatalf("%s: selected %v with last rune at 29 samples", language, got)
		}
		skill.Samples++
		skills[SkillKey(domain.SkillRune, last)] = skill
		if got := SelectSkill(p, skills, now); got.Kind != domain.SkillBigram {
			t.Fatalf("%s: selected %v after all runes reached 30 samples", language, got)
		}
	}
}

func TestSchedulerPrioritizesOverdueObservedSkill(t *testing.T) {
	p := Profiles()["en"]
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	skills := map[string]domain.Skill{}
	for _, candidate := range CandidateSkills(p) {
		skills[SkillKey(candidate.Kind, candidate.Pattern)] = domain.Skill{
			Samples: 30, Confidence: 1, DueAt: now.Add(time.Hour),
		}
	}
	want := CandidateSkills(p)[len(p.UnlockOrder)+3]
	key := SkillKey(want.Kind, want.Pattern)
	overdue := skills[key]
	overdue.DueAt = now.Add(-time.Minute)
	skills[key] = overdue
	if got := SelectSkill(p, skills, now); got.Kind != want.Kind || got.Pattern != want.Pattern {
		t.Fatalf("selected %v, want overdue %v", got, want)
	}
}

func TestSchedulerSkipsDisabledOptionalCategories(t *testing.T) {
	p := Profiles()["en"]
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	skills := map[string]domain.Skill{}
	for _, candidate := range CandidateSkills(p) {
		skills[SkillKey(candidate.Kind, candidate.Pattern)] = domain.Skill{Language: "en", Kind: candidate.Kind, Pattern: candidate.Pattern, Samples: 30, Confidence: 1, DueAt: now.Add(time.Hour)}
	}
	for _, candidate := range CandidateSkills(p) {
		if candidate.Kind != domain.SkillRune && candidate.Kind != domain.SkillBigram {
			skill := skills[SkillKey(candidate.Kind, candidate.Pattern)]
			skill.Samples, skill.DueAt = 0, time.Time{}
			skills[SkillKey(candidate.Kind, candidate.Pattern)] = skill
		}
	}
	got := SelectSkillWithOptions(p, skills, now, false, false, false)
	if got.Kind == domain.SkillNumber || got.Kind == domain.SkillUppercase || got.Kind == domain.SkillPunctuation {
		t.Fatalf("selected optional category %s:%s while all were disabled", got.Kind, got.Pattern)
	}
}

func TestSchedulerCanKeepOneOptionalCategoryEnabled(t *testing.T) {
	p := Profiles()["en"]
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	skills := map[string]domain.Skill{}
	for _, candidate := range CandidateSkills(p) {
		skills[SkillKey(candidate.Kind, candidate.Pattern)] = domain.Skill{Language: "en", Kind: candidate.Kind, Pattern: candidate.Pattern, Samples: 1000, Confidence: 1, DueAt: now.Add(time.Hour)}
	}
	want := domain.Skill{Kind: domain.SkillNumber, Pattern: "7"}
	key := SkillKey(want.Kind, want.Pattern)
	skill := skills[key]
	skill.Samples, skill.DueAt = 0, time.Time{}
	skills[key] = skill
	got := SelectSkillWithOptions(p, skills, now, true, false, false)
	if got.Kind != want.Kind || got.Pattern != want.Pattern {
		t.Fatalf("selected %s:%s, want enabled skill %s:%s", got.Kind, got.Pattern, want.Kind, want.Pattern)
	}
}

func TestUpdateSkillsPromotesAndDemotesReviewLevel(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	settings := domain.DefaultSettings()
	key := SkillKey(domain.SkillRune, "e")
	result := domain.SessionResult{Language: "en", Skills: map[string]*domain.SkillStat{key: {Kind: domain.SkillRune, Pattern: "e", Samples: 6, LatencySamples: 6, LatencyMS: 600}}}
	got := UpdateSkills(nil, result, settings, now)[key]
	if got.Level != 1 || !got.DueAt.Equal(now.Add(24*time.Hour)) {
		t.Fatal(got)
	}
	result.Skills[key].Errors = 6
	got = UpdateSkills(map[string]domain.Skill{key: got}, result, settings, now.Add(time.Hour))[key]
	if got.Level != 0 || !got.DueAt.Equal(now.Add(time.Hour)) {
		t.Fatal(got)
	}
}

func TestReviewLevelRequiresSixObservationsAndStaysWithinBounds(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	settings := domain.DefaultSettings()
	key := SkillKey(domain.SkillRune, "e")
	stat := &domain.SkillStat{Kind: domain.SkillRune, Pattern: "e", Samples: 5, LatencySamples: 5, LatencyMS: 500}
	result := domain.SessionResult{Language: "en", Skills: map[string]*domain.SkillStat{key: stat}}
	got := UpdateSkills(map[string]domain.Skill{key: {Level: 2}}, result, settings, now)[key]
	if got.Level != 2 || !got.DueAt.Equal(now.Add(3*24*time.Hour)) {
		t.Fatalf("five observations changed review level: %v", got)
	}
	stat.Samples = 6
	stat.LatencySamples = 6
	stat.LatencyMS = 600
	got = UpdateSkills(map[string]domain.Skill{key: {Level: 5}}, result, settings, now)[key]
	if got.Level != 5 || !got.DueAt.Equal(now.Add(30*24*time.Hour)) {
		t.Fatalf("promotion exceeded maximum level: %v", got)
	}
	stat.Errors = 6
	got = UpdateSkills(map[string]domain.Skill{key: {Level: 0}}, result, settings, now)[key]
	if got.Level != 0 || !got.DueAt.Equal(now) {
		t.Fatalf("demotion exceeded minimum level: %v", got)
	}
}

func TestEngineCollectsBigramStatistics(t *testing.T) {
	now := time.Now()
	e := NewAdaptiveEngine("ab", "en", "ab", time.Time{})
	e.Input('a', now)
	e.Input('b', now.Add(100*time.Millisecond))
	stat := e.Result.Skills[SkillKey(domain.SkillBigram, "ab")]
	if stat == nil || stat.Samples != 1 || stat.LatencySamples != 1 || stat.LatencyMS != 100 {
		t.Fatal(stat)
	}
}

func TestCandidateSkillsIncludeNumbersUppercaseAndPunctuation(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		profile := Profiles()[language]
		found := map[string]bool{}
		for _, skill := range CandidateSkills(profile) {
			found[SkillKey(skill.Kind, skill.Pattern)] = true
		}
		for _, want := range []string{
			SkillKey(domain.SkillNumber, "7"),
			SkillKey(domain.SkillUppercase, string(unicode.ToUpper(profile.UnlockOrder[0]))),
			SkillKey(domain.SkillPunctuation, "!"),
		} {
			if !found[want] {
				t.Fatalf("%s: missing candidate %s", language, want)
			}
		}
	}
}

func TestEngineMeasuresNewSkillsInCustomText(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	e := NewEngine("A7!", domain.ModeText, "en", 0, time.Time{})
	for i, r := range "A7!" {
		e.Input(r, now.Add(time.Duration(i)*100*time.Millisecond))
	}
	for _, tc := range []struct {
		kind    domain.SkillKind
		pattern string
	}{
		{domain.SkillUppercase, "A"},
		{domain.SkillNumber, "7"},
		{domain.SkillPunctuation, "!"},
	} {
		stat := e.Result.Skills[SkillKey(tc.kind, tc.pattern)]
		if stat == nil || stat.Samples != 1 || stat.Errors != 0 {
			t.Fatalf("%s %q: %v", tc.kind, tc.pattern, stat)
		}
	}
	if _, ok := e.Result.Skills[SkillKey(domain.SkillBigram, "A7")]; ok {
		t.Fatal("non-letter pair was counted as a bigram")
	}
}
