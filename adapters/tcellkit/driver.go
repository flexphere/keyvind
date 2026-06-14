package tcellkit

import (
	"time"

	"github.com/gdamore/tcell/v2"

	"github.com/flexphere/keyvind"
)

// DefaultTimeout mirrors vim's default 'timeoutlen' (1000ms): how long an
// ambiguous, complete-but-extendable sequence (e.g. "g" when "gg" also exists)
// waits for the next key before committing the shorter match.
const DefaultTimeout = 1000 * time.Millisecond

// TimeoutEvent is posted onto the tcell event loop when an armed ambiguity
// timeout elapses. The host receives it from PollEvent like any other event and
// passes it back to Driver.Update; the token guards against a stale timer that
// fired after a newer key already advanced the matcher.
type TimeoutEvent struct {
	when  time.Time
	token uint64
}

// When implements tcell.Event.
func (e *TimeoutEvent) When() time.Time { return e.when }

// Driver wraps a keyvind.Matcher for use inside a tcell event loop. Unlike
// bubbletea's command model, tcell is poll-based, so the Driver arms the
// ambiguity timeout itself with a timer that posts a TimeoutEvent back via the
// supplied post function (typically screen.PostEvent). It is not safe for
// concurrent use; drive it from the single event-loop goroutine.
type Driver struct {
	matcher *keyvind.Matcher
	timeout time.Duration
	post    func(tcell.Event) error
	token   uint64
}

// New returns a Driver over matcher, posting timeout events through post (pass
// screen.PostEvent). It uses DefaultTimeout.
func New(matcher *keyvind.Matcher, post func(tcell.Event) error) *Driver {
	return &Driver{matcher: matcher, timeout: DefaultTimeout, post: post}
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

// Update processes one tcell event. For a key event it feeds every decoded key
// to the matcher and arms the ambiguity timeout when needed; for a TimeoutEvent
// it resolves a held ambiguous match (ignoring stale timers). It returns the
// commands the host should dispatch. Other events yield nil.
func (d *Driver) Update(ev tcell.Event) []keyvind.Command {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		var cmds []keyvind.Command
		var res keyvind.Result
		for _, k := range FromEventKey(ev) {
			res = d.matcher.Feed(k)
			cmds = append(cmds, res.Commands...)
		}
		if res.ArmTimeout && d.timeout > 0 && d.post != nil {
			d.armTimeout()
		}
		return cmds
	case *TimeoutEvent:
		if ev.token != d.token {
			return nil // superseded by a newer key
		}
		return d.matcher.Timeout().Commands
	default:
		return nil
	}
}

func (d *Driver) armTimeout() {
	d.token++
	token := d.token
	post := d.post
	time.AfterFunc(d.timeout, func() {
		_ = post(&TimeoutEvent{when: time.Now(), token: token})
	})
}
