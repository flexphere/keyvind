# keyvind examples

各例は独立した Go モジュール（`replace` でローカルのコア/アダプタを参照）。
どちらもテキスト編集はせず、いくつかの単純なコマンドを登録して「どのコマンドが
解決したか」を表示する最小デモ。`Keymapper` でキーマップを宣言的に構築する。

共通のコマンド:

- `p` `s` … 単一キーの簡単なコマンド（`3p` のようにカウント対応）
- `gg` … 複数キーを組み合わせたコマンド
- `g` … `gg` と曖昧なので、タイムアウト（待つ or Enter 相当）でのみ確定
- `<Space>w` `<Space>r` … leader コマンド（`<leader>` = Space）
- `<C-s>` `<S-Tab>` `<C-x><C-s>` … Ctrl / Shift / Ctrl チョードの修飾子コマンド
- `q` / `ctrl+c` … 終了

## bubbletea — teakit を使った実 TUI

`teakit.Driver` が `tea.KeyMsg → keyvind.Key` 変換と曖昧性 timeout（`g` vs `gg`）を
`tea.Tick` で担う。解決したコマンドを履歴表示する。

```sh
cd examples/bubbletea
go run .
```

## tview — tcellkit を使った実 TUI

`tcellkit.FromEventKey` を tview の `SetInputCapture`（`*tcell.EventKey` を受け取る）に
差し、timeout は `app.QueueUpdateDraw` 経由で `Driver` に戻す。

```sh
cd examples/tview
go run .
```
