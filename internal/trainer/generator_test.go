package trainer

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"btyper/internal/domain"
)

func TestLessonVarietyAndFocus(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		profile := Profiles()[language]
		unlocked := InitialUnlocked(profile)
		target := profile.UnlockOrder[0]
		for seed := int64(0); seed < 20; seed++ {
			lesson := NewGenerator(seed).Lesson(profile, unlocked, target, map[rune]float64{}, 140, domain.ModeLearn)
			words := strings.Fields(lesson)
			if len(words) < 15 {
				t.Fatalf("%s seed %d: lesson has only %d words", language, seed, len(words))
			}

			unique := map[string]bool{}
			focused := 0
			for i, word := range words {
				if i > 0 && word == words[i-1] {
					t.Fatalf("%s seed %d: adjacent word %q repeated", language, seed, word)
				}
				unique[word] = true
				if strings.ContainsRune(word, target) {
					focused++
				}
				for _, r := range word {
					if unicode.IsSpace(r) || !unlocked[r] {
						t.Fatalf("%s seed %d: locked rune %q in %q", language, seed, r, word)
					}
				}
			}

			variety := float64(len(unique)) / float64(len(words))
			if variety < .95 {
				t.Fatalf("%s seed %d: unique-word ratio %.2f is too low: %q", language, seed, variety, lesson)
			}
			focusRatio := float64(focused) / float64(len(words))
			if focusRatio < .30 {
				t.Fatalf("%s seed %d: target ratio %.2f is too low", language, seed, focusRatio)
			}
			if utf8.RuneCountInString(lesson) < 140 {
				t.Fatalf("%s seed %d: lesson is shorter than requested", language, seed)
			}
		}
	}
}

func TestLessonIsDeterministicForSeed(t *testing.T) {
	profile := Profiles()["en"]
	unlocked := InitialUnlocked(profile)
	one := NewGenerator(42).Lesson(profile, unlocked, 'e', map[rune]float64{}, 140, domain.ModeLearn)
	two := NewGenerator(42).Lesson(profile, unlocked, 'e', map[rune]float64{}, 140, domain.ModeLearn)
	if one != two {
		t.Fatal("same seed produced different lessons")
	}
}

func TestAdaptiveLessonsVaryWithSmallVocabulary(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		profile := Profiles()[language]
		unlocked := InitialUnlocked(profile)
		for _, pattern := range []string{string(profile.UnlockOrder[0]), string(profile.UnlockOrder[:2])} {
			kind := domain.SkillRune
			if utf8.RuneCountInString(pattern) == 2 {
				kind = domain.SkillBigram
			}
			target := domain.Skill{Language: language, Kind: kind, Pattern: pattern}
			lessons := map[string]bool{}
			for seed := int64(0); seed < 20; seed++ {
				lesson := NewGenerator(seed).AdaptiveLesson(profile, unlocked, target, nil, 140)
				if lessons[lesson] {
					t.Fatalf("%s %q: different seeds produced the same lesson", language, pattern)
				}
				lessons[lesson] = true
				words := strings.Fields(lesson)
				unique := map[string]bool{}
				focused := 0
				for _, word := range words {
					unique[word] = true
					if strings.Contains(word, pattern) {
						focused++
					}
					for _, r := range word {
						if !unlocked[r] {
							t.Fatalf("%s: locked rune %q in %q", language, r, word)
						}
					}
				}
				if float64(len(unique))/float64(len(words)) < .95 || focused < 2 {
					t.Fatalf("%s %q: insufficient variety or focus: %q", language, pattern, lesson)
				}
			}
		}
	}
}

func TestSyntheticLearningWordsAlternateLetterClasses(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		profile := Profiles()[language]
		allowed := profile.UnlockOrder[:6]
		vowels := vowelRunes(profile, allowed)
		generator := NewGenerator(7)
		for i := 0; i < 100; i++ {
			word := []rune(generator.syntheticWord(profile, allowed, allowed[0], map[rune]float64{}))
			for j := 1; j < len(word); j++ {
				if contains(vowels, word[j]) == contains(vowels, word[j-1]) {
					t.Fatalf("%s: %q does not alternate vowels and consonants", language, string(word))
				}
			}
		}
	}
}

func TestGeneratorRebuildsModelForAnotherLanguage(t *testing.T) {
	g := NewGenerator(42)
	for _, language := range []string{"en", "ru", "en"} {
		profile := Profiles()[language]
		unlocked := InitialUnlocked(profile)
		lesson := g.Lesson(profile, unlocked, profile.UnlockOrder[0], nil, 140, domain.ModeLearn)
		for _, r := range lesson {
			if r != ' ' && !unlocked[r] {
				t.Fatalf("%s: unexpected rune %q", language, r)
			}
		}
		if g.model[""][profile.UnlockOrder[0]] == 0 {
			t.Fatalf("%s: model still uses previous language", language)
		}
	}
}

func TestAdaptiveLessonContainsScheduledSkill(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		profile := Profiles()[language]
		unlocked := make(map[rune]bool, len(profile.UnlockOrder))
		for _, r := range profile.UnlockOrder {
			unlocked[r] = true
		}
		var pair domain.Skill
		for _, candidate := range CandidateSkills(profile) {
			if candidate.Kind == domain.SkillBigram {
				pair = candidate
				break
			}
		}
		for _, target := range []domain.Skill{
			{Language: language, Kind: domain.SkillRune, Pattern: string(profile.UnlockOrder[0])},
			pair,
			{Language: language, Kind: domain.SkillNumber, Pattern: "7"},
			{Language: language, Kind: domain.SkillUppercase, Pattern: strings.ToUpper(string(profile.UnlockOrder[0]))},
			{Language: language, Kind: domain.SkillPunctuation, Pattern: "!"},
		} {
			for seed := int64(0); seed < 10; seed++ {
				lesson := NewGenerator(seed).AdaptiveLesson(profile, unlocked, target, nil, 140)
				focused := 0
				for _, word := range strings.Fields(lesson) {
					if strings.Contains(word, target.Pattern) {
						focused++
					}
				}
				if focused < 2 {
					t.Fatalf("%s seed %d: target %q occurs in only %d words: %q", language, seed, target.Pattern, focused, lesson)
				}
			}
		}
	}
}
