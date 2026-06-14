package tcellkit

import (
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"

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
func ambigDriver(t *testing.T, post func(tcell.Event) error) *Driver {
	t.Helper()
	km := keyvind.NewKeymap()
	add(t, km, "g", "g_solo")
	add(t, km, "gg", "goto_top")
	add(t, km, "x", "del")
	m := keyvind.NewMatcher(map[string]*keyvind.Keymap{"normal": km}, "normal")
	return New(m, post)
}

func rk(r rune) *tcell.EventKey { return tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone) }

// waitEvent returns the next posted event or fails after a generous timeout.
func waitEvent(t *testing.T, ch <-chan tcell.Event) tcell.Event {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(2 * time.Second):
		t.Fatal("expected a posted timeout event")
		return nil
	}
}

func TestDriverSimpleKey(t *testing.T) {
	posted := make(chan tcell.Event, 4)
	d := ambigDriver(t, func(ev tcell.Event) error { posted <- ev; return nil })
	cmds := d.Update(rk('x'))
	if len(cmds) != 1 || cmds[0].Target.Name != "del" {
		t.Fatalf("x: got %+v", cmds)
	}
	select {
	case ev := <-posted:
		t.Fatalf("x should not arm a timeout, but posted %v", ev)
	default:
	}
}

func TestDriverArmAndResolveTimeout(t *testing.T) {
	posted := make(chan tcell.Event, 4)
	d := ambigDriver(t, func(ev tcell.Event) error { posted <- ev; return nil }).
		WithTimeout(15 * time.Millisecond)

	if cmds := d.Update(rk('g')); len(cmds) != 0 {
		t.Fatalf("g should not commit immediately, got %+v", cmds)
	}
	ev := waitEvent(t, posted)
	cmds := d.Update(ev)
	if len(cmds) != 1 || cmds[0].Target.Name != "g_solo" {
		t.Fatalf("timeout should commit g_solo, got %+v", cmds)
	}
}

func TestDriverExtendBeatsTimeout(t *testing.T) {
	posted := make(chan tcell.Event, 4)
	d := ambigDriver(t, func(ev tcell.Event) error { posted <- ev; return nil }).
		WithTimeout(15 * time.Millisecond)
	d.Update(rk('g'))
	cmds := d.Update(rk('g')) // extends to "gg"
	if len(cmds) != 1 || cmds[0].Target.Name != "goto_top" {
		t.Fatalf("gg should commit goto_top, got %+v", cmds)
	}
}

func TestDriverStaleTimeoutIgnored(t *testing.T) {
	posted := make(chan tcell.Event, 4)
	d := ambigDriver(t, func(ev tcell.Event) error { posted <- ev; return nil }).
		WithTimeout(15 * time.Millisecond)

	d.Update(rk('g')) // arm token 1
	stale := waitEvent(t, posted)
	if cmds := d.Update(stale); len(cmds) != 1 { // valid: resolves g_solo
		t.Fatalf("first timeout should resolve, got %+v", cmds)
	}
	d.Update(rk('g'))                            // arm token 2
	if cmds := d.Update(stale); len(cmds) != 0 { // re-deliver token 1: ignored
		t.Fatalf("stale timeout should be ignored, got %+v", cmds)
	}
}

func TestDriverTimeoutDisabled(t *testing.T) {
	posted := make(chan tcell.Event, 4)
	d := ambigDriver(t, func(ev tcell.Event) error { posted <- ev; return nil }).WithTimeout(0)
	if cmds := d.Update(rk('g')); len(cmds) != 0 {
		t.Fatalf("g should still not commit, got %+v", cmds)
	}
	select {
	case ev := <-posted:
		t.Fatalf("WithTimeout(0) should not arm a timer, posted %v", ev)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestDriverIgnoresNonKeyEvent(t *testing.T) {
	d := ambigDriver(t, func(ev tcell.Event) error { return nil })
	if cmds := d.Update(tcell.NewEventResize(80, 24)); cmds != nil {
		t.Fatalf("resize event should be ignored, got %+v", cmds)
	}
}

func TestDriverMatcherAccessor(t *testing.T) {
	d := ambigDriver(t, func(ev tcell.Event) error { return nil })
	if d.Matcher().Mode() != "normal" {
		t.Fatalf("Matcher() mode = %q, want normal", d.Matcher().Mode())
	}
}
