package teakitv2

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/flexphere/keyvind"
)

func TestFromKeyPress(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyPressMsg
		want []keyvind.Key
	}{
		{"rune", tea.KeyPressMsg{Code: 'd'}, []keyvind.Key{{Rune: 'd'}}},
		{"alt rune", tea.KeyPressMsg{Code: 'x', Mod: tea.ModAlt}, []keyvind.Key{{Rune: 'x', Alt: true}}},
		{"ctrl", tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl}, []keyvind.Key{{Rune: 'x', Ctrl: true}}},
		{"ctrl-shift", tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl | tea.ModShift}, []keyvind.Key{{Rune: 'x', Ctrl: true, Shift: true}}},
		{"ctrl-shift upper code", tea.KeyPressMsg{Code: 'X', Mod: tea.ModCtrl | tea.ModShift}, []keyvind.Key{{Rune: 'x', Ctrl: true, Shift: true}}},
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}, []keyvind.Key{{Name: "enter"}}},
		{"space code", tea.KeyPressMsg{Code: tea.KeySpace}, []keyvind.Key{{Name: "space"}}},
		{"up", tea.KeyPressMsg{Code: tea.KeyUp}, []keyvind.Key{{Name: "up"}}},
		{"shift-tab", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, []keyvind.Key{{Name: "tab", Shift: true}}},
		{"shifted letter stays literal", tea.KeyPressMsg{Code: 'a', Text: "A", Mod: tea.ModShift}, []keyvind.Key{{Rune: 'A'}}},
		{"multi-rune text", tea.KeyPressMsg{Text: "ab"}, []keyvind.Key{{Rune: 'a'}, {Rune: 'b'}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromKeyPress(tt.msg)
			if !keysEqual(got, tt.want) {
				t.Fatalf("FromKeyPress(%+v) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func keysEqual(a, b []keyvind.Key) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
