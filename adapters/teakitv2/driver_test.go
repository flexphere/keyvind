package teakitv2

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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

func kp(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r} }

func TestDriverSimpleKey(t *testing.T) {
	d := ambigDriver(t)
	cmds, cmd := d.Update(kp('x'))
	if len(cmds) != 1 || cmds[0].Target.Name != "del" {
		t.Fatalf("x: got %+v", cmds)
	}
	if cmd != nil {
		t.Fatal("x should not arm a timeout")
	}
}

func TestDriverArmAndResolveTimeout(t *testing.T) {
	d := ambigDriver(t)
	cmds, cmd := d.Update(kp('g'))
	if len(cmds) != 0 {
		t.Fatalf("g should not commit immediately, got %+v", cmds)
	}
	if cmd == nil {
		t.Fatal("g should arm a timeout cmd")
	}
	out, _ := d.Update(timeoutMsg{token: d.token})
	if len(out) != 1 || out[0].Target.Name != "g_solo" {
		t.Fatalf("timeout should commit g_solo, got %+v", out)
	}
}

func TestDriverExtendBeatsTimeout(t *testing.T) {
	d := ambigDriver(t)
	if _, cmd := d.Update(kp('g')); cmd == nil {
		t.Fatal("g should arm timeout")
	}
	cmds, _ := d.Update(kp('g')) // extends to "gg"
	if len(cmds) != 1 || cmds[0].Target.Name != "goto_top" {
		t.Fatalf("gg should commit goto_top, got %+v", cmds)
	}
}

func TestDriverStaleTimeoutIgnored(t *testing.T) {
	d := ambigDriver(t)
	d.Update(kp('g'))                        // arm token 1
	d.Update(timeoutMsg{token: d.token})     // resolve token 1
	d.Update(kp('g'))                        // arm token 2
	out, _ := d.Update(timeoutMsg{token: 1}) // re-deliver token 1: stale
	if len(out) != 0 {
		t.Fatalf("stale timeout should be ignored, got %+v", out)
	}
}

func TestDriverTickProducesTimeoutMsg(t *testing.T) {
	d := ambigDriver(t).WithTimeout(5 * time.Millisecond)
	_, cmd := d.Update(kp('g'))
	if cmd == nil {
		t.Fatal("g should arm timeout")
	}
	if _, ok := cmd().(timeoutMsg); !ok {
		t.Fatal("tick cmd should yield a timeoutMsg")
	}
}

func TestDriverTimeoutDisabled(t *testing.T) {
	d := ambigDriver(t).WithTimeout(0)
	cmds, cmd := d.Update(kp('g'))
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
