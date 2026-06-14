package teakit_test

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/keyvind"
	"github.com/flexphere/keyvind/adapters/teakit"
)

// app is a minimal bubbletea model wired to a keyvind Driver.
type app struct {
	keys   *teakit.Driver
	status string
}

func (a app) Init() tea.Cmd { return nil }

// Update delegates key handling to the Driver: it feeds messages to the matcher,
// dispatches any resolved commands, and returns the Driver's tea.Cmd (the
// ambiguity-timeout tick) so vim-style "g vs gg" disambiguation works.
func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds, tick := a.keys.Update(msg)
	for _, c := range cmds {
		a = a.dispatch(c)
	}
	return a, tick
}

func (a app) View() string { return a.status }

// dispatch is where the host turns a resolved Command into an actual edit. The
// library only resolves intent (operator, target, count); the host owns the effect.
func (a app) dispatch(c keyvind.Command) app {
	switch {
	case c.Operator != nil:
		a.status = c.Operator.Name + " x" + itoa(c.Count) + " over " + c.Target.Name
	default:
		a.status = c.Target.Name + " x" + itoa(c.Count)
	}
	return a
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// Example shows the full wiring: register named bindings, build a Matcher, wrap
// it in a Driver, and hand the model to bubbletea.
func Example() {
	k := keyvind.NewKeymapper()
	k.Operator(keyvind.OperatorMapping{Mode: "normal", Keys: "d", Name: "delete", OperandMode: "motion"})
	k.Map(keyvind.Mapping{Mode: "normal", Keys: "x", Name: "delete_char"})
	k.Map(keyvind.Mapping{Mode: "motion", Keys: "w", Name: "word_fwd"})
	keymaps, _, _ := k.Build(keyvind.Key{})

	m := keyvind.NewMatcher(keymaps, "normal")
	model := app{keys: teakit.New(m)}

	// Hand to bubbletea (not run here):
	//   tea.NewProgram(model).Run()
	_ = model
}
