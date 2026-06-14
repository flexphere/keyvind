package keyvind

import "testing"

// --- shared test helpers ---

func ks(t *testing.T, spec string) []Key {
	t.Helper()
	keys, err := ParseKeys(spec, Key{})
	if err != nil {
		t.Fatalf("ParseKeys(%q): %v", spec, err)
	}
	return keys
}

func key1(t *testing.T, spec string) Key {
	t.Helper()
	keys := ks(t, spec)
	if len(keys) != 1 {
		t.Fatalf("key1(%q): want one key, got %d", spec, len(keys))
	}
	return keys[0]
}

func term(t *testing.T, spec, name string) *Binding {
	return &Binding{Keys: ks(t, spec), Name: name}
}

func operator(t *testing.T, spec, name, operandMode string) *Binding {
	return &Binding{Keys: ks(t, spec), Name: name, OperandMode: operandMode}
}

func argTerm(t *testing.T, spec, name string) *Binding {
	return &Binding{Keys: ks(t, spec), Name: name, AwaitsArg: true}
}

func keymap(t *testing.T, binds ...*Binding) *Keymap {
	t.Helper()
	k := NewKeymap()
	for _, b := range binds {
		k.MustAdd(b)
	}
	return k
}

// feed runs every key of spec, then flushes a held ambiguous match via Timeout.
func feed(t *testing.T, m *Matcher, spec string) []Command {
	t.Helper()
	keys := ks(t, spec)
	cmds := make([]Command, 0, len(keys)+1)
	for _, k := range keys {
		cmds = append(cmds, m.Feed(k).Commands...)
	}
	cmds = append(cmds, m.Timeout().Commands...)
	return cmds
}

// opMatcher: normal mode with operators d/c → operand mode "o"; "o" holds the
// operands (motions / text objects). The same motion "w"/"f" is in both normal
// (standalone) and "o" (operand); "iw" is operand-only ("o" only).
func opMatcher(t *testing.T) *Matcher {
	t.Helper()
	normal := keymap(t,
		term(t, "x", "del-char"),
		term(t, "gg", "goto-top"),
		term(t, "g", "g-solo"), // ambiguous with gg
		term(t, "w", "word"),   // w standalone
		argTerm(t, "f", "find"),
		term(t, "i", "insert"),
		argTerm(t, "r", "replace"),
		operator(t, "d", "delete", "o"),
		operator(t, "c", "change", "o"),
	)
	o := keymap(t,
		term(t, "w", "word"),
		term(t, "iw", "inner-word"),
		argTerm(t, "f", "find"),
	)
	return NewMatcher(map[string]*Keymap{"normal": normal, "o": o}, "normal")
}

// --- tests ---

func TestStandaloneAction(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "x")
	if len(cmds) != 1 || cmds[0].Target.Name != "del-char" || cmds[0].Operator != nil || cmds[0].Count != 1 {
		t.Fatalf("x: %+v", cmds)
	}
}

func TestCountedMotionStandalone(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "3w")
	if len(cmds) != 1 || cmds[0].Target.Name != "word" || cmds[0].Operator != nil || cmds[0].Count != 3 {
		t.Fatalf("3w: %+v", cmds)
	}
}

func TestOperatorMotion(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "dw")
	if len(cmds) != 1 || cmds[0].Operator == nil || cmds[0].Operator.Name != "delete" || cmds[0].Target.Name != "word" {
		t.Fatalf("dw: %+v", cmds)
	}
	if cmds[0].Linewise {
		t.Fatalf("dw should not be linewise")
	}
}

func TestOperatorTextObject(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "diw")
	if len(cmds) != 1 || cmds[0].Operator.Name != "delete" || cmds[0].Target.Name != "inner-word" {
		t.Fatalf("diw: %+v", cmds)
	}
}

func TestDoubledOperatorLinewise(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "dd")
	if len(cmds) != 1 || cmds[0].Operator.Name != "delete" || !cmds[0].Linewise || cmds[0].Count != 1 {
		t.Fatalf("dd: %+v", cmds)
	}
}

func TestCountMultiplies(t *testing.T) {
	m := opMatcher(t)
	if cmds := feed(t, m, "3dw"); len(cmds) != 1 || cmds[0].Count != 3 {
		t.Fatalf("3dw: %+v", cmds)
	}
	if cmds := feed(t, m, "2d3w"); len(cmds) != 1 || cmds[0].Count != 6 {
		t.Fatalf("2d3w: %+v", cmds)
	}
	if cmds := feed(t, m, "2dd"); len(cmds) != 1 || cmds[0].Count != 2 || !cmds[0].Linewise {
		t.Fatalf("2dd: %+v", cmds)
	}
}

func TestStandaloneArg(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "fx")
	if len(cmds) != 1 || cmds[0].Target.Name != "find" || cmds[0].Arg != (Key{Rune: 'x'}) || cmds[0].Operator != nil {
		t.Fatalf("fx: %+v", cmds)
	}
}

func TestOperatorArg(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "dfx")
	if len(cmds) != 1 || cmds[0].Operator.Name != "delete" || cmds[0].Target.Name != "find" || cmds[0].Arg != (Key{Rune: 'x'}) {
		t.Fatalf("dfx: %+v", cmds)
	}
}

func TestReplaceArg(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "rx")
	if len(cmds) != 1 || cmds[0].Target.Name != "replace" || cmds[0].Arg != (Key{Rune: 'x'}) {
		t.Fatalf("rx: %+v", cmds)
	}
}

func TestArgEscCancels(t *testing.T) {
	m := opMatcher(t)
	if cmds := feed(t, m, "f<Esc>"); len(cmds) != 0 {
		t.Fatalf("f<Esc> should cancel, got %+v", cmds)
	}
	if cmds := feed(t, m, "x"); len(cmds) != 1 || cmds[0].Target.Name != "del-char" {
		t.Fatalf("x after cancel: %+v", cmds)
	}
}

// In normal mode, i and w are separate standalone commands; the text object "iw"
// lives only in the operand mode, so "iw" typed in normal is insert then word.
func TestStandaloneSeparateFromTextObject(t *testing.T) {
	m := opMatcher(t)
	r := m.Feed(key1(t, "i"))
	if len(r.Commands) != 1 || r.Commands[0].Target.Name != "insert" || r.Pending || r.ArmTimeout {
		t.Fatalf("i should resolve to insert immediately, got %+v", r)
	}
	// continue with w → standalone word (not a text object)
	w := m.Feed(key1(t, "w")).Commands
	if len(w) != 1 || w[0].Target.Name != "word" || w[0].Operator != nil {
		t.Fatalf("w in normal should be standalone word, got %+v", w)
	}
}

func TestAmbiguityHeldThenTimeout(t *testing.T) {
	m := opMatcher(t)
	r := m.Feed(key1(t, "g"))
	if len(r.Commands) != 0 || !r.ArmTimeout || !r.Pending {
		t.Fatalf("g should be held with ArmTimeout, got %+v", r)
	}
	tr := m.Timeout()
	if len(tr.Commands) != 1 || tr.Commands[0].Target.Name != "g-solo" {
		t.Fatalf("timeout should commit g-solo, got %+v", tr)
	}
}

func TestAmbiguityExtended(t *testing.T) {
	m := opMatcher(t)
	cmds := feed(t, m, "gg")
	if len(cmds) != 1 || cmds[0].Target.Name != "goto-top" {
		t.Fatalf("gg: %+v", cmds)
	}
}

func TestOperatorDeadEndResets(t *testing.T) {
	m := opMatcher(t)
	// "dq": q is not an operand in "o" → no command, state resets.
	if cmds := feed(t, m, "dq"); len(cmds) != 0 {
		t.Fatalf("dq: %+v", cmds)
	}
	if cmds := feed(t, m, "x"); len(cmds) != 1 || cmds[0].Target.Name != "del-char" {
		t.Fatalf("x after dq: %+v", cmds)
	}
}

func TestModeSwitch(t *testing.T) {
	normal := keymap(t, term(t, "i", "enter-insert"))
	insert := keymap(t, term(t, "<Esc>", "leave-insert"))
	m := NewMatcher(map[string]*Keymap{"normal": normal, "insert": insert}, "normal")
	if m.Mode() != "normal" {
		t.Fatalf("initial mode %q", m.Mode())
	}
	m.SetMode("insert")
	if cmds := feed(t, m, "<Esc>"); len(cmds) != 1 || cmds[0].Target.Name != "leave-insert" {
		t.Fatalf("esc in insert: %+v", cmds)
	}
}

func TestSetModeDiscardsPendingOperator(t *testing.T) {
	m := opMatcher(t)
	m.Feed(key1(t, "d")) // operator pending
	m.SetMode("normal")  // discard
	if cmds := feed(t, m, "x"); len(cmds) != 1 || cmds[0].Target.Name != "del-char" {
		t.Fatalf("x after SetMode: %+v", cmds)
	}
}

func TestZeroIsCountAfterDigit(t *testing.T) {
	m := opMatcher(t)
	if cmds := feed(t, m, "10w"); len(cmds) != 1 || cmds[0].Count != 10 {
		t.Fatalf("10w: %+v", cmds)
	}
}

func TestSetKeymapsSwapsBindings(t *testing.T) {
	a := keymap(t, term(t, "x", "del-char"))
	m := NewMatcher(map[string]*Keymap{"normal": a}, "normal")
	if cmds := feed(t, m, "x"); len(cmds) != 1 || cmds[0].Target.Name != "del-char" {
		t.Fatalf("before swap: %+v", cmds)
	}
	// swap to a different keymap set; the old binding is gone, the new one works
	b := keymap(t, term(t, "p", "play"))
	m.SetKeymaps(map[string]*Keymap{"normal": b})
	if cmds := feed(t, m, "x"); len(cmds) != 0 {
		t.Fatalf("x should be unmapped after swap: %+v", cmds)
	}
	if cmds := feed(t, m, "p"); len(cmds) != 1 || cmds[0].Target.Name != "play" {
		t.Fatalf("after swap: %+v", cmds)
	}
	if m.Mode() != "normal" {
		t.Fatalf("mode should be preserved, got %q", m.Mode())
	}
}

func TestSetKeymapsDiscardsInflight(t *testing.T) {
	m := opMatcher(t)
	m.Feed(key1(t, "d")) // operator pending
	m.SetKeymaps(map[string]*Keymap{"normal": keymap(t, term(t, "p", "play"))})
	if cmds := feed(t, m, "p"); len(cmds) != 1 || cmds[0].Target.Name != "play" || cmds[0].Operator != nil {
		t.Fatalf("in-flight operator should be discarded: %+v", cmds)
	}
}

func TestSetKeymapsMissingModeNoOps(t *testing.T) {
	m := NewMatcher(map[string]*Keymap{"normal": keymap(t, term(t, "x", "del-char"))}, "normal")
	// new set lacks "normal"; Feed no-ops until SetMode picks an existing mode
	m.SetKeymaps(map[string]*Keymap{"insert": keymap(t, term(t, "a", "append"))})
	if cmds := feed(t, m, "x"); len(cmds) != 0 {
		t.Fatalf("missing mode should no-op: %+v", cmds)
	}
	m.SetMode("insert")
	if cmds := feed(t, m, "a"); len(cmds) != 1 || cmds[0].Target.Name != "append" {
		t.Fatalf("after SetMode insert: %+v", cmds)
	}
}

// opMatcher2 has a multi-key operator "gU" → operand mode "o", which also holds
// a "gg" motion (so "g" is a shared prefix between the doubled operator and an
// operand), plus a single-key operator "d".
func opMatcher2(t *testing.T) *Matcher {
	t.Helper()
	normal := keymap(t,
		operator(t, "gU", "upper", "o"),
		operator(t, "d", "delete", "o"),
		term(t, "x", "del-char"),
	)
	o := keymap(t,
		term(t, "w", "word"),
		term(t, "gg", "goto-top"),
		term(t, "iw", "inner-word"),
	)
	return NewMatcher(map[string]*Keymap{"normal": normal, "o": o}, "normal")
}

func TestMultiKeyOperatorDoubledLinewise(t *testing.T) {
	m := opMatcher2(t)
	cmds := feed(t, m, "gUgU")
	if len(cmds) != 1 || cmds[0].Operator == nil || cmds[0].Operator.Name != "upper" || !cmds[0].Linewise {
		t.Fatalf("gUgU: %+v", cmds)
	}
}

func TestMultiKeyOperatorMotion(t *testing.T) {
	m := opMatcher2(t)
	// gUw: operator + motion (not linewise)
	if cmds := feed(t, m, "gUw"); len(cmds) != 1 || cmds[0].Operator.Name != "upper" || cmds[0].Target.Name != "word" || cmds[0].Linewise {
		t.Fatalf("gUw: %+v", cmds)
	}
	// gUiw: operator + text object
	if cmds := feed(t, m, "gUiw"); len(cmds) != 1 || cmds[0].Operator.Name != "upper" || cmds[0].Target.Name != "inner-word" {
		t.Fatalf("gUiw: %+v", cmds)
	}
}

// The doubled operator and an operand sharing a prefix must diverge correctly:
// "gUgg" is upper + goto-top, not the linewise double.
func TestMultiKeyOperatorSharedPrefixDiverges(t *testing.T) {
	m := opMatcher2(t)
	cmds := feed(t, m, "gUgg")
	if len(cmds) != 1 || cmds[0].Operator.Name != "upper" || cmds[0].Target.Name != "goto-top" || cmds[0].Linewise {
		t.Fatalf("gUgg: %+v", cmds)
	}
}

func TestMultiKeyOperatorDeadEndResets(t *testing.T) {
	m := opMatcher2(t)
	// gUq: q neither continues the operand nor the doubled operator → reset.
	if cmds := feed(t, m, "gUq"); len(cmds) != 0 {
		t.Fatalf("gUq: %+v", cmds)
	}
	if cmds := feed(t, m, "x"); len(cmds) != 1 || cmds[0].Target.Name != "del-char" {
		t.Fatalf("x after gUq: %+v", cmds)
	}
}
