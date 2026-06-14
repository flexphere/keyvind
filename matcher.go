package keyvind

// Matcher is the per-application state machine. Feed it decoded Keys; it walks
// the active mode's Keymap, applies the grammar (count, operator+operand via
// a mode switch, doubled operators, argument capture), and emits resolved
// Commands.
//
// Composition has no editor knowledge: an operator switches matching into its
// OperandMode keymap, reads the operand there, and composes one Command. Whether
// a binding is standalone or an operand is simply which mode's keymap it is in.
//
// A Matcher is not safe for concurrent use; drive it from a single goroutine
// (in bubbletea, the Update loop).
type Matcher struct {
	modes map[string]*Keymap
	mode  string // host-visible mode

	// in-flight command state
	cur      *node    // cursor in the active keymap (root between keys; nil = operand branch dead, doubling-only)
	keys     []Key    // raw keys consumed for the in-flight command
	count1   int      // count before the operator (0 = unset)
	op       *Binding // pending operator, nil when none
	count2   int      // count after the operator (0 = unset)
	dbl      int      // while op-pending: how many of op.Keys have been re-typed (-1 = broken)
	awaiting *Command // command awaiting its captured argument (AwaitsArg), or nil
}

// NewMatcher builds a Matcher over the given modes, starting in initialMode.
func NewMatcher(modes map[string]*Keymap, initialMode string) *Matcher {
	m := &Matcher{modes: modes, mode: initialMode}
	m.resetSequence()
	return m
}

// Mode returns the active (host-visible) mode.
func (m *Matcher) Mode() string { return m.mode }

// SetMode switches the active mode and discards any in-flight sequence. The
// host calls this from a Command handler that changes mode.
func (m *Matcher) SetMode(mode string) {
	m.mode = mode
	m.resetAll()
}

// SetKeymaps swaps the mode keymaps and discards any in-flight sequence, keeping
// the current mode. Use it to hot-reload bindings: rebuild with Keymapper.Build
// (which resolves overrides and unmap) and hand the result here instead of
// constructing a new Matcher. If the current mode is absent from the new set,
// Feed no-ops until SetMode selects an existing mode.
func (m *Matcher) SetKeymaps(modes map[string]*Keymap) {
	m.modes = modes
	m.resetAll()
}

// keymap returns the keymap currently being matched: the operand mode's keymap
// while an operator is pending, otherwise the active mode's keymap.
func (m *Matcher) keymap() *Keymap {
	if m.op != nil {
		return m.modes[m.op.OperandMode]
	}
	return m.modes[m.mode]
}

func (m *Matcher) resetSequence() {
	if km := m.keymap(); km != nil {
		m.cur = km.root
	} else {
		m.cur = nil // no keymap for the active mode; Feed will no-op
	}
}

func (m *Matcher) resetAll() {
	m.op = nil
	m.count1 = 0
	m.count2 = 0
	m.dbl = -1
	m.awaiting = nil
	m.keys = nil
	m.resetSequence() // now resolves to the base mode's keymap
}

// atRoot reports whether the cursor is between keys in the active keymap.
func (m *Matcher) atRoot() bool {
	km := m.keymap()
	return km != nil && m.cur == km.root
}

// Feed advances the state machine by one key and returns the resulting commands
// and pending status.
func (m *Matcher) Feed(k Key) Result {
	if m.keymap() == nil {
		return Result{}
	}
	var res Result
	for {
		consumed, out, done := m.step(k, &res)
		res = out
		if consumed || done {
			return res
		}
		// not consumed: a held match committed; reprocess the same key.
	}
}

func (m *Matcher) step(k Key, res *Result) (bool, Result, bool) {
	// A pending AwaitsArg command captures this key as its argument.
	if m.awaiting != nil {
		return m.captureArg(k, res)
	}

	// Count digits only matter between keys (cursor at root).
	if m.atRoot() && m.isCountDigit(k) {
		m.addCount(k)
		m.keys = append(m.keys, k)
		r := *res
		r.Pending = true
		return true, r, true
	}

	if m.op != nil {
		return m.stepOperand(k, res)
	}
	return m.stepKey(k, res)
}

// stepKey advances the (non-operator) walk in the active mode's keymap.
func (m *Matcher) stepKey(k Key, res *Result) (bool, Result, bool) {
	next := m.cur.children[k]
	if next == nil {
		return m.handleDeadEnd(res)
	}
	m.cur = next
	m.keys = append(m.keys, k)
	return m.resolveNode(res)
}

// stepOperand advances the operand walk while an operator is pending, tracking in
// parallel the operator's own key sequence: re-typing it in full means linewise
// (dd/cc/yy, but also multi-key operators like gUgU). The operand trie and the
// doubling sequence can share a prefix (e.g. a "gg" motion vs the "gU" operator),
// so both are followed until one wins.
func (m *Matcher) stepOperand(k Key, res *Result) (bool, Result, bool) {
	dblNext := -1
	if m.dbl >= 0 && m.dbl < len(m.op.Keys) && k == m.op.Keys[m.dbl] {
		dblNext = m.dbl + 1
	}
	var operandNext *node
	if m.cur != nil {
		operandNext = m.cur.children[k]
	}

	// The operator's whole key sequence was re-typed and the operand branch is
	// exhausted ⇒ linewise (dd, gUgU, …).
	if dblNext == len(m.op.Keys) && operandNext == nil {
		m.keys = append(m.keys, k)
		cmd := Command{
			Mode:     m.mode,
			Operator: m.op,
			Target:   m.op,
			Linewise: true,
			Count:    resolveCount(m.count1, m.count2),
			Keys:     m.keys,
		}
		m.resetAll()
		return true, res.withCommand(cmd), true
	}

	// Neither branch can continue ⇒ a held operand terminal commits, else fail.
	if operandNext == nil && dblNext < 0 {
		if m.cur == nil { // doubling-only branch broke mid-sequence
			m.resetAll()
			return true, *res, true
		}
		return m.handleDeadEnd(res)
	}

	m.dbl = dblNext
	m.keys = append(m.keys, k)
	if operandNext == nil {
		// Only the doubling sequence is still alive (its prefix is not an operand);
		// hold until it completes or breaks.
		m.cur = nil
		r := *res
		r.Pending = true
		return true, r, true
	}
	m.cur = operandNext
	return m.resolveNode(res)
}

// resolveNode handles the binding at the freshly advanced cursor m.cur: pending,
// held-with-timeout, finalized, or a dead end.
func (m *Matcher) resolveNode(res *Result) (bool, Result, bool) {
	b := m.cur.binding
	extendable := len(m.cur.children) > 0
	if b != nil && b.AwaitsArg {
		// the next key is captured as the argument, so children are unreachable
		extendable = false
	}

	switch {
	case b == nil && extendable:
		r := *res
		r.Pending = true
		return true, r, true
	case b == nil && !extendable:
		m.resetAll()
		return true, *res, true
	case b != nil && extendable:
		// complete but extendable: hold and ask for a timeout
		r := *res
		r.Pending = true
		r.ArmTimeout = true
		return true, r, true
	default: // b != nil && !extendable
		return true, m.resolve(b, *res), true
	}
}

// handleDeadEnd is reached when the active cursor has no child for the incoming
// key. If the cursor sits on a held terminal, commit it and signal the caller
// (consumed=false) to reprocess the key; otherwise the sequence fails.
func (m *Matcher) handleDeadEnd(res *Result) (bool, Result, bool) {
	if !m.atRoot() && m.cur.binding != nil {
		r := m.resolve(m.cur.binding, *res)
		return false, r, false
	}
	m.resetAll()
	return true, *res, true
}

// resolve turns a matched terminal binding into an operator-pending transition,
// an argument capture, or a finalized Command appended to res.
func (m *Matcher) resolve(b *Binding, res Result) Result {
	if m.op == nil && b.isOperator() {
		// Enter operator-pending: subsequent keys are matched in the operand mode.
		m.op = b
		m.dbl = 0         // start tracking a re-typed operator (linewise)
		m.resetSequence() // cursor → operand keymap root (keys/count kept)
		res.Pending = true
		return res
	}

	cmd := Command{Mode: m.mode, Keys: m.keys}
	if m.op != nil {
		cmd.Operator = m.op
		cmd.Target = b
		cmd.Count = resolveCount(m.count1, m.count2)
	} else {
		cmd.Target = b
		cmd.Count = resolveCount(m.count1, 0)
	}

	if b.AwaitsArg {
		m.resetAll()
		m.awaiting = &cmd
		res.Pending = true
		return res
	}

	m.resetAll()
	return res.withCommand(cmd)
}

// captureArg consumes k as the argument of a pending AwaitsArg command. An <Esc>
// cancels it (no command emitted); any other key becomes Command.Arg.
func (m *Matcher) captureArg(k Key, res *Result) (bool, Result, bool) {
	cmd := m.awaiting
	m.resetAll() // also clears m.awaiting
	if k.Name == "esc" {
		return true, *res, true
	}
	cmd.Arg = k
	cmd.HasArg = true
	cmd.Keys = append(cmd.Keys, k)
	return true, res.withCommand(*cmd), true
}

// Timeout commits a held ambiguous match (the binding at the cursor). The host
// calls it when the timer armed by ArmTimeout elapses.
func (m *Matcher) Timeout() Result {
	if m.cur == nil || m.atRoot() || m.cur.binding == nil {
		return Result{}
	}
	return m.resolve(m.cur.binding, Result{})
}

func (m *Matcher) isCountDigit(k Key) bool {
	if k.Ctrl || k.Alt || k.Name != "" {
		return false
	}
	if k.Rune < '0' || k.Rune > '9' {
		return false
	}
	// A leading '0' is the "first column" motion, not a count.
	if k.Rune == '0' && m.currentCount() == 0 {
		return false
	}
	return true
}

func (m *Matcher) currentCount() int {
	if m.op == nil {
		return m.count1
	}
	return m.count2
}

func (m *Matcher) addCount(k Key) {
	d := int(k.Rune - '0')
	if m.op == nil {
		m.count1 = m.count1*10 + d
	} else {
		m.count2 = m.count2*10 + d
	}
}

// resolveCount combines the pre- and post-operator counts. Missing counts
// default to 1; counts multiply (2d3w == 6).
func resolveCount(c1, c2 int) int {
	a, b := c1, c2
	if a == 0 {
		a = 1
	}
	if b == 0 {
		b = 1
	}
	return a * b
}
