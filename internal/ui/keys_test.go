package ui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func key(code rune, mod tea.KeyMod, base rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Text: string(code), Mod: mod, BaseCode: base})
}

func TestPhysicalKeysAcrossLayouts(t *testing.T) {
	if !isCtrlKey(key('c', tea.ModCtrl, 0), 'c') {
		t.Fatal("Latin Ctrl+C was not recognized")
	}
	if !isCtrlKey(key('с', tea.ModCtrl, 0), 'c') {
		t.Fatal("Russian-layout Ctrl+C was not recognized")
	}
	if !isCtrlKey(key('с', tea.ModCtrl, 'c'), 'c') {
		t.Fatal("Ctrl+C with an alternate base code was not recognized")
	}
	if isCtrlKey(key('с', 0, 0), 'c') {
		t.Fatal("plain Russian text was treated as Ctrl+C")
	}
}

func TestVimNavigationAcrossLayouts(t *testing.T) {
	for _, tt := range []struct {
		name string
		msg  tea.KeyPressMsg
		want func(tea.KeyPressMsg) bool
	}{
		{"Latin down", key('j', 0, 0), isDown},
		{"Russian down", key('о', 0, 0), isDown},
		{"Latin up", key('k', 0, 0), isUp},
		{"Russian up", key('л', 0, 0), isUp},
		{"Russian left", key('р', 0, 0), isLeft},
		{"Russian right", key('д', 0, 0), isRight},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.want(tt.msg) {
				t.Fatal("key was not recognized")
			}
		})
	}
}
