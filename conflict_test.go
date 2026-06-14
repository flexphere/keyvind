package keyvind

import "testing"

func TestConflictsNoneWhenDistinct(t *testing.T) {
	// "d" is only an intermediate node (just "dd" bound), so it is not a terminal.
	km := keymap(t, term(t, "x", "del-char"), term(t, "dd", "del-line"), term(t, "w", "word"))
	if got := km.Conflicts(); len(got) != 0 {
		t.Fatalf("expected no conflicts, got %+v", got)
	}
}

func TestConflictsAmbiguous(t *testing.T) {
	km := keymap(t, term(t, "g", "g-solo"), term(t, "gg", "goto-top"))
	got := km.Conflicts()
	if len(got) != 1 || got[0].Kind != ConflictAmbiguous || got[0].Keys != "g" || got[0].Name != "g-solo" {
		t.Fatalf("got %+v", got)
	}
	if len(got[0].Extends) != 1 || got[0].Extends[0] != "goto-top" {
		t.Fatalf("Extends = %v", got[0].Extends)
	}
}

func TestConflictsOperatorShadow(t *testing.T) {
	// An operator "d" with a longer terminal "dx" in the same keymap.
	km := keymap(t, operator(t, "d", "delete", "o"), term(t, "dx", "delete-x"))
	got := km.Conflicts()
	if len(got) != 1 || got[0].Kind != ConflictOperatorShadow || got[0].Keys != "d" {
		t.Fatalf("got %+v", got)
	}
}

func TestConflictsArgShadow(t *testing.T) {
	km := keymap(t, argTerm(t, "f", "find"), term(t, "fb", "shadowed"))
	got := km.Conflicts()
	if len(got) != 1 || got[0].Kind != ConflictArgShadow || got[0].Keys != "f" {
		t.Fatalf("got %+v", got)
	}
	if len(got[0].Extends) != 1 || got[0].Extends[0] != "shadowed" {
		t.Fatalf("Extends = %v", got[0].Extends)
	}
}

func TestConflictsOrderIndependent(t *testing.T) {
	km := keymap(t, term(t, "gg", "goto-top"), term(t, "g", "g-solo")) // reverse order
	got := km.Conflicts()
	if len(got) != 1 || got[0].Keys != "g" {
		t.Fatalf("got %+v", got)
	}
}

func TestMatcherConflictsAcrossModes(t *testing.T) {
	a := keymap(t, term(t, "g", "a-g"), term(t, "gg", "a-gg"))
	b := keymap(t, term(t, "z", "b-z"), term(t, "zz", "b-zz"))
	m := NewMatcher(map[string]*Keymap{"beta": b, "alpha": a}, "alpha")
	got := m.Conflicts()
	if len(got) != 2 {
		t.Fatalf("expected 2, got %+v", got)
	}
	if got[0].Mode != "alpha" || got[0].Keys != "g" || got[1].Mode != "beta" || got[1].Keys != "z" {
		t.Fatalf("got %+v", got)
	}
}

func TestConflictKindString(t *testing.T) {
	cases := map[ConflictKind]string{
		ConflictAmbiguous:      "ambiguous",
		ConflictOperatorShadow: "operator-shadow",
		ConflictArgShadow:      "arg-shadow",
		ConflictKind(99):       "unknown",
	}
	for k, want := range cases {
		if got := k.String(); got != want {
			t.Errorf("ConflictKind(%d) = %q, want %q", int(k), got, want)
		}
	}
}
