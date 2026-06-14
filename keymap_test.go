package keyvind

import "testing"

func TestKeymapAddDuplicate(t *testing.T) {
	km := NewKeymap()
	if err := km.Add(term(t, "dd", "first")); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if err := km.Add(term(t, "dd", "second")); err == nil {
		t.Fatal("expected duplicate error on second Add of \"dd\"")
	}
}

func TestKeymapAddEmpty(t *testing.T) {
	km := NewKeymap()
	if err := km.Add(&Binding{Name: "empty"}); err == nil {
		t.Fatal("expected error adding binding with empty key sequence")
	}
}

func TestKeymapPrefixCoexists(t *testing.T) {
	km := NewKeymap()
	if err := km.Add(term(t, "g", "g")); err != nil {
		t.Fatalf("add g: %v", err)
	}
	if err := km.Add(term(t, "gg", "gg")); err != nil {
		t.Fatalf("add gg over prefix g: %v", err)
	}
}

func TestMustAddPanicsOnDuplicate(t *testing.T) {
	km := NewKeymap()
	km.MustAdd(term(t, "x", "x"))
	defer func() {
		if recover() == nil {
			t.Fatal("MustAdd should panic on duplicate")
		}
	}()
	km.MustAdd(term(t, "x", "x2"))
}

func TestKeymapBindings(t *testing.T) {
	km := keymap(t,
		term(t, "x", "del"),
		term(t, "gg", "goto-top"),
		operator(t, "d", "delete", "o"),
		argTerm(t, "f", "find"),
	)
	bs := km.Bindings()
	if len(bs) != 4 {
		t.Fatalf("want 4 bindings, got %d", len(bs))
	}
	got := make(map[string]string, len(bs))
	for _, b := range bs {
		got[b.keyString()] = b.Name
	}
	for keys, name := range map[string]string{"x": "del", "gg": "goto-top", "d": "delete", "f": "find"} {
		if got[keys] != name {
			t.Fatalf("binding %q = %q, want %q (all: %v)", keys, got[keys], name, got)
		}
	}
	// deterministic, sorted by key sequence
	if bs[0].keyString() != "d" || bs[1].keyString() != "f" || bs[2].keyString() != "gg" || bs[3].keyString() != "x" {
		t.Fatalf("not sorted by keys: %q %q %q %q", bs[0].keyString(), bs[1].keyString(), bs[2].keyString(), bs[3].keyString())
	}
}
