// Command tview demonstrates keyvind driving vim-like keys in a tview app via
// the tcellkit adapter.
//
// tview hands key events to SetInputCapture as *tcell.EventKey, so tcellkit's
// FromEventKey/Driver work directly. Because tview owns its own event loop, the
// ambiguity timeout is fed back through app.QueueUpdateDraw (see main): the
// Driver's post function re-delivers the timeout event on the UI goroutine.
//
// It is deliberately minimal: a handful of unrelated commands, no text editing.
// Press a key sequence and the demo logs which command(s) resolved.
//
//	p s         simple single-key commands       (count-aware, e.g. 3p)
//	gg          a command built from two keys
//	g           "g-solo" — fires only after the ambiguity timeout (g vs gg)
//	<Space>w    a leader command  (<leader> = Space)
//	<Space>r    another leader command
//	<C-s>       a Ctrl-modified command
//	<S-Tab>     a Shift-modified command
//	<C-x><C-s>  a Ctrl chord (two modifier keys in sequence)
//	q / ctrl+c  quit
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/flexphere/keyvind"
	"github.com/flexphere/keyvind/adapters/tcellkit"
)

const logMax = 6

type app struct {
	tapp   *tview.Application
	view   *tview.TextView
	driver *tcellkit.Driver
	log    []string
}

func main() {
	a := &app{
		tapp: tview.NewApplication(),
		view: tview.NewTextView().SetDynamicColors(true),
		log:  []string{"(press keys: p, 3p, gg, g, <Space>w, <C-s>, <S-Tab>)"},
	}

	a.driver = tcellkit.New(newMatcher(), func(ev tcell.Event) error {
		// The timeout fires on a timer goroutine; re-deliver it to the Driver on
		// the UI goroutine via QueueUpdateDraw.
		a.tapp.QueueUpdateDraw(func() {
			for _, c := range a.driver.Update(ev) {
				a.dispatch(c)
			}
		})
		return nil
	})

	a.view.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyCtrlC || (ev.Key() == tcell.KeyRune && ev.Rune() == 'q') {
			a.tapp.Stop()
			return nil
		}
		for _, c := range a.driver.Update(ev) {
			a.dispatch(c)
		}
		a.render()
		return nil
	})

	a.render()
	if err := a.tapp.SetRoot(a.view, true).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newMatcher() *keyvind.Matcher {
	leader := keyvind.Key{Name: "space"}

	// Build the keymap declaratively with Keymapper: key sequences stay strings
	// and the leader is supplied once at Build.
	km := keyvind.NewKeymapper()

	// simple single-key commands
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "p", Name: "play"})
	km.Map(keyvind.Mapping{Mode: "normal", Keys: "s", Name: "stop"})

	// a command built from a two-key sequence; the lone "g" is ambiguous with
	// it, so "g" alone commits only on the timeout
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
	return keyvind.NewMatcher(keymaps, "normal")
}

func (a *app) dispatch(c keyvind.Command) {
	a.log = append(a.log, describe(c))
	if len(a.log) > logMax {
		a.log = a.log[len(a.log)-logMax:]
	}
	a.render()
}

func (a *app) render() {
	var b strings.Builder
	b.WriteString("keyvind + tview demo (normal mode)\n\n  resolved commands:\n")
	for _, e := range a.log {
		b.WriteString("    [green]→[-] " + e + "\n")
	}
	b.WriteString("\n  p s play/stop (count: 3p) | gg go-first / g (wait) | <Space>w write\n")
	b.WriteString("  <C-s> save | <S-Tab> back-tab | <C-x><C-s> save-all | q / ctrl+c quit")
	a.view.SetText(b.String())
}

func describe(c keyvind.Command) string {
	if c.Count != 1 {
		return fmt.Sprintf("%s x%d", c.Target.Name, c.Count)
	}
	return c.Target.Name
}
