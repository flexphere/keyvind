package teakit

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/flexphere/keyvind"
)

// DefaultTimeout mirrors vim's default 'timeoutlen' (1000ms): how long an
// ambiguous, complete-but-extendable sequence (e.g. "g" when "gg" also exists)
// waits for the next key before committing the shorter match.
const DefaultTimeout = 1000 * time.Millisecond

// timeoutMsg is delivered by the Tick armed after an ambiguous match. The token
// guards against a stale timer firing after a newer key already advanced the
// matcher: only a token equal to the driver's current one is honored.
type timeoutMsg struct{ token uint64 }

// Driver wraps a keyvind.Matcher for use inside a bubbletea Update loop. It feeds
// converted key messages to the matcher and arms / resolves the ambiguity
// timeout via tea.Tick. It is not safe for concurrent use; drive it from the
// single Update goroutine.
type Driver struct {
	matcher *keyvind.Matcher
	timeout time.Duration
	token   uint64
}

// New returns a Driver over matcher using DefaultTimeout.
func New(matcher *keyvind.Matcher) *Driver {
	return &Driver{matcher: matcher, timeout: DefaultTimeout}
}

// WithTimeout sets the ambiguity timeout (vim 'timeoutlen'). A non-positive
// duration disables timed resolution: an ambiguous match is then held until the
// next key extends or diverts it.
func (d *Driver) WithTimeout(timeout time.Duration) *Driver {
	d.timeout = timeout
	return d
}

// Matcher exposes the underlying matcher (e.g. to query or change Mode).
func (d *Driver) Matcher() *keyvind.Matcher { return d.matcher }

// Update processes one bubbletea message. For a key message it feeds every
// decoded key to the matcher; for the internal timeout message it resolves a
// held ambiguous match (ignoring stale timers). It returns the commands the
// host should dispatch, plus an optional tea.Cmd that the host MUST return from
// its own Update so the timeout fires. Non-key messages yield (nil, nil).
func (d *Driver) Update(msg tea.Msg) ([]keyvind.Command, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmds []keyvind.Command
		var res keyvind.Result
		for _, k := range FromKeyMsg(msg) {
			res = d.matcher.Feed(k)
			cmds = append(cmds, res.Commands...)
		}
		if res.ArmTimeout && d.timeout > 0 {
			return cmds, d.armTimeout()
		}
		return cmds, nil
	case timeoutMsg:
		if msg.token != d.token {
			return nil, nil // superseded by a newer key
		}
		return d.matcher.Timeout().Commands, nil
	default:
		return nil, nil
	}
}

func (d *Driver) armTimeout() tea.Cmd {
	d.token++
	token := d.token
	return tea.Tick(d.timeout, func(time.Time) tea.Msg {
		return timeoutMsg{token: token}
	})
}
