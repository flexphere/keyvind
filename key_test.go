package keyvind

import (
	"strings"
	"testing"
)

var leaderSpace = Key{Name: "space"}

func TestParseKeysErrorPrefix(t *testing.T) {
	if _, err := ParseKeys("<C-", leaderSpace); err == nil || !strings.Contains(err.Error(), "keyvind:") {
		t.Fatalf("error should be prefixed %q, got %v", "keyvind:", err)
	}
}

func TestParseKeys(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want []Key
	}{
		{"plain pair", "dd", []Key{{Rune: 'd'}, {Rune: 'd'}}},
		{"text object", "iw", []Key{{Rune: 'i'}, {Rune: 'w'}}},
		{"leader expands", "<leader>ff", []Key{{Name: "space"}, {Rune: 'f'}, {Rune: 'f'}}},
		{"ctrl modifier", "<C-w>v", []Key{{Rune: 'w', Ctrl: true}, {Rune: 'v'}}},
		{"alt via M", "<M-x>", []Key{{Rune: 'x', Alt: true}}},
		{"special enter", "<CR>", []Key{{Name: "enter"}}},
		{"special esc alias", "<Escape>", []Key{{Name: "esc"}}},
		{"ctrl special", "g<C-a>", []Key{{Rune: 'g'}, {Rune: 'a', Ctrl: true}}},
		{"literal lt", "<lt>", []Key{{Rune: '<'}}},
		{"multi modifier", "<C-S-Tab>", []Key{{Name: "tab", Ctrl: true, Shift: true}}},
		{"literal space normalizes", "a b", []Key{{Rune: 'a'}, {Name: "space"}, {Rune: 'b'}}},
		{"shift letter folds to upper", "<S-a>", []Key{{Rune: 'A'}}},
		{"shift upper letter", "<S-A>", []Key{{Rune: 'A'}}},
		{"ctrl upper folds to lower", "<C-X>", []Key{{Rune: 'x', Ctrl: true}}},
		{"ctrl shift keeps shift, base lower", "<C-S-x>", []Key{{Rune: 'x', Ctrl: true, Shift: true}}},
		{"ctrl shift upper base", "<C-S-X>", []Key{{Rune: 'x', Ctrl: true, Shift: true}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseKeys(tt.spec, leaderSpace)
			if err != nil {
				t.Fatalf("ParseKeys(%q) error: %v", tt.spec, err)
			}
			if !keysEqual(got, tt.want) {
				t.Fatalf("ParseKeys(%q) = %v, want %v", tt.spec, got, tt.want)
			}
		})
	}
}

func TestParseKeysErrors(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{"unterminated bracket", "<C-w"},
		{"empty token", "<>"},
		{"unknown modifier", "<X-w>"},
		{"unknown key name", "<frobnicate>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseKeys(tt.spec, leaderSpace); err == nil {
				t.Fatalf("ParseKeys(%q) expected error, got nil", tt.spec)
			}
		})
	}
}

func TestParseKeysLeaderUnset(t *testing.T) {
	if _, err := ParseKeys("<leader>x", Key{}); err == nil {
		t.Fatal("expected error when <leader> used without configured leader")
	}
}

func TestKeyString(t *testing.T) {
	tests := []struct {
		key  Key
		want string
	}{
		{Key{Rune: 'd'}, "d"},
		{Key{Rune: 'w', Ctrl: true}, "<C-w>"},
		{Key{Name: "enter"}, "<enter>"},
		{Key{Name: "tab", Ctrl: true, Shift: true}, "<C-S-tab>"},
		{Key{Rune: 'x', Alt: true}, "<A-x>"},
	}
	for _, tt := range tests {
		if got := tt.key.String(); got != tt.want {
			t.Errorf("Key%+v.String() = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func keysEqual(a, b []Key) bool {
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
