package keyvind

import "testing"

// vimish builds a small vim-like setup with the Keymapper: operators in normal
// pointing to operand mode "motion"; operands (incl. the same motions, also
// placed standalone in normal) and a text object that is operand-only.
func vimish(t *testing.T) (map[string]*Keymap, []Conflict) {
	t.Helper()
	k := NewKeymapper()
	k.Operator(OperatorMapping{Mode: "normal", Keys: "d", Name: "delete", OperandMode: "motion"})
	k.Operator(OperatorMapping{Mode: "normal", Keys: "c", Name: "change", OperandMode: "motion"})
	// operands
	k.Map(Mapping{Mode: "motion", Keys: "w", Name: "word"})
	k.Map(Mapping{Mode: "motion", Keys: "iw", Name: "inner-word"})
	k.Map(Mapping{Mode: "motion", Keys: "f", Name: "find", AwaitsArg: true})
	// the same motions standalone in normal (must be added to both modes)
	k.Map(Mapping{Mode: "normal", Keys: "w", Name: "word"})
	k.Map(Mapping{Mode: "normal", Keys: "f", Name: "find", AwaitsArg: true})
	// normal-only
	k.Map(Mapping{Mode: "normal", Keys: "x", Name: "del-char"})
	k.Map(Mapping{Mode: "normal", Keys: "qas", Name: "my-cmd"})
	keymaps, conflicts, err := k.Build(Key{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return keymaps, conflicts
}

func TestKeymapperBuildAndResolve(t *testing.T) {
	keymaps, conflicts := vimish(t)
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
	m := NewMatcher(keymaps, "normal")

	if cmds := feed(t, m, "3w"); len(cmds) != 1 || cmds[0].Target.Name != "word" || cmds[0].Count != 3 {
		t.Fatalf("3w: %+v", cmds)
	}
	if cmds := feed(t, m, "dw"); len(cmds) != 1 || cmds[0].Operator == nil || cmds[0].Target.Name != "word" {
		t.Fatalf("dw: %+v", cmds)
	}
	if cmds := feed(t, m, "diw"); len(cmds) != 1 || cmds[0].Target.Name != "inner-word" {
		t.Fatalf("diw: %+v", cmds)
	}
	if cmds := feed(t, m, "dd"); len(cmds) != 1 || !cmds[0].Linewise {
		t.Fatalf("dd: %+v", cmds)
	}
	if cmds := feed(t, m, "fx"); len(cmds) != 1 || cmds[0].Target.Name != "find" || cmds[0].Arg != (Key{Rune: 'x'}) {
		t.Fatalf("fx: %+v", cmds)
	}
	if cmds := feed(t, m, "dfx"); len(cmds) != 1 || cmds[0].Operator == nil || cmds[0].Arg != (Key{Rune: 'x'}) {
		t.Fatalf("dfx: %+v", cmds)
	}
	if cmds := feed(t, m, "qas"); len(cmds) != 1 || cmds[0].Target.Name != "my-cmd" {
		t.Fatalf("qas: %+v", cmds)
	}
}

func TestKeymapperOverrideLastWins(t *testing.T) {
	k := NewKeymapper()
	k.Map(Mapping{Mode: "normal", Keys: "l", Name: "left"})
	k.Map(Mapping{Mode: "normal", Keys: "l", Name: "right"}) // override
	keymaps, _, err := k.Build(Key{})
	if err != nil {
		t.Fatal(err)
	}
	m := NewMatcher(keymaps, "normal")
	if cmds := feed(t, m, "l"); len(cmds) != 1 || cmds[0].Target.Name != "right" {
		t.Fatalf("override: %+v", cmds)
	}
}

func TestKeymapperUnmap(t *testing.T) {
	k := NewKeymapper()
	k.Map(Mapping{Mode: "normal", Keys: "x", Name: "del"})
	k.Map(Mapping{Mode: "normal", Keys: "x", Name: ""}) // unmap
	keymaps, _, err := k.Build(Key{})
	if err != nil {
		t.Fatal(err)
	}
	m := NewMatcher(keymaps, "normal")
	if cmds := feed(t, m, "x"); len(cmds) != 0 {
		t.Fatalf("unmap: %+v", cmds)
	}
}

func TestKeymapperParseError(t *testing.T) {
	k := NewKeymapper()
	k.Map(Mapping{Mode: "normal", Keys: "<C-", Name: "bad"})
	if _, _, err := k.Build(Key{}); err == nil {
		t.Fatal("expected error for bad key spec")
	}
}

func TestKeymapperConflicts(t *testing.T) {
	k := NewKeymapper()
	k.Map(Mapping{Mode: "normal", Keys: "g", Name: "g-solo"})
	k.Map(Mapping{Mode: "normal", Keys: "gg", Name: "goto-top"})
	_, conflicts, err := k.Build(Key{})
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0].Keys != "g" || conflicts[0].Mode != "normal" {
		t.Fatalf("conflicts: %+v", conflicts)
	}
}

func TestKeymapperCarriesDesc(t *testing.T) {
	k := NewKeymapper()
	k.Map(Mapping{Mode: "normal", Keys: "x", Name: "del-char", Desc: "delete char under cursor"})
	k.Operator(OperatorMapping{Mode: "normal", Keys: "d", Name: "delete", OperandMode: "motion", Desc: "delete operator"})
	k.Map(Mapping{Mode: "motion", Keys: "w", Name: "word", Desc: "next word"})
	keymaps, conflicts, err := k.Build(Key{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %+v", conflicts)
	}
	m := NewMatcher(keymaps, "normal")

	// terminal binding carries its description on Target
	cmds := feed(t, m, "x")
	if len(cmds) != 1 || cmds[0].Target.Desc != "delete char under cursor" {
		t.Fatalf("x desc: %+v", cmds)
	}
	// operator + operand: both descriptions survive
	cmds = feed(t, m, "dw")
	if len(cmds) != 1 || cmds[0].Operator.Desc != "delete operator" || cmds[0].Target.Desc != "next word" {
		t.Fatalf("dw desc: operator=%q target=%q", cmds[0].Operator.Desc, cmds[0].Target.Desc)
	}
}
