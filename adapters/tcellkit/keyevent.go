// Package tcellkit adapts the framework-agnostic keyvind core to tcell:
// it converts *tcell.EventKey into keyvind.Key and drives the matcher's ambiguity
// timeout by posting an event back onto the tcell event loop. Importing the
// keyvind core alone never pulls in tcell; only this module depends on it.
//
// Because tview, cview, and other widget toolkits expose *tcell.EventKey (e.g.
// via tview's SetInputCapture), the same FromEventKey conversion works for them
// too — see the tview example.
package tcellkit

import (
	"github.com/gdamore/tcell/v2"

	"github.com/flexphere/keyvind"
)

// specialKeys maps tcell named/control key codes to keyvind's canonical key names.
// tcell keeps Ctrl-letters (KeyCtrlA..KeyCtrlZ) in a distinct range from the
// ASCII control codes, so Enter/Tab/Backspace do not collide with them.
var specialKeys = map[tcell.Key]string{
	tcell.KeyEnter:      "enter",
	tcell.KeyEsc:        "esc", // == tcell.KeyEscape (same value)
	tcell.KeyTab:        "tab",
	tcell.KeyBackspace:  "bs", // KeyBS (0x08)
	tcell.KeyBackspace2: "bs", // KeyDEL (0x7f)
	tcell.KeyDelete:     "del",
	tcell.KeyUp:         "up",
	tcell.KeyDown:       "down",
	tcell.KeyLeft:       "left",
	tcell.KeyRight:      "right",
	tcell.KeyHome:       "home",
	tcell.KeyEnd:        "end",
	tcell.KeyPgUp:       "pageup",
	tcell.KeyPgDn:       "pagedown",
}

// FromEventKey converts a tcell key event into zero or more keyvind.Keys. A normal
// keypress yields exactly one Key. Key types keyvind cannot yet express (function
// keys, etc.) yield nil.
//
// The Ctrl/Alt/Shift modifiers are passed through when tcell reports them, so on
// terminals with an enhanced keyboard protocol a binding like <C-S-x> can match.
// Legacy terminals cannot distinguish e.g. Ctrl+Shift+x from Ctrl+x and simply
// won't report the Shift; a plain shifted letter stays a literal rune (no Shift
// flag), since tcell bakes the shift into the rune.
func FromEventKey(ev *tcell.EventKey) []keyvind.Key {
	mod := ev.Modifiers()
	alt := mod&tcell.ModAlt != 0
	shift := mod&tcell.ModShift != 0
	ctrl := mod&tcell.ModCtrl != 0
	k := ev.Key()

	switch k {
	case tcell.KeyRune:
		r := ev.Rune()
		// On an enhanced terminal Ctrl+rune (e.g. Ctrl+Shift+x) can arrive as a
		// rune carrying the Ctrl modifier rather than a KeyCtrl* code. Normalize
		// the base key to lower case so it matches <C-x>/<C-S-x>.
		if ctrl {
			return []keyvind.Key{{Rune: toLower(r), Ctrl: true, Shift: shift, Alt: alt}}
		}
		// Normalize a literal space to the "space" name so it matches bindings
		// written as <Space>/<leader>.
		if r == ' ' {
			return []keyvind.Key{{Name: "space", Alt: alt}}
		}
		return []keyvind.Key{{Rune: r, Alt: alt}}
	case tcell.KeyBacktab:
		return []keyvind.Key{{Name: "tab", Shift: true, Alt: alt}}
	}

	if name, ok := specialKeys[k]; ok {
		return []keyvind.Key{{Name: name, Ctrl: ctrl, Shift: shift, Alt: alt}}
	}
	if k >= tcell.KeyCtrlA && k <= tcell.KeyCtrlZ {
		return []keyvind.Key{{Rune: 'a' + rune(k-tcell.KeyCtrlA), Ctrl: true, Shift: shift, Alt: alt}}
	}
	return nil
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
