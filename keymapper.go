package keyvind

import (
	"fmt"
	"sort"
	"strings"
)

// Mapping adds a terminal binding to a mode's keymap: a key sequence that
// resolves to a command on its own (within that mode). Whether it acts as a
// standalone command or as an operator's operand is purely which mode it is in.
// A Mapping with an empty Name unmaps that key sequence.
type Mapping struct {
	Mode      string // which mode's keymap (e.g. "normal", "motion", "visual")
	Keys      string // vim notation, e.g. "iw", "<leader>w", "qas"
	Name      string // handler name ("" unmaps)
	AwaitsArg bool   // capture the next keystroke as Command.Arg (f/F/t/T, r, ...)
	Desc      string // optional human-readable description (carried to Binding.Desc)
}

// OperatorMapping adds an operator: a binding that, when resolved, switches
// matching into OperandMode's keymap to read the operand and composes one
// Command (e.g. "d" → operand mode "motion" → "dw"/"diw").
type OperatorMapping struct {
	Mode        string // mode the operator lives in (e.g. "normal")
	Keys        string // e.g. "d"
	Name        string // e.g. "delete"
	OperandMode string // mode whose keymap holds the operands (e.g. "motion")
	Desc        string // optional human-readable description (carried to Binding.Desc)
}

type entry struct {
	mode, keys, name, operandMode, desc string
	awaitsArg                           bool
}

// Keymapper is a declarative, dispatch-free keymap builder. Add bindings with
// Map (terminals) and Operator (operators) across arbitrary modes, then Build
// per-mode keymaps. It owns no handlers and knows no editor roles, so a host
// dispatches resolved Commands however it likes (a switch, a table, ...) and
// layers any vim-flavored vocabulary or default keymap on top.
type Keymapper struct {
	entries []entry
}

// NewKeymapper returns an empty Keymapper.
func NewKeymapper() *Keymapper {
	return &Keymapper{}
}

// Map adds a terminal binding. Calls accumulate; a later call for the same
// (mode, key sequence) overrides an earlier one (last wins). Returns the
// Keymapper for chaining.
func (k *Keymapper) Map(m Mapping) *Keymapper {
	k.entries = append(k.entries, entry{m.Mode, m.Keys, m.Name, "", m.Desc, m.AwaitsArg})
	return k
}

// Operator adds an operator binding.
func (k *Keymapper) Operator(m OperatorMapping) *Keymapper {
	k.entries = append(k.entries, entry{m.Mode, m.Keys, m.Name, m.OperandMode, m.Desc, false})
	return k
}

// Build resolves the accumulated mappings into per-mode keymaps, applying
// last-wins overrides and unmap (empty Name). It returns the keymaps and any
// prefix conflicts (sorted by mode then keys). An error is returned for an
// unparseable key sequence or a duplicate exact binding.
func (k *Keymapper) Build(leader Key) (map[string]*Keymap, []Conflict, error) {
	type mk struct{ mode, keys string }
	resolved := make(map[mk]entry)
	var order []mk

	for _, e := range k.entries {
		keys, err := ParseKeys(e.keys, leader)
		if err != nil {
			return nil, nil, fmt.Errorf("keyvind: mapping %q: %w", e.keys, err)
		}
		id := mk{e.mode, keyString(keys)}
		if _, seen := resolved[id]; !seen {
			order = append(order, id)
		}
		resolved[id] = e // last wins
	}

	keymaps := make(map[string]*Keymap)
	for _, id := range order {
		e := resolved[id]
		if e.name == "" {
			continue // unmapped
		}
		keys, _ := ParseKeys(e.keys, leader) // already validated above
		km := keymaps[e.mode]
		if km == nil {
			km = NewKeymap()
			keymaps[e.mode] = km
		}
		if err := km.Add(&Binding{Keys: keys, Name: e.name, OperandMode: e.operandMode, AwaitsArg: e.awaitsArg, Desc: e.desc}); err != nil {
			return nil, nil, fmt.Errorf("keyvind: %w", err)
		}
	}

	var conflicts []Conflict
	modes := make([]string, 0, len(keymaps))
	for mode := range keymaps {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	for _, mode := range modes {
		for _, c := range keymaps[mode].Conflicts() {
			c.Mode = mode
			conflicts = append(conflicts, c)
		}
	}

	return keymaps, conflicts, nil
}

func keyString(keys []Key) string {
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k.String())
	}
	return b.String()
}
