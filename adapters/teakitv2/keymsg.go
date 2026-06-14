// Package teakitv2 adapts the framework-agnostic keyvind core to bubbletea v2
// (charm.land/bubbletea/v2). Unlike the v1 adapter (teakit), bubbletea v2 reports
// precise modifiers via Key.Mod, so on terminals with an enhanced keyboard
// protocol (e.g. Kitty) it can distinguish Ctrl+Shift+x from Ctrl+x — a binding
// like <C-S-x> then matches. On legacy terminals the Shift modifier is simply
// absent and behavior degrades to the same as v1.
//
// Importing the keyvind core alone never pulls in bubbletea; only this module
// depends on it.
package teakitv2

import (
	"unicode"

	tea "charm.land/bubbletea/v2"

	"github.com/flexphere/keyvind"
)

// specialKeys maps bubbletea v2 key codes to keyvind's canonical names. v2 keeps
// Enter/Tab/Backspace/Space as their ASCII codes and navigation keys above
// unicode.MaxRune, so none collide with printable runes.
var specialKeys = map[rune]string{
	tea.KeyEnter:     "enter",
	tea.KeyEscape:    "esc", // tea.KeyEsc == tea.KeyEscape
	tea.KeyTab:       "tab",
	tea.KeyBackspace: "bs",
	tea.KeySpace:     "space",
	tea.KeyDelete:    "del",
	tea.KeyUp:        "up",
	tea.KeyDown:      "down",
	tea.KeyLeft:      "left",
	tea.KeyRight:     "right",
	tea.KeyHome:      "home",
	tea.KeyEnd:       "end",
	tea.KeyPgUp:      "pageup",
	tea.KeyPgDown:    "pagedown",
}

// FromKeyPress converts a bubbletea v2 key-press message into zero or more
// keyvind.Keys. A normal keypress yields one Key; text that decodes to multiple
// runes yields one Key each. Key codes keyvind cannot yet express (function keys,
// etc.) yield nil.
func FromKeyPress(msg tea.KeyPressMsg) []keyvind.Key {
	mod := msg.Mod
	alt := mod&tea.ModAlt != 0
	ctrl := mod&tea.ModCtrl != 0
	shift := mod&tea.ModShift != 0

	if name, ok := specialKeys[msg.Code]; ok {
		return []keyvind.Key{{Name: name, Ctrl: ctrl, Shift: shift, Alt: alt}}
	}

	// Control combos: Code is the base (unshifted) key; Shift is a modifier, so
	// Ctrl+Shift+x arrives as Code 'x' with Mod = Ctrl|Shift.
	if ctrl {
		if msg.Code == 0 || msg.Code > unicode.MaxRune {
			return nil
		}
		return []keyvind.Key{{Rune: toLower(msg.Code), Ctrl: true, Shift: shift, Alt: alt}}
	}

	// Plain printable: prefer the actual text (the shift is already baked into
	// the character, so it carries no separate Shift flag).
	text := msg.Text
	if text == "" {
		if msg.Code == 0 || msg.Code > unicode.MaxRune {
			return nil
		}
		text = string(msg.Code)
	}
	keys := make([]keyvind.Key, 0, len(text))
	for _, r := range text {
		if r == ' ' {
			keys = append(keys, keyvind.Key{Name: "space", Alt: alt})
			continue
		}
		keys = append(keys, keyvind.Key{Rune: r, Alt: alt})
	}
	return keys
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
