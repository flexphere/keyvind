// Command bubbletea is an interactive demo of keyvind driving vim-like keys in
// a bubbletea TUI via the teakit adapter.
//
// It is deliberately minimal: a handful of unrelated commands, no text editing.
// Press a key sequence and the demo logs which command(s) resolved.
//
//	p s         simple single-key commands       (count-aware, e.g. 3p)
//	gg          a command built from two keys
//	g           "g-solo" — fires only after the ambiguity timeout, showing the
//	            tea.Tick that teakit arms to disambiguate g vs gg
//	<Space>w    a leader command  (<leader> = Space)
//	<Space>r    another leader command
//	<C-s>       a Ctrl-modified command
//	<S-Tab>     a Shift-modified command
//	<C-x><C-s>  a Ctrl chord (two modifier keys in sequence)
//	q / ctrl+c  quit
//
// Note: bubbletea v1 cannot distinguish Ctrl+Shift+letter from Ctrl+letter; the
// teakitv2 / tcellkit adapters can on terminals with an enhanced keyboard
// protocol. The combos used here work everywhere.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/keyvind"
	"github.com/flexphere/keyvind/adapters/teakit"
)

const logMax = 6

type model struct {
	log  []string
	keys *teakit.Driver
}

func initialModel() model {
	leader := keyvind.Key{Name: "space"}

	// Build the keymap declaratively with Keymapper: key sequences stay strings
	// and the leader is supplied once at Build.
	km := keyvind.NewKeymapper()

	// simple single-key commands
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "p", Name: "play"})
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "s", Name: "stop"})

	// a command built from a two-key sequence; the lone "g" is ambiguous with
	// it, so "g" alone commits only on the timeout (teakit's tea.Tick)
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "gg", Name: "go-first"})
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "g", Name: "g-solo"})

	// leader commands (<leader> expands to the leader key supplied to Build)
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "<leader>w", Name: "write"})
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "<leader>r", Name: "reload"})

	// modifier-key commands: Ctrl, Shift, and a Ctrl chord (two keys in sequence)
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "<C-s>", Name: "save"})
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "<S-Tab>", Name: "back-tab"})
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "<C-x><C-s>", Name: "save-all"})

	keymaps, _, err := km.Build(leader)
	if err != nil {
		panic(err)
	}
	m := keyvind.NewMatcher(keymaps, "normal")
	return model{
		keys: teakit.New(m),
		log:  []string{"(press keys: p, 3p, gg, g, <Space>w, <C-s>, <S-Tab>)"},
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		if k.Type == tea.KeyCtrlC || (k.Type == tea.KeyRunes && string(k.Runes) == "q") {
			return m, tea.Quit
		}
	}
	cmds, tick := m.keys.Update(msg)
	for _, c := range cmds {
		m = m.push(describe(c))
	}
	return m, tick
}

// push appends a line to the bounded command log.
func (m model) push(s string) model {
	next := append(append([]string(nil), m.log...), s)
	if len(next) > logMax {
		next = next[len(next)-logMax:]
	}
	m.log = next
	return m
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("keyvind + bubbletea demo (normal mode)\n\n  resolved commands:\n")
	for _, e := range m.log {
		b.WriteString("    → " + e + "\n")
	}
	b.WriteString("\n  p s play/stop (count: 3p) | gg go-first / g (wait) | <Space>w write\n")
	b.WriteString("  <C-s> save | <S-Tab> back-tab | <C-x><C-s> save-all | q / ctrl+c quit\n")
	return b.String()
}

func describe(c keyvind.Command) string {
	if c.Count != 1 {
		return fmt.Sprintf("%s x%d", c.Target.Name, c.Count)
	}
	return c.Target.Name
}

func main() {
	if _, err := tea.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
