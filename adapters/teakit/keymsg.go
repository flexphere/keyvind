// Package teakit adapts the framework-agnostic keyvind core to bubbletea:
// it converts tea.KeyMsg into keyvind.Key and drives the matcher's ambiguity
// timeout with tea.Tick. Importing the keyvind core alone never pulls in
// bubbletea; only this module depends on it.
package teakit

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/keyvind"
)

// specialNames maps bubbletea special key types to keyvind's canonical names.
// KeyEnter/KeyTab alias the Ctrl-M/Ctrl-I control codes, so they are matched
// here (before the Ctrl-letter range) and win.
var specialNames = map[tea.KeyType]string{
	tea.KeyEnter:     "enter",
	tea.KeyEsc:       "esc", // == tea.KeyEscape (same value)
	tea.KeyTab:       "tab",
	tea.KeyBackspace: "bs",
	tea.KeyDelete:    "del",
	tea.KeyUp:        "up",
	tea.KeyDown:      "down",
	tea.KeyLeft:      "left",
	tea.KeyRight:     "right",
	tea.KeyHome:      "home",
	tea.KeyEnd:       "end",
	tea.KeyPgUp:      "pageup",
	tea.KeyPgDown:    "pagedown",
	tea.KeySpace:     "space",
}

// FromKeyMsg converts a bubbletea key message into zero or more keyvind.Keys.
// A normal keypress yields exactly one Key; a paste (multiple runes in one
// message) yields one Key per rune. Unrecognized key types yield nil.
func FromKeyMsg(msg tea.KeyMsg) []keyvind.Key {
	t := msg.Type

	if t == tea.KeyShiftTab {
		return []keyvind.Key{{Name: "tab", Shift: true, Alt: msg.Alt}}
	}
	if name, ok := specialNames[t]; ok {
		return []keyvind.Key{{Name: name, Alt: msg.Alt}}
	}
	if t >= tea.KeyCtrlA && t <= tea.KeyCtrlZ {
		return []keyvind.Key{{Rune: 'a' + rune(t-tea.KeyCtrlA), Ctrl: true, Alt: msg.Alt}}
	}
	if t == tea.KeyRunes {
		keys := make([]keyvind.Key, 0, len(msg.Runes))
		for _, r := range msg.Runes {
			// Normalize a literal space to the "space" name so it matches
			// bindings written as <Space>/<leader>, regardless of how the
			// terminal delivered it.
			if r == ' ' {
				keys = append(keys, keyvind.Key{Name: "space", Alt: msg.Alt})
				continue
			}
			keys = append(keys, keyvind.Key{Rune: r, Alt: msg.Alt})
		}
		return keys
	}
	return nil
}
