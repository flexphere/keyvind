package tcellkit

import (
	"testing"

	"github.com/gdamore/tcell/v2"

	"github.com/flexphere/keyvind"
)

func TestFromEventKey(t *testing.T) {
	tests := []struct {
		name string
		ev   *tcell.EventKey
		want []keyvind.Key
	}{
		{"rune", tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone), []keyvind.Key{{Rune: 'd'}}},
		{"alt rune", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModAlt), []keyvind.Key{{Rune: 'x', Alt: true}}},
		{"space rune", tcell.NewEventKey(tcell.KeyRune, ' ', tcell.ModNone), []keyvind.Key{{Name: "space"}}},
		{"enter", tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone), []keyvind.Key{{Name: "enter"}}},
		{"esc", tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone), []keyvind.Key{{Name: "esc"}}},
		{"tab", tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone), []keyvind.Key{{Name: "tab"}}},
		{"backtab", tcell.NewEventKey(tcell.KeyBacktab, 0, tcell.ModNone), []keyvind.Key{{Name: "tab", Shift: true}}},
		{"ctrl-w", tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModCtrl), []keyvind.Key{{Rune: 'w', Ctrl: true}}},
		{"ctrl-a", tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModCtrl), []keyvind.Key{{Rune: 'a', Ctrl: true}}},
		{"ctrl-shift-x", tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModCtrl|tcell.ModShift), []keyvind.Key{{Rune: 'x', Ctrl: true, Shift: true}}},
		{"ctrl-shift-X upper", tcell.NewEventKey(tcell.KeyRune, 'X', tcell.ModCtrl|tcell.ModShift), []keyvind.Key{{Rune: 'x', Ctrl: true, Shift: true}}},
		{"shift-up", tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModShift), []keyvind.Key{{Name: "up", Shift: true}}},
		{"pgup", tcell.NewEventKey(tcell.KeyPgUp, 0, tcell.ModNone), []keyvind.Key{{Name: "pageup"}}},
		{"delete", tcell.NewEventKey(tcell.KeyDelete, 0, tcell.ModNone), []keyvind.Key{{Name: "del"}}},
		{"backspace", tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone), []keyvind.Key{{Name: "bs"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FromEventKey(tt.ev)
			if !keysEqual(got, tt.want) {
				t.Fatalf("FromEventKey(%v) = %v, want %v", tt.ev.Key(), got, tt.want)
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
