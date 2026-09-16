package trainer

import "btyper/internal/domain"

func Profiles() map[string]domain.LanguageProfile {
	return map[string]domain.LanguageProfile{
		"en": {
			ID: "en", NameID: "language.english",
			UnlockOrder: []rune("enitrlsaudcyghpmobwfvkxzqj"),
			Rows:        []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
			Finger:      fingers([]string{"qaz", "wsx", "edc", "rfvtgb", "yhnujm", "ik", "ol", "p"}),
			Words:       []string{"the", "there", "their", "learn", "letter", "line", "time", "train", "read", "write", "speed", "skill", "type", "quick", "brown", "fox", "jump", "over", "keyboard", "practice", "screen", "focus", "better", "daily", "simple", "word", "sound", "hand", "finger", "home"},
		},
		"ru": {
			ID: "ru", NameID: "language.russian",
			UnlockOrder: []rune("оеаинтсрвлкмдпуяызьбгчйхжюшцщэфъё"),
			Rows:        []string{"йцукенгшщзхъ", "фывапролджэ", "ячсмитьбю"},
			Finger:      fingers([]string{"йфя", "цыч", "увс", "камепи", "нртьго", "шлб", "щдю", "зхъэ"}),
			Words:       []string{"это", "как", "она", "они", "его", "для", "слово", "время", "рука", "палец", "строка", "текст", "урок", "скорость", "точность", "навык", "экран", "клавиша", "практика", "работа", "читать", "писать", "лучше", "каждый", "день", "просто", "новый", "звук", "дом", "мир"},
		},
	}
}

func fingers(groups []string) map[rune]string {
	names := []string{"LP", "LR", "LM", "LI", "RI", "RM", "RR", "RP"}
	m := map[rune]string{}
	for i, group := range groups {
		for _, r := range group {
			m[r] = names[i]
		}
	}
	return m
}
