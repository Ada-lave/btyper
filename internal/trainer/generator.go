package trainer

import (
	"math/rand"
	"strings"
	"unicode/utf8"

	"btyper/internal/domain"
)

type Generator struct{ Rand *rand.Rand }

func NewGenerator(seed int64) *Generator { return &Generator{Rand: rand.New(rand.NewSource(seed))} }

func (g *Generator) Lesson(profile domain.LanguageProfile, unlocked map[rune]bool, target rune, weak map[rune]float64, limit int) string {
	allowed := make([]rune, 0)
	for _, r := range profile.UnlockOrder {
		if unlocked[r] {
			allowed = append(allowed, r)
		}
	}
	if len(allowed) == 0 {
		allowed = append(allowed, profile.UnlockOrder[:min(6, len(profile.UnlockOrder))]...)
	}
	var out []string
	length := 0
	for length < limit {
		word := g.word(profile, allowed, target, weak)
		out = append(out, word)
		length += utf8.RuneCountInString(word) + 1
	}
	return strings.Join(out, " ")
}

func (g *Generator) word(p domain.LanguageProfile, allowed []rune, target rune, weak map[rune]float64) string {
	valid := func(s string) bool {
		hasTarget := target == 0
		for _, r := range s {
			if !contains(allowed, r) {
				return false
			}
			if r == target {
				hasTarget = true
			}
		}
		return hasTarget && utf8.RuneCountInString(s) >= 3 && utf8.RuneCountInString(s) <= 8
	}
	var candidates []string
	for _, w := range p.Words {
		if valid(w) {
			candidates = append(candidates, w)
		}
	}
	if len(candidates) > 0 && g.Rand.Intn(100) < 65 {
		return candidates[g.Rand.Intn(len(candidates))]
	}

	n := 3 + g.Rand.Intn(6)
	runes := make([]rune, n)
	for i := range runes {
		runes[i] = g.nextRune(p.Words, allowed, runes[:i], weak)
	}
	if target != 0 {
		runes[g.Rand.Intn(n)] = target
	}
	return string(runes)
}

// nextRune builds a tiny bigram/trigram model directly from the bundled word
// list. It falls back to frequency-weighted letters when the current prefix
// has no continuation in the restricted alphabet.
func (g *Generator) nextRune(words []string, allowed, prefix []rune, weak map[rune]float64) rune {
	weights := map[rune]float64{}
	for _, word := range words {
		rs := []rune(word)
		for i, candidate := range rs {
			if !contains(allowed, candidate) {
				continue
			}
			matched := len(prefix) == 0
			if len(prefix) >= 2 && i >= 2 {
				matched = rs[i-2] == prefix[len(prefix)-2] && rs[i-1] == prefix[len(prefix)-1]
			} else if len(prefix) >= 1 && i >= 1 {
				matched = rs[i-1] == prefix[len(prefix)-1]
			}
			if matched {
				weights[candidate] += 1 + (1-weak[candidate])*2
			}
		}
	}
	if len(weights) == 0 {
		return weightedRune(g.Rand, allowed, weak)
	}
	total := 0.0
	for _, w := range weights {
		total += w
	}
	x := g.Rand.Float64() * total
	for _, candidate := range allowed {
		x -= weights[candidate]
		if x <= 0 && weights[candidate] > 0 {
			return candidate
		}
	}
	return weightedRune(g.Rand, allowed, weak)
}

func weightedRune(r *rand.Rand, letters []rune, weak map[rune]float64) rune {
	total := 0.0
	for _, c := range letters {
		total += 1 + (1-weak[c])*2
	}
	x := r.Float64() * total
	for _, c := range letters {
		x -= 1 + (1-weak[c])*2
		if x <= 0 {
			return c
		}
	}
	return letters[len(letters)-1]
}

func contains(rs []rune, x rune) bool {
	for _, r := range rs {
		if r == x {
			return true
		}
	}
	return false
}

func InitialUnlocked(p domain.LanguageProfile) map[rune]bool {
	m := map[rune]bool{}
	for _, r := range p.UnlockOrder[:min(6, len(p.UnlockOrder))] {
		m[r] = true
	}
	return m
}
