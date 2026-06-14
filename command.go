package keyvind

import "strings"

// Binding is a single key sequence mapped to a named handler, registered in a
// mode's Keymap. There is no editor-role taxonomy: a binding is just a terminal
// command, unless it is an operator (OperandMode set) or captures an argument
// (AwaitsArg). Whether a binding is "standalone" or an "operand" is expressed by
// which mode's keymap it lives in — not by the binding itself.
type Binding struct {
	Keys []Key  // the key sequence that triggers this binding
	Name string // handler name the host application dispatches on

	// OperandMode, when non-empty, makes this binding an operator: resolving it
	// switches matching into that mode's keymap to read the operand, then
	// composes one Command (e.g. "d" with OperandMode "motion" → "dw"/"diw").
	// Multiple operators may share one operand mode (so the operands are stored
	// once, not per operator).
	OperandMode string

	// AwaitsArg makes the binding capture the next raw keystroke as Command.Arg
	// before the command is finalized — the vim character-argument commands like
	// f/F/t/T (find char) or r (replace char). It composes with operators and
	// counts, and arms no timeout while waiting; an <Esc> argument cancels.
	AwaitsArg bool

	// Desc is an optional human-readable description of what the binding does.
	// The engine never interprets it; it is carried through to the resolved
	// Command (via Operator/Target) so a host can drive help screens, a
	// which-key-style popup, or logging.
	Desc string
}

func (b *Binding) isOperator() bool { return b.OperandMode != "" }

func (b *Binding) keyString() string {
	var sb strings.Builder
	for _, k := range b.Keys {
		sb.WriteString(k.String())
	}
	return sb.String()
}

// Command is a fully resolved user intent produced by the Matcher. The host
// application dispatches on it; keyvind never performs the edit itself.
//
// Shapes:
//
//	"x"    -> {Count:1, Target:x}
//	"3j"   -> {Count:3, Target:j}
//	"3dw"  -> {Count:3, Operator:d, Target:w}
//	"dd"   -> {Count:1, Operator:d, Target:d, Linewise:true}
//	"diw"  -> {Count:1, Operator:d, Target:iw}
//	"fx"   -> {Count:1, Target:f, Arg:'x'}
type Command struct {
	Mode     string   // mode the command was resolved in
	Count    int      // resolved repeat count, always >= 1
	Operator *Binding // nil unless this is an operator+operand command
	Target   *Binding // the terminal / operand that completes the command
	Linewise bool     // true for a doubled operator (dd, yy)
	Keys     []Key    // raw keys consumed to produce this command

	// Arg is the captured argument keystroke for an AwaitsArg binding (e.g. the
	// char after "f"/"r"). HasArg reports whether one was captured.
	Arg    Key
	HasArg bool
}

// Result is returned by Feed and Timeout. A single keystroke can finalize zero
// or more commands, leave the matcher pending, or request a timeout timer.
type Result struct {
	// Commands are the resolved commands, in the order they were finalized.
	Commands []Command
	// Pending is true when the matcher is mid-sequence and awaiting more keys.
	Pending bool
	// ArmTimeout is true when a complete-but-extendable match is being held.
	// The host should start a timer and call Timeout when it elapses; an
	// intervening Feed that extends the match cancels the need for it.
	ArmTimeout bool
}

func (r Result) withCommand(c Command) Result {
	r.Commands = append(r.Commands, c)
	return r
}
