package teakit

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/keyvind"
)

func TestFromKeyMsg(t *testing.T) {
	tests := []struct {
		name string
		msg  tea.KeyMsg
		want []keyvind.Key
	}{
		{"rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, []keyvind.Key{{Rune: 'd'}}},
		{"alt rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}, Alt: true}, []keyvind.Key{{Rune: 'x', Alt: true}}},
		{"paste multi-rune", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a', 'b'}}, []keyvind.Key{{Rune: 'a'}, {Rune: 'b'}}},
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}, []keyvind.Key{{Name: "enter"}}},
		{"esc", tea.KeyMsg{Type: tea.KeyEsc}, []keyvind.Key{{Name: "esc"}}},
		{"tab", tea.KeyMsg{Type: tea.KeyTab}, []keyvind.Key{{Name: "tab"}}},
		{"shift-tab", tea.KeyMsg{Type: tea.KeyShiftTab}, []keyvind.Key{{Name: "tab", Shift: true}}},
		{"space type", tea.KeyMsg{Type: tea.KeySpace}, []keyvind.Key{{Name: "space"}}},
		{"space rune normalized", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}, []keyvind.Key{{Name: "space"}}},
		{"ctrl-w", tea.KeyMsg{Type: tea.KeyCtrlW}, []keyvind.Key{{Rune: 'w', Ctrl: true}}},
		{"ctrl-a", tea.KeyMsg{Type: tea.KeyCtrlA}, []keyvind.Key{{Rune: 'a', Ctrl: true}}},
		{"pgup", tea.KeyMsg{Type: tea.KeyPgUp}, []keyvind.Key{{Name: "pageup"}}},
		{"delete", tea.KeyMsg{Type: tea.KeyDelete}, []keyvind.Key{{Name: "del"}}},
		{"backspace", tea.KeyMsg{Type: tea.KeyBackspace}, []keyvind.Key{{Name: "bs"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromKeyMsg(tt.msg)
			if !keysEqual(got, tt.want) {
				t.Fatalf("FromKeyMsg(%+v) = %v, want %v", tt.msg, got, tt.want)
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
