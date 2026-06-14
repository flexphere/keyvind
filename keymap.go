package keyvind

import (
	"fmt"
	"sort"
)

// node is a trie node keyed by Key. A binding terminates at the node reached by
// walking its key sequence.
type node struct {
	children map[Key]*node
	binding  *Binding
}

func newNode() *node {
	return &node{children: make(map[Key]*node)}
}

// Keymap is the set of bindings for a single mode, stored as a trie so that
// prefix relationships (e.g. "g" vs "gg") are explicit.
type Keymap struct {
	root *node
}

// NewKeymap returns an empty Keymap.
func NewKeymap() *Keymap {
	return &Keymap{root: newNode()}
}

// Add inserts a binding. It returns an error if the exact key sequence is
// already bound (a duplicate); use Conflicts for non-fatal static analysis.
func (km *Keymap) Add(b *Binding) error {
	if len(b.Keys) == 0 {
		return fmt.Errorf("keyvind: binding %q has empty key sequence", b.Name)
	}
	n := km.root
	for _, k := range b.Keys {
		child, ok := n.children[k]
		if !ok {
			child = newNode()
			n.children[k] = child
		}
		n = child
	}
	if n.binding != nil {
		return fmt.Errorf("keyvind: duplicate binding for %q: %q and %q",
			b.keyString(), n.binding.Name, b.Name)
	}
	n.binding = b
	return nil
}

// MustAdd is Add that panics on error, for static binding tables.
func (km *Keymap) MustAdd(b *Binding) {
	if err := km.Add(b); err != nil {
		panic(err)
	}
}

// Bindings returns every binding in the keymap, sorted by key sequence. It walks
// the trie so a host can render a help screen, a which-key-style popup, or a
// cheat sheet (each Binding carries its Name and optional Desc).
func (km *Keymap) Bindings() []*Binding {
	var out []*Binding
	var walk func(n *node)
	walk = func(n *node) {
		if n.binding != nil {
			out = append(out, n.binding)
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(km.root)
	sort.Slice(out, func(i, j int) bool { return out[i].keyString() < out[j].keyString() })
	return out
}
