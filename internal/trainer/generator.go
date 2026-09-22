package trainer

import (
	"math/rand"
	"strings"
	"unicode/utf8"

	"btyper/internal/domain"
)

type Generator struct{ Rand *rand.Rand }

func NewGenerator(seed int64) *Generator { return &Generator{Rand: rand.New(rand.NewSource(seed))} }

func (g *Generator) AdaptiveLesson(profile domain.LanguageProfile, unlocked map[rune]bool, target domain.Skill, weak map[rune]float64, limit int) string {
	rs := []rune(target.Pattern)
	if target.Kind != domain.SkillBigram || len(rs) != 2 {
		targetRune := rune(0)
		if len(rs) == 1 {
			targetRune = rs[0]
		}
		return g.Lesson(profile, unlocked, targetRune, weak, limit, domain.ModeAdaptive)
	}
	// Prefer real words containing the target pair. The fallback keeps the pair
	// intact and remains pronounceable enough for a short focused drill.
	var focused []string
	for _, word := range profile.Words {
		valid := strings.Contains(word, target.Pattern)
		for _, r := range word {
			if !unlocked[r] {
				valid = false
				break
			}
		}
		if valid {
			focused = append(focused, word)
		}
	}
	base := g.Lesson(profile, unlocked, 0, weak, limit, domain.ModeAdaptive)
	words := strings.Fields(base)
	for i := 0; i < len(words); i += 5 {
		for _, offset := range []int{0, 2} {
			at := i + offset
			if at >= len(words) {
				continue
			}
			if len(focused) > 0 {
				words[at] = focused[g.Rand.Intn(len(focused))]
			} else {
				words[at] = target.Pattern + target.Pattern
			}
		}
	}
	return strings.Join(words, " ")
}

func (g *Generator) Lesson(profile domain.LanguageProfile, unlocked map[rune]bool, target rune, weak map[rune]float64, limit int, mode domain.Mode) string {
	allowed := make([]rune, 0)
	for _, r := range profile.UnlockOrder {
		if unlocked[r] {
			allowed = append(allowed, r)
		}
	}
	if len(allowed) == 0 {
		allowed = append(allowed, profile.UnlockOrder[:min(6, len(profile.UnlockOrder))]...)
	}
	if target != 0 && !contains(allowed, target) {
		allowed = append(allowed, target)
	}
	allWords, targetWords := availableWords(profile.Words, allowed, target)
	used := make(map[string]int, len(allWords))
	var out []string
	length := 0
	focusOffset := g.Rand.Intn(5)
	realWordPercent := 65
	if mode == domain.ModeLearn {
		realWordPercent = 15
	}
	for length < limit {
		// Two words in every five deliberately contain the current target. This
		// keeps the lesson focused without making every word look the same.
		phase := (len(out) + focusOffset) % 5
		focus := target != 0 && (phase == 0 || phase == 2)
		pool := allWords
		if focus && len(targetWords) > 0 {
			pool = targetWords
		}
		word := ""
		if len(pool) > 0 && g.Rand.Intn(100) < realWordPercent {
			word = g.leastUsed(pool, used, out, mode != domain.ModeLearn)
		}
		if word == "" {
			forced := rune(0)
			if focus {
				forced = target
			}
			word = g.synthetic(profile, allowed, forced, weak, used, out)
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

func (g *Generator) leastUsed(pool []string, used map[string]int, recent []string, allowReuse bool) string {
	minimum := int(^uint(0) >> 1)
	var choices []string
	for _, word := range pool {
		if !allowReuse && used[word] > 0 {
			continue
		}
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
		if !allowReuse {
			return ""
		}
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

func (g *Generator) synthetic(profile domain.LanguageProfile, allowed []rune, target rune, weak map[rune]float64, used map[string]int, recent []string) string {
	best := ""
	for attempt := 0; attempt < 12; attempt++ {
		word := g.syntheticWord(profile, allowed, target, weak)
		if best == "" || used[word] < used[best] {
			best = word
		}
		if used[word] == 0 && !isRecent(word, recent, 4) {
			return word
		}
	}
	return best
}

func (g *Generator) syntheticWord(profile domain.LanguageProfile, allowed []rune, target rune, weak map[rune]float64) string {
	vowels := vowelRunes(profile.ID, allowed)
	consonants := without(allowed, vowels)
	n := 3 + g.Rand.Intn(5)
	runes := make([]rune, n)
	targetPos := -1
	startWithVowel := g.Rand.Intn(2) == 0
	if target != 0 {
		targetPos = g.Rand.Intn(n)
		if contains(vowels, target) {
			startWithVowel = targetPos%2 == 0
		} else if contains(consonants, target) {
			startWithVowel = targetPos%2 != 0
		}
	}
	for i := range runes {
		if i == targetPos {
			runes[i] = target
			continue
		}
		pool := consonants
		if (i%2 == 0) == startWithVowel {
			pool = vowels
		}
		if len(pool) == 0 {
			pool = allowed
		}
		runes[i] = g.nextRune(profile.Words, pool, runes[:i], weak)
		for retry := 0; i > 0 && len(pool) > 1 && runes[i] == runes[i-1] && retry < 3; retry++ {
			runes[i] = g.nextRune(profile.Words, pool, runes[:i], weak)
		}
	}
	return string(runes)
}

func vowelRunes(language string, allowed []rune) []rune {
	vowels := "aeiouy"
	if language == "ru" {
		vowels = "аеёиоуыэюя"
	}
	var out []rune
	for _, r := range allowed {
		if strings.ContainsRune(vowels, r) {
			out = append(out, r)
		}
	}
	return out
}

func without(all, excluded []rune) []rune {
	var out []rune
	for _, r := range all {
		if !contains(excluded, r) {
			out = append(out, r)
		}
	}
	return out
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
