package teakit

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/keyvind"
)

func add(t *testing.T, km *keyvind.Keymap, spec, name string) {
	t.Helper()
	keys, err := keyvind.ParseKeys(spec, keyvind.Key{})
	if err != nil {
		t.Fatalf("ParseKeys(%q): %v", spec, err)
	}
	km.MustAdd(&keyvind.Binding{Keys: keys, Name: name})
}

// ambigDriver has "g" (solo) ambiguous with "gg", plus standalone "x".
func ambigDriver(t *testing.T) *Driver {
	t.Helper()
	km := keyvind.NewKeymap()
	add(t, km, "g", "g_solo")
	add(t, km, "gg", "goto_top")
	add(t, km, "x", "del")
	m := keyvind.NewMatcher(map[string]*keyvind.Keymap{"normal": km}, "normal")
	return New(m)
}

func rune1(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestDriverSimpleKey(t *testing.T) {
	d := ambigDriver(t)
	cmds, cmd := d.Update(rune1('x'))
	if len(cmds) != 1 || cmds[0].Target.Name != "del" {
		t.Fatalf("x: got cmds %+v", cmds)
	}
	if cmd != nil {
		t.Fatalf("x should not arm a timeout")
	}
}

func TestDriverArmAndResolveTimeout(t *testing.T) {
	d := ambigDriver(t)
	cmds, cmd := d.Update(rune1('g'))
	if len(cmds) != 0 {
		t.Fatalf("g should not commit immediately, got %+v", cmds)
	}
	if cmd == nil {
		t.Fatal("g should arm a timeout cmd")
	}
	// Firing the tick yields the internal timeout message.
	msg := cmd()
	tcmds, _ := d.Update(msg)
	if len(tcmds) != 1 || tcmds[0].Target.Name != "g_solo" {
		t.Fatalf("timeout should commit g_solo, got %+v", tcmds)
	}
}

func TestDriverExtendBeatsTimeout(t *testing.T) {
	d := ambigDriver(t)
	if _, cmd := d.Update(rune1('g')); cmd == nil {
		t.Fatal("g should arm timeout")
	}
	cmds, _ := d.Update(rune1('g')) // extends to "gg"
	if len(cmds) != 1 || cmds[0].Target.Name != "goto_top" {
		t.Fatalf("gg should commit goto_top, got %+v", cmds)
	}
}

func TestDriverStaleTimeoutIgnored(t *testing.T) {
	d := ambigDriver(t)
	// First arm (token 1) and capture its stale tick message.
	_, cmd1 := d.Update(rune1('g'))
	stale := cmd1()
	// Resolve the first sequence via its (valid) timer.
	if cmds, _ := d.Update(stale); len(cmds) != 1 {
		t.Fatalf("first timeout should resolve once, got %+v", cmds)
	}
	// Second arm (token 2).
	if _, cmd2 := d.Update(rune1('g')); cmd2 == nil {
		t.Fatal("second g should arm timeout")
	}
	// Re-delivering the OLD token-1 message must be ignored, not resolve again.
	cmds, _ := d.Update(stale)
	if len(cmds) != 0 {
		t.Fatalf("stale timeout should be ignored, got %+v", cmds)
	}
}

func TestDriverTimeoutDisabled(t *testing.T) {
	d := ambigDriver(t).WithTimeout(0)
	cmds, cmd := d.Update(rune1('g'))
	if len(cmds) != 0 {
		t.Fatalf("g should still not commit, got %+v", cmds)
	}
	if cmd != nil {
		t.Fatal("WithTimeout(0) should not arm a tick")
	}
}

func TestDriverIgnoresNonKeyMsg(t *testing.T) {
	d := ambigDriver(t)
	cmds, cmd := d.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if cmds != nil || cmd != nil {
		t.Fatalf("non-key msg should be ignored, got %+v / %v", cmds, cmd)
	}
}

func TestDriverMatcherAccessor(t *testing.T) {
	d := ambigDriver(t)
	if d.Matcher().Mode() != "normal" {
		t.Fatalf("Matcher() mode = %q, want normal", d.Matcher().Mode())
	}
}

func TestDefaultTimeoutValue(t *testing.T) {
	if DefaultTimeout != 1000*time.Millisecond {
		t.Fatalf("DefaultTimeout = %v, want 1s", DefaultTimeout)
	}
}
