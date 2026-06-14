# keyvind

[![CI](https://github.com/flexphere/keyvind/actions/workflows/ci.yml/badge.svg)](https://github.com/flexphere/keyvind/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/flexphere/keyvind.svg)](https://pkg.go.dev/github.com/flexphere/keyvind)

Vim-like key-sequence bindings for Go TUI applications.

`keyvind` resolves a stream of keystrokes — counts, operators, motions, text
objects, custom modes — into structured commands your app then acts on.
It pairs naturally with [bubbletea](https://github.com/charmbracelet/bubbletea) but depends on no UI framework.

```
[]Key ──▶ Matcher (state machine) ──▶ Command ──▶ your handler
```

## Design

- **A grammar engine, not an editor.** `keyvind` turns `3dw` into
  `Command{Count: 3, Operator: "delete", Target: "word"}` and stops there.
  Computing the affected range and performing the edit is your app's job. This
  keeps the engine independent of your data model and easy to test.
- **No editor vocabulary anywhere in the library.** It knows modes, keymaps, and
  how to discriminate a command from a key sequence — nothing about cursors,
  selection, yank, or undo. The familiar vim command names and default keymaps
  belong in the host application, not here.
- **Key sequences vs. actions are separate.** Code registers _named_ handlers
  (`delete`, `word`, `goto-top`); a layer above maps _key sequences_ to those
  names — so bindings can be overridden without touching the actions.
- **Modes are user-defined.** There is no hard-coded `normal`/`insert`/`visual`;
  a mode is just a namespace with its own keymap. An operator names the mode it
  reads its operand from.
- **Framework-agnostic core, zero dependencies.** The core imports only the
  standard library. bubbletea support lives in the separate `adapters/teakit`
  module, so importing the core never pulls in a UI framework.

## Install

```sh
go get github.com/flexphere/keyvind
```

For a UI adapter (pick the one matching your stack):

```sh
go get github.com/flexphere/keyvind/adapters/teakit    # bubbletea v1
go get github.com/flexphere/keyvind/adapters/teakitv2  # bubbletea v2 (Go 1.25+)
go get github.com/flexphere/keyvind/adapters/tcellkit  # tcell (and tview, cview, ...)
```

## The model: bindings and operand modes

A binding is just:

```go
type Binding struct {
	Keys        []Key
	Name        string
	OperandMode string // "" = terminal; non-empty = operator
	AwaitsArg   bool   // capture the next keystroke as Command.Arg
	Desc        string // optional description, carried through to the Command
}
```

There are no grammatical "kinds". A binding's role is **where it lives**:

- A **terminal** binding (`OperandMode == ""`) resolves to a command on its own.
- An **operator** binding (`OperandMode != ""`) resolves by switching matching
  into the named operand mode's keymap, reading one operand there, and composing
  a single `Command` carrying both `Operator` and `Target`.

So "motion" vs "text object" is not a tag — it is which keymap the binding is in:

- a key in **both** the active mode and an operand mode acts standalone _and_ as
  an operand (vim's nmap + omap, e.g. `w`);
- a key only in an operand mode is operand-only (a text object, e.g. `iw`);
- a key only in the active mode is a plain command (e.g. `x`).

Two uniform, first-class properties round it out:

- **Doubled operator** → linewise: re-typing an operator's own key sequence
  (`dd`, `yy`, and multi-key operators like `gUgU`) resolves with
  `Linewise: true`.
- **`AwaitsArg`** → the next raw keystroke is captured as `Command.Arg` instead
  of being matched (`f`/`F`/`t`/`T`, `r`).

This avoids any combinatorial blow-up: N operators × M operands cost `N + M`
bindings (operands are stored once, shared by every operator), not `N × M`.

## Quick start (core)

```go
package main

import (
	"fmt"

	"github.com/flexphere/keyvind"
)

func main() {
	// normal mode: standalone commands plus the "d" operator, which reads its
	// operand from mode "o".
	normal := keyvind.NewKeymap()
	normal.MustAdd(&keyvind.Binding{Keys: ks("x"), Name: "del-char"})
	normal.MustAdd(&keyvind.Binding{Keys: ks("w"), Name: "word"})
	normal.MustAdd(&keyvind.Binding{Keys: ks("d"), Name: "delete", OperandMode: "o"})

	// operand mode "o": what an operator can act over (motions + text objects).
	o := keyvind.NewKeymap()
	o.MustAdd(&keyvind.Binding{Keys: ks("w"), Name: "word"})        // shared with normal
	o.MustAdd(&keyvind.Binding{Keys: ks("iw"), Name: "inner-word"}) // operand-only

	m := keyvind.NewMatcher(map[string]*keyvind.Keymap{"normal": normal, "o": o}, "normal")

	for _, k := range ks("2d3w") {
		for _, c := range m.Feed(k).Commands {
			fmt.Printf("%s x%d over %s\n", c.Operator.Name, c.Count, c.Target.Name)
			// -> delete x6 over word
		}
	}
}

func ks(s string) []keyvind.Key {
	keys, err := keyvind.ParseKeys(s, keyvind.Key{})
	if err != nil {
		panic(err)
	}
	return keys
}
```

## Concepts

### Resolved command

`Feed` returns a `Result`; each finalized `Command` looks like:

| Input | Command                                              |
| ----- | ---------------------------------------------------- |
| `x`   | `{Count: 1, Target: x}`                              |
| `3j`  | `{Count: 3, Target: j}`                              |
| `3dw` | `{Count: 3, Operator: d, Target: w}`                 |
| `dd`  | `{Count: 1, Operator: d, Target: d, Linewise: true}` |
| `diw` | `{Count: 1, Operator: d, Target: iw}`                |
| `fx`  | `{Count: 1, Target: f, Arg: 'x'}`                    |

Counts before and after an operator multiply (`2d3w` → count 6).

### Key notation

`ParseKeys` accepts vim-style specs: literal runes, `<leader>`, `<C-x>`,
`<S-Tab>`, `<CR>`, `<Esc>`, `<Space>`, etc.

### Character-argument commands (`f` `F` `t` `T`, `r`, ...)

Set `Binding.AwaitsArg` to capture the next raw keystroke as `Command.Arg`
instead of matching it against the keymap — the vim commands that take a
character argument: `f`/`F`/`t`/`T` (find char), `r` (replace char), and so on.
It composes with counts and operators (`3fx`, `dfx`), arms no timeout (it waits
indefinitely for the char), and an `<Esc>` argument cancels.

```go
km.MustAdd(&keyvind.Binding{Keys: ks("f"), Name: "find", AwaitsArg: true})
// "dfx" -> Command{Operator: d, Target: find, Arg: 'x'}
```

Line-input commands (`:` ex, `/` search) are intentionally out of scope — those
are a host text-input mode, not a single-key argument.

### Modes

A mode is a namespace with its own keymap (its own trie). `Matcher.SetMode`
switches the active mode; an operator switches into its `OperandMode` only for
the duration of reading the operand. Nothing in the core ascribes meaning to a
mode's name — `"normal"`, `"o"`, `"visual"` are all just strings the host picks.

### Ambiguity & timeout

When a sequence is both a complete match and a prefix of a longer one (`g` vs
`gg`) _within the same keymap_, the engine holds it and returns
`Result{Pending, ArmTimeout}` instead of committing. The host resolves it one of
three ways:

1. **extend** — the next key forms the longer binding (`gg`)
2. **divert** — the next key doesn't extend it: the short match commits, then
   the new key is reprocessed
3. **time out** — the host calls `Matcher.Timeout()` when its timer elapses

The engine never blocks; it only signals `ArmTimeout`. (`teakit` arms the timer
for you via `tea.Tick`.) Note that `i` (a command) and `iw` (an operand) no
longer collide — they live in different keymaps, so there is no ambiguity to
arbitrate.

### Conflict detection

`Keymap.Conflicts()` / `Matcher.Conflicts()` statically report any binding whose
key sequence is a strict prefix of a longer one _in the same keymap_ — i.e. a
terminal trie node that still has children. This catches both `g`+`gg` and
"binding `d` after `dw`" (binding onto a non-terminal), independent of insertion
order.

```go
for _, c := range m.Conflicts() {
	fmt.Printf("[%s] %q extended by %v\n", c.Kind, c.Keys, c.Extends)
}
```

Kinds are `ConflictAmbiguous`, `ConflictOperatorShadow`, and `ConflictArgShadow`.
Exact duplicates are rejected eagerly by `Keymap.Add`.

## Bindings as data (`Keymapper`)

`keyvind.Keymapper` builds keymaps declaratively. Add terminal bindings with `Map`
and operators with `Operator`, across arbitrary modes, then `Build` per-mode
keymaps. It separates _key sequences_ (data, which a user can override) from the
_vocabulary_ (handler names), resolves last-wins overrides and `unmap` (empty
`Name`), and reports prefix conflicts. It owns no handlers, so a host with its
own dispatch uses it directly.

```go
km := keyvind.NewKeymapper()

// defaults shipped in code
km.Map(keyvind.Mapping{Mode: "normal", Keys: "l", Name: "right"})
km.Map(keyvind.Mapping{Mode: "normal", Keys: "x", Name: "del-char", Desc: "delete char"})
km.Map(keyvind.Mapping{Mode: "normal", Keys: "f", Name: "find", AwaitsArg: true})
km.Operator(keyvind.OperatorMapping{Mode: "normal", Keys: "d", Name: "delete", OperandMode: "o"})
km.Map(keyvind.Mapping{Mode: "o", Keys: "w", Name: "word"})
km.Map(keyvind.Mapping{Mode: "o", Keys: "iw", Name: "inner-word"})

// user config, applied after defaults — last wins per (mode, keys); empty Name unmaps
km.Map(keyvind.Mapping{Mode: "normal", Keys: "<leader>g", Name: "goto-top"})

keymaps, conflicts, err := km.Build(keyvind.Key{Name: "space"})
// err: bad key spec or duplicate exact binding; conflicts: []keyvind.Conflict
m := keyvind.NewMatcher(keymaps, "normal")
// dispatch resolved commands however your app likes (a switch, a table, ...)
```

The optional `Desc` field is carried through to `Binding.Desc` (and so to a
resolved `Command.Target`/`Command.Operator`). The engine never reads it — it is
there for the host to build a help screen, a which-key-style popup, or logging.
`Keymap.Bindings()` returns every binding (sorted by key sequence) for exactly
this — or read `c.Target.Desc` at dispatch time.

To hot-reload bindings (e.g. the user edited their config), rebuild and hand the
new keymaps to the existing matcher instead of constructing a new one — the
current mode is preserved and any in-flight sequence is discarded:

```go
keymaps, _, _ := km.Build(leader) // resolves overrides + unmap
m.SetKeymaps(keymaps)
```

A vim-flavored preset — the command vocabulary and a default keymap (motions,
text objects, operators, search, marks, visual mode, …) — is intentionally **not**
shipped here: that set encodes a specific editor's model, so it belongs in the
host. Build it on a `Keymapper` (or the low-level `Keymap`/`Binding`/`Add` API),
which is all the engine needs.

## Framework integration

Each adapter is a separate module, so importing the core never pulls in a UI
framework. An adapter does two things: convert the framework's key event into
`keyvind.Key`, and drive the ambiguity timeout using that framework's mechanism.

### bubbletea (`teakit` for v1, `teakitv2` for v2)

Converts the key message to `keyvind.Key` and arms the timeout with `tea.Tick`.
The `Driver` API is identical for both versions:

```go
func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds, tick := a.keys.Update(msg) // a.keys is *teakit.Driver / *teakitv2.Driver
	for _, c := range cmds {
		a = a.dispatch(c) // interpret the keyvind.Command
	}
	return a, tick // the timeout tick — return it from Update
}
```

Use `teakitv2` with bubbletea v2 (`charm.land/bubbletea/v2`) to get precise
modifiers — see _Modifiers_ below.

### tcell / tview (`tcellkit`)

Converts `*tcell.EventKey` to `keyvind.Key`. Because tcell is poll-based, the
`Driver` arms the timeout itself and posts a `TimeoutEvent` back via the
function you supply (`screen.PostEvent` for raw tcell):

```go
driver := tcellkit.New(matcher, screen.PostEvent)
for {
	switch ev := screen.PollEvent().(type) {
	case *tcell.EventResize:
		screen.Sync()
	default:
		for _, c := range driver.Update(ev) { // also handles TimeoutEvent
			dispatch(c)
		}
	}
}
```

The same `FromEventKey` works inside **tview**'s `SetInputCapture` (it receives
`*tcell.EventKey`); see [`examples/tview`](examples/tview).

### Modifiers (Ctrl / Alt / Shift)

`keyvind.Key` carries `Ctrl`, `Alt`, and `Shift`, and `ParseKeys` accepts specs
like `<C-S-x>`, so the core can define and match modifier combos. Whether the
combo actually reaches you depends on the terminal:

- **Shift on special keys** (e.g. `<S-Tab>`) works everywhere.
- **Ctrl+Shift+letter** can only be distinguished by terminals that implement an
  enhanced keyboard protocol (Kitty, and the Windows console). `teakitv2` and
  `tcellkit` pass the `Shift` modifier through when reported, so `<C-S-x>` matches
  there. On legacy terminals (and bubbletea v1 / `teakit`) `Ctrl+Shift+x` is
  indistinguishable from `Ctrl+x`, so prefer `<leader>` sequences or key chains
  (`<C-x><C-y>`) if you need portability.

## Examples

- [`examples/bubbletea`](examples/bubbletea) — interactive TUI via `teakit`
- [`examples/tview`](examples/tview) — interactive TUI via `tcellkit` + tview

```sh
cd examples/bubbletea && go run .
cd examples/tview     && go run .
```

## Repository layout

```
keyvind/                core module — github.com/flexphere/keyvind (zero deps)
  adapters/teakit/      bubbletea v1 adapter — nested module
  adapters/teakitv2/    bubbletea v2 adapter (Go 1.25+) — nested module
  adapters/tcellkit/    tcell adapter (also serves tview/cview) — nested module
  examples/bubbletea/   bubbletea demo — nested module
  examples/tview/       tview demo — nested module
```

## Development

```sh
make all    # fmt-check + vet + lint + test (local gate)
make ci     # + tidy-check + race + examples build
make test   # tests only
make cover  # coverage per module
```

Enable the pre-commit hook (runs `make all`):

```sh
git config core.hooksPath .githooks
```

## Status

Early development. The grammar engine, conflict detection, and the bubbletea
v1/v2 (`teakit`, `teakitv2`) and tcell (`tcellkit`) adapters are implemented and
tested. Vim command vocabularies and default keymaps are intentionally left to
the host application, as are config _file formats_ (parse your config into
`Mapping`/`OperatorMapping` values).
```
