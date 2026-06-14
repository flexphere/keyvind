package keyvind

import "sort"

// ConflictKind classifies a binding conflict reported by Conflicts.
type ConflictKind int

const (
	// ConflictAmbiguous marks a binding whose key sequence is a strict prefix of
	// one or more longer bindings in the same keymap. The binding still fires,
	// but only after the host's ambiguity timeout elapses (e.g. "g" when "gg"
	// also exists).
	ConflictAmbiguous ConflictKind = iota
	// ConflictOperatorShadow marks an operator binding that is a strict prefix of
	// longer bindings: the operator only activates after a timeout, and its
	// descendant sequences compete with operator input.
	ConflictOperatorShadow
	// ConflictArgShadow marks an AwaitsArg binding that has longer bindings
	// beneath it: those are unreachable, because the next key is captured as the
	// argument rather than matched.
	ConflictArgShadow
)

// String renders the ConflictKind for messages.
func (k ConflictKind) String() string {
	switch k {
	case ConflictAmbiguous:
		return "ambiguous"
	case ConflictOperatorShadow:
		return "operator-shadow"
	case ConflictArgShadow:
		return "arg-shadow"
	default:
		return "unknown"
	}
}

// Conflict describes a binding whose key sequence is a strict prefix of other
// bindings in the same mode, i.e. a terminal trie node that still has children.
type Conflict struct {
	Mode    string       // mode the conflict was found in ("" for a bare Keymap)
	Kind    ConflictKind // severity classification
	Keys    string       // the prefix binding's key sequence, in vim notation
	Name    string       // the prefix binding's handler name
	Extends []string     // handler names of the longer bindings that extend it
}

// Conflicts reports every binding in the keymap whose key sequence is a strict
// prefix of one or more longer bindings — a trie node that is both a terminal
// and a prefix, so typing it can only resolve after the ambiguity timeout (or,
// for an AwaitsArg binding, makes the longer bindings unreachable). The result
// is deterministic (sorted by keys).
func (km *Keymap) Conflicts() []Conflict {
	return km.conflictsForMode("")
}

func (km *Keymap) conflictsForMode(mode string) []Conflict {
	var out []Conflict
	var walk func(n *node)
	walk = func(n *node) {
		if n.binding != nil {
			if ext := descendantNames(n); len(ext) > 0 {
				kind := ConflictAmbiguous
				switch {
				case n.binding.AwaitsArg:
					kind = ConflictArgShadow
				case n.binding.isOperator():
					kind = ConflictOperatorShadow
				}
				out = append(out, Conflict{
					Mode:    mode,
					Kind:    kind,
					Keys:    n.binding.keyString(),
					Name:    n.binding.Name,
					Extends: ext,
				})
			}
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(km.root)
	sortConflicts(out)
	return out
}

// descendantNames returns the sorted handler names of every binding strictly
// below n (its extenders).
func descendantNames(n *node) []string {
	var names []string
	var walk func(c *node)
	walk = func(c *node) {
		if c.binding != nil {
			names = append(names, c.binding.Name)
		}
		for _, ch := range c.children {
			walk(ch)
		}
	}
	for _, ch := range n.children {
		walk(ch)
	}
	sort.Strings(names)
	return names
}

func sortConflicts(cs []Conflict) {
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Mode != cs[j].Mode {
			return cs[i].Mode < cs[j].Mode
		}
		return cs[i].Keys < cs[j].Keys
	})
}

// Conflicts reports prefix conflicts across every mode of the matcher, with each
// Conflict's Mode populated. The result is deterministic (sorted by mode, then
// keys).
func (m *Matcher) Conflicts() []Conflict {
	modes := make([]string, 0, len(m.modes))
	for name := range m.modes {
		modes = append(modes, name)
	}
	sort.Strings(modes)

	out := make([]Conflict, 0, len(modes))
	for _, name := range modes {
		out = append(out, m.modes[name].conflictsForMode(name)...)
	}
	return out
}
