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
	allWords, targetWords := availableWords(profile.Words, allowed, target)
	used := make(map[string]int, len(allWords))
	var out []string
	length := 0
	focusOffset := g.Rand.Intn(5)
	for length < limit {
		// Two words in every five deliberately contain the current target. This
		// keeps the lesson focused without making every word look the same.
		focus := target != 0 && (len(out)+focusOffset)%5 < 2
		pool := allWords
		if focus && len(targetWords) > 0 {
			pool = targetWords
		}
		word := ""
		if len(pool) > 0 && g.Rand.Intn(100) < 75 {
			word = g.leastUsed(pool, used, out)
		}
		if word == "" {
			forced := rune(0)
			if focus {
				forced = target
			}
			word = g.synthetic(profile.Words, allowed, forced, weak, used, out)
		}
		used[word]++
		if len(out) > 0 {
			length++
		}
		out = append(out, word)
		length += utf8.RuneCountInString(word)
	}
	return strings.Join(out, " ")
}

func availableWords(words []string, allowed []rune, target rune) ([]string, []string) {
	var all, focused []string
	for _, word := range words {
		n := utf8.RuneCountInString(word)
		if n < 2 || n > 10 {
			continue
		}
		valid, hasTarget := true, target == 0
		for _, r := range word {
			if !contains(allowed, r) {
				valid = false
				break
			}
			if r == target {
				hasTarget = true
			}
		}
		if valid {
			all = append(all, word)
			if hasTarget {
				focused = append(focused, word)
			}
		}
	}
	return all, focused
}

func (g *Generator) leastUsed(pool []string, used map[string]int, recent []string) string {
	minimum := int(^uint(0) >> 1)
	var choices []string
	for _, word := range pool {
		if isRecent(word, recent, 4) {
			continue
		}
		count := used[word]
		if count < minimum {
			minimum, choices = count, []string{word}
		} else if count == minimum {
			choices = append(choices, word)
		}
	}
	if len(choices) == 0 {
		for _, word := range pool {
			if !isRecent(word, recent, 1) {
				choices = append(choices, word)
			}
		}
	}
	if len(choices) == 0 {
		return ""
	}
	return choices[g.Rand.Intn(len(choices))]
}

func (g *Generator) synthetic(words []string, allowed []rune, target rune, weak map[rune]float64, used map[string]int, recent []string) string {
	best := ""
	for attempt := 0; attempt < 12; attempt++ {
		word := g.syntheticWord(words, allowed, target, weak)
		if best == "" || used[word] < used[best] {
			best = word
		}
		if used[word] == 0 && !isRecent(word, recent, 4) {
			return word
		}
	}
	return best
}

func (g *Generator) syntheticWord(words []string, allowed []rune, target rune, weak map[rune]float64) string {
	n := 3 + g.Rand.Intn(6)
	runes := make([]rune, n)
	for i := range runes {
		runes[i] = g.nextRune(words, allowed, runes[:i], weak)
	}
	if target != 0 {
		runes[g.Rand.Intn(n)] = target
	}
	return string(runes)
}

func isRecent(word string, recent []string, window int) bool {
	start := max(0, len(recent)-window)
	for _, previous := range recent[start:] {
		if previous == word {
			return true
		}
	}
	return false
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
