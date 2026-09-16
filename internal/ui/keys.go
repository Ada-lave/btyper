package ui

import (
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
)

var russianPhysicalKeys = map[rune]rune{
	'c': 'с',
	'h': 'р',
	'j': 'о',
	'k': 'л',
	'l': 'д',
	'n': 'т',
	'o': 'щ',
	'q': 'й',
	'r': 'к',
	's': 'ы',
	'y': 'н',
}

func keyRune(k tea.Key) rune {
	if k.BaseCode != 0 {
		return k.BaseCode
	}
	if k.Code != 0 {
		return k.Code
	}
	r, _ := utf8.DecodeRuneInString(k.Text)
	return r
}

func matchesPhysical(k tea.Key, latin rune) bool {
	r := keyRune(k)
	return r == latin || r == russianPhysicalKeys[latin]
}

func isPlainKey(msg tea.KeyPressMsg, latin rune) bool {
	k := msg.Key()
	return k.Mod == 0 && matchesPhysical(k, latin)
}

func isCtrlKey(msg tea.KeyPressMsg, latin rune) bool {
	k := msg.Key()
	return k.Mod&tea.ModCtrl != 0 && matchesPhysical(k, latin)
}

func isUp(msg tea.KeyPressMsg) bool {
	return msg.Key().Code == tea.KeyUp || isPlainKey(msg, 'k')
}

func isDown(msg tea.KeyPressMsg) bool {
	return msg.Key().Code == tea.KeyDown || isPlainKey(msg, 'j')
}

func isLeft(msg tea.KeyPressMsg) bool {
	return msg.Key().Code == tea.KeyLeft || isPlainKey(msg, 'h')
}

func isRight(msg tea.KeyPressMsg) bool {
	return msg.Key().Code == tea.KeyRight || isPlainKey(msg, 'l')
}
