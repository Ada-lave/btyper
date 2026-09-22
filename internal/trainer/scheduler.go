package trainer

import (
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

	"btyper/internal/domain"
)

var reviewIntervals = [...]time.Duration{0, 24 * time.Hour, 3 * 24 * time.Hour, 7 * 24 * time.Hour, 14 * 24 * time.Hour, 30 * 24 * time.Hour}

const RuneFoundationSamples = 30

func SkillKey(kind domain.SkillKind, pattern string) string { return string(kind) + ":" + pattern }

// CandidateSkills returns the portable curriculum encoded by a language
// profile. Bigrams are ranked by their frequency in the bundled word list.
func CandidateSkills(p domain.LanguageProfile) []domain.Skill {
	out := make([]domain.Skill, 0, len(p.UnlockOrder)+64)
	for _, r := range p.UnlockOrder {
		out = append(out, domain.Skill{Language: p.ID, Kind: domain.SkillRune, Pattern: string(r)})
	}
	counts := map[string]int{}
	for _, word := range p.Words {
		rs := []rune(strings.ToLower(word))
		for i := 1; i < len(rs); i++ {
			if !unicode.IsLetter(rs[i-1]) || !unicode.IsLetter(rs[i]) {
				continue
			}
			counts[string(rs[i-1:i+1])]++
		}
	}
	type pair struct {
		pattern string
		count   int
	}
	pairs := make([]pair, 0, len(counts))
	for pattern, count := range counts {
		pairs = append(pairs, pair{pattern, count})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].pattern < pairs[j].pattern
	})
	for _, pair := range pairs[:min(64, len(pairs))] {
		out = append(out, domain.Skill{Language: p.ID, Kind: domain.SkillBigram, Pattern: pair.pattern})
	}
	return out
}

func SelectSkill(p domain.LanguageProfile, skills map[string]domain.Skill, now time.Time) domain.Skill {
	candidates := CandidateSkills(p)
	// Build a reliable sample base for every rune before introducing bigrams.
	// Six observations are enough to unlock a rune for generated words, but not
	// enough to make pair-level timing useful or understandable to the learner.
	runesReady := true
	for _, c := range candidates {
		if c.Kind == domain.SkillBigram {
			break
		}
		if skills[SkillKey(c.Kind, c.Pattern)].Samples < RuneFoundationSamples {
			runesReady = false
			state := skills[SkillKey(c.Kind, c.Pattern)]
			state.Language, state.Kind, state.Pattern = p.ID, c.Kind, c.Pattern
			return state
		}
	}
	bestScore := -1.0
	best := candidates[0]
	for index, candidate := range candidates {
		if candidate.Kind == domain.SkillBigram && !runesReady {
			continue
		}
		state := skills[SkillKey(candidate.Kind, candidate.Pattern)]
		state.Language, state.Kind, state.Pattern = p.ID, candidate.Kind, candidate.Pattern
		score := skillPriority(state, index, now)
		if score > bestScore {
			bestScore, best = score, state
		}
	}
	return best
}

func skillPriority(s domain.Skill, curriculumIndex int, now time.Time) float64 {
	if s.Samples == 0 {
		return 10000 - float64(curriculumIndex)
	}
	uncertainty := 1 / math.Sqrt(float64(s.Samples))
	overdue := 0.0
	if s.DueAt.IsZero() || !s.DueAt.After(now) {
		overdue = 2
		if !s.DueAt.IsZero() {
			overdue += math.Min(3, now.Sub(s.DueAt).Hours()/24/7)
		}
	}
	return overdue + (1-s.Confidence)*3 + uncertainty
}

func UpdateSkills(old map[string]domain.Skill, result domain.SessionResult, settings domain.Settings, now time.Time) map[string]domain.Skill {
	out := make(map[string]domain.Skill, len(old)+len(result.Skills))
	for key, value := range old {
		out[key] = value
	}
	for key, stat := range result.Skills {
		if stat.Samples == 0 || strings.TrimSpace(stat.Pattern) == "" {
			continue
		}
		s := out[key]
		s.Language, s.Kind, s.Pattern = result.Language, stat.Kind, stat.Pattern
		accuracy := float64(stat.Samples-stat.Errors) / float64(stat.Samples)
		if s.Samples == 0 {
			s.Accuracy = accuracy
		} else {
			s.Accuracy = .2*accuracy + .8*s.Accuracy
		}
		if stat.LatencySamples > 0 {
			latency := stat.LatencyMS / float64(stat.LatencySamples)
			if s.LatencySamples == 0 {
				s.LatencyMS = latency
			} else {
				s.LatencyMS = .2*latency + .8*s.LatencyMS
			}
		}
		s.Samples += stat.Samples
		s.Errors += stat.Errors
		s.LatencySamples += stat.LatencySamples
		targetMS := 12000 / settings.TargetWPM
		speed := 0.0
		if s.LatencyMS > 0 {
			speed = math.Min(1, targetMS/s.LatencyMS)
		}
		s.Confidence = math.Min(math.Min(1, s.Accuracy/settings.Accuracy), math.Min(speed, float64(s.Samples)/30))
		s.LastPracticed = now
		if stat.Samples >= 6 {
			passed := accuracy >= settings.Accuracy && stat.LatencySamples > 0 && stat.LatencyMS/float64(stat.LatencySamples) <= targetMS
			if passed {
				s.Level = min(len(reviewIntervals)-1, s.Level+1)
			} else {
				s.Level = max(0, s.Level-1)
			}
		}
		s.DueAt = now.Add(reviewIntervals[s.Level])
		out[key] = s
	}
	return out
}
