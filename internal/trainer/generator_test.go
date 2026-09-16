package trainer

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestLessonVarietyAndFocus(t *testing.T) {
	for _, language := range []string{"en", "ru"} {
		profile := Profiles()[language]
		unlocked := InitialUnlocked(profile)
		target := profile.UnlockOrder[0]
		for seed := int64(0); seed < 20; seed++ {
			lesson := NewGenerator(seed).Lesson(profile, unlocked, target, map[rune]float64{}, 140)
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
			if variety < .55 {
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
	one := NewGenerator(42).Lesson(profile, unlocked, 'e', map[rune]float64{}, 140)
	two := NewGenerator(42).Lesson(profile, unlocked, 'e', map[rune]float64{}, 140)
	if one != two {
		t.Fatal("same seed produced different lessons")
	}
}
