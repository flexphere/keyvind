# Changelog

All notable changes to keyvind are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and the project follows
[Semantic Versioning](https://semver.org). All modules (core + adapters) share
one version and are released together — see [RELEASING.md](RELEASING.md).

## [Unreleased]

## [v0.0.1] - 2026-06-14

Initial release.

- **Core engine** (`github.com/flexphere/keyvind`, dependency-free): vim-style
  key parsing (`<leader>`, `<C-x>`, `<S-Tab>`, …), trie-based per-mode keymaps,
  and the operand-mode-switch matcher — counts, operators reading an operand
  from a switched mode, doubled operators (single- and multi-key, e.g. `dd` /
  `gUgU` → linewise), and `AwaitsArg` character-argument commands (`f`/`r`/…).
- Declarative `Keymapper` (`Map` / `Operator` / `Build`) with last-wins
  overrides, unmap, and static prefix-conflict detection (`Conflicts`).
- `Keymap.Bindings()` and an optional `Mapping.Desc` for help screens.
- `Matcher.SetKeymaps` for hot-reloading bindings.
- **Adapters** (nested modules): `adapters/teakit` (bubbletea v1),
  `adapters/teakitv2` (bubbletea v2), `adapters/tcellkit` (tcell / tview),
  each converting the framework's key events to `keyvind.Key` and driving the
  ambiguity timeout.

[Unreleased]: https://github.com/flexphere/keyvind/compare/v0.0.1...HEAD
[v0.0.1]: https://github.com/flexphere/keyvind/releases/tag/v0.0.1
