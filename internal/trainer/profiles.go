package trainer

import "btyper/internal/domain"

func Profiles() map[string]domain.LanguageProfile {
	return map[string]domain.LanguageProfile{
		"en": {
			ID: "en", NameID: "language.english",
			UnlockOrder: []rune("enitrlsaudcyghpmobwfvkxzqj"),
			Rows:        []string{"qwertyuiop", "asdfghjkl", "zxcvbnm"},
			Finger:      fingers([]string{"qaz", "wsx", "edc", "rfvtgb", "yhnujm", "ik", "ol", "p"}),
			Words: []string{
				"in", "it", "is", "on", "to", "an", "at", "no", "or", "be", "we", "he", "me", "do", "go", "up",
				"ten", "net", "tie", "let", "lie", "tin", "one", "not", "note", "tone", "into", "line", "tile", "tree", "enter", "entire", "little",
				"the", "there", "their", "learn", "letter", "time", "train", "read", "write", "speed", "skill", "type", "quick", "brown", "fox", "jump", "over",
				"keyboard", "practice", "screen", "focus", "better", "daily", "simple", "word", "sound", "hand", "finger", "home", "work", "world", "light", "right",
				"start", "close", "open", "place", "point", "again", "under", "after", "before", "small", "large", "first", "last", "next", "every", "other", "same",
				"make", "take", "give", "keep", "move", "look", "think", "know", "good", "new", "long", "great", "clear", "correct", "steady", "relax", "repeat",
			},
		},
		"ru": {
			ID: "ru", NameID: "language.russian",
			UnlockOrder: []rune("оеаинтсрвлкмдпуяызьбгчйхжюшцщэфъё"),
			Rows:        []string{"йцукенгшщзхъ", "фывапролджэ", "ячсмитьбю"},
			Finger:      fingers([]string{"йфя", "цыч", "увс", "камепи", "нртьго", "шлб", "щдю", "зхъэ"}),
			Words: []string{
				"он", "но", "на", "не", "и", "а", "то", "её", "мы", "вы", "да", "из", "за", "по", "до", "уже",
				"она", "они", "оно", "иной", "нет", "тон", "нота", "енот", "тени", "неон", "иена", "тина", "тент", "нетто", "тонна", "тема", "имя", "имена", "тайна", "монета", "именно", "момент",
				"это", "как", "его", "для", "слово", "время", "рука", "палец", "строка", "текст", "урок", "скорость", "точность", "навык",
				"экран", "клавиша", "практика", "работа", "читать", "писать", "лучше", "каждый", "день", "просто", "новый", "звук", "дом", "мир",
				"место", "точка", "начало", "конец", "первый", "после", "снова", "рядом", "малый", "большой", "делать", "знать", "думать", "смотреть",
				"держать", "идти", "стоять", "хорошо", "быстро", "ровно", "точно", "спокойно", "повтор", "буква", "пальцы", "клавиши", "печать",
			},
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
