package keyvind

import (
	"fmt"
	"strings"
)

// Key is a single, fully-decoded keystroke. It is intentionally independent of
// bubbletea so the core can be tested without a terminal. Conversion to and from
// tea.KeyMsg lives in the adapters/teakit module.
//
// A Key is either a printable rune (Rune != 0, Name == "") or a named special
// key (Name != "", Rune == 0), optionally combined with modifiers.
type Key struct {
	Rune  rune   // printable rune, 0 when Name is set
	Name  string // canonical special-key name: "enter","esc","tab","space","bs","up",...
	Ctrl  bool
	Alt   bool
	Shift bool
}

// String renders the Key in vim-like notation. It is the canonical form used as
// the trie edge label and in conflict/debug messages, so it must be stable.
func (k Key) String() string {
	base := k.Name
	if base == "" {
		base = string(k.Rune)
	}
	if !k.Ctrl && !k.Alt && !k.Shift && k.Name == "" {
		// plain printable rune: no angle brackets
		return base
	}
	var mods strings.Builder
	if k.Ctrl {
		mods.WriteString("C-")
	}
	if k.Alt {
		mods.WriteString("A-")
	}
	if k.Shift {
		mods.WriteString("S-")
	}
	return "<" + mods.String() + base + ">"
}

// specialNames maps lower-cased vim special-key spellings to their canonical name.
var specialNames = map[string]string{
	"cr": "enter", "enter": "enter", "return": "enter",
	"esc": "esc", "escape": "esc",
	"tab":   "tab",
	"space": "space",
	"bs":    "bs", "backspace": "bs",
	"del": "del", "delete": "del",
	"up": "up", "down": "down", "left": "left", "right": "right",
	"home": "home", "end": "end", "pageup": "pageup", "pagedown": "pagedown",
	"lt": "lt", // literal "<"
}

// ParseKeys turns a vim-style key spec into a sequence of Keys.
//
//	"dd"          -> [d d]
//	"<leader>ff"  -> [<leader> f f]   (leader expanded to its configured key)
//	"<C-w>v"      -> [Ctrl+w v]
//	"g<C-a>"      -> [g Ctrl+a]
//
// leader is the key that "<leader>" expands to. Pass the zero Key to reject
// "<leader>" usage.
func ParseKeys(spec string, leader Key) ([]Key, error) {
	var keys []Key
	runes := []rune(spec)
	for i := 0; i < len(runes); {
		r := runes[i]
		if r != '<' {
			if r == ' ' {
				// match how adapters deliver the space key
				keys = append(keys, Key{Name: "space"})
			} else {
				keys = append(keys, Key{Rune: r})
			}
			i++
			continue
		}
		// bracketed token: find the matching '>'
		end := indexRune(runes[i+1:], '>')
		if end < 0 {
			return nil, fmt.Errorf("keyvind: unterminated %q at offset %d in %q", "<", i, spec)
		}
		token := string(runes[i+1 : i+1+end])
		k, err := parseBracketToken(token, leader)
		if err != nil {
			return nil, fmt.Errorf("keyvind: %w in %q", err, spec)
		}
		keys = append(keys, k)
		i += 1 + end + 1
	}
	return keys, nil
}

func parseBracketToken(token string, leader Key) (Key, error) {
	if token == "" {
		return Key{}, fmt.Errorf("empty <> token")
	}
	if strings.EqualFold(token, "leader") {
		if (leader == Key{}) {
			return Key{}, fmt.Errorf("<leader> used but no leader configured")
		}
		return leader, nil
	}
	var k Key
	parts := strings.Split(token, "-")
	for i, p := range parts {
		last := i == len(parts)-1
		if !last {
			switch strings.ToUpper(p) {
			case "C":
				k.Ctrl = true
			case "A", "M":
				k.Alt = true
			case "S":
				k.Shift = true
			default:
				return Key{}, fmt.Errorf("unknown modifier %q", p)
			}
			continue
		}
		// final part: the base key
		if name, ok := specialNames[strings.ToLower(p)]; ok {
			if name == "lt" {
				k.Rune = '<'
			} else {
				k.Name = name
			}
		} else if len([]rune(p)) == 1 {
			k.Rune = []rune(p)[0]
		} else {
			return Key{}, fmt.Errorf("unknown key name %q", p)
		}
	}
	normalizeModifierKey(&k)
	return k, nil
}

// normalizeModifierKey folds printable-letter modifier combos to the form
// terminals actually deliver, so such bindings can match:
//   - Ctrl+letter uses the lowercase base: "<C-X>" == "<C-x>" (Ctrl+Shift keeps
//     the Shift flag, e.g. "<C-S-X>" == "<C-S-x>").
//   - A bare Shift+letter is the uppercase letter with no separate Shift flag:
//     "<S-a>" == "A". Terminals bake the shift into the rune.
//
// Shift on non-letter keys is layout-dependent and left as-is.
func normalizeModifierKey(k *Key) {
	if k.Name != "" || !isASCIILetter(k.Rune) {
		return
	}
	switch {
	case k.Ctrl:
		k.Rune = toLowerASCII(k.Rune)
	case k.Shift:
		k.Rune = toUpperASCII(k.Rune)
		k.Shift = false
	}
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func toLowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func toUpperASCII(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}

func indexRune(rs []rune, target rune) int {
	for i, r := range rs {
		if r == target {
			return i
		}
	}
	return -1
}
