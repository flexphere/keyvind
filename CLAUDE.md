# keyvind — Claude 作業ルール

任意の TUI アプリに vim ライクなキーシーケンス→機能バインディングを提供する Go ライブラリ。
設計の全体像は [DESIGN.md](DESIGN.md) を参照。

## Quality Rules（コミット前に必須）

1. **`make all` が green であること**（fmt-check + vet + lint + test）。落ちたまま commit しない。
2. **変更には必ずテストを伴う**。新しい挙動・分岐・バグ修正には対応するテストを追加する。
3. **公開 API の変更時は DESIGN.md / README.md / godoc コメントを更新**する。
4. pre-commit フックを有効化しておく: `git config core.hooksPath .githooks`

## アーキテクチャ不変条件（壊さない）

- **コア（ルートパッケージ `keyvind`）は標準ライブラリのみに依存**する。bubbletea や端末ライブラリを
  import してはいけない。フレームワーク連携は `adapters/` 配下の別モジュール（`adapters/teakit` 等）に
  隔離する。keyvind は engine（core ＋ Keymapper ＋ adapters）に徹し、vim の語彙・既定キーマップ・
  dispatch 糖衣は同梱しない（利用側の責務）。
- **ライブラリはパーサであってエディタではない**。キー列を `Command` に解決するまでが責務で、
  範囲計算や実際の編集は利用側に委ねる。
- マッチャの状態機械（`matcher.go`）は純粋関数として保つ。I/O・タイマ・乱数を持ち込まない。
  タイムアウトは `Result.ArmTimeout` を返してホストに委譲する。

### コアが「知ってはいけない」もの（negative space）

ドリフト（エディタ概念がコアに漏れる）を避けるため、コアが**扱わない**ことを明示する:

- **テキスト/編集の内容**: テキスト本文・カーソル位置・選択範囲・範囲計算・undo/redo・
  クリップボード/ペースト・レジスタやマクロの中身・スクロール/viewport。
- **特定エディタモードの意味**: `visual`/`insert`/`cmdline` などの**意味**。モードは
  **任意の名前空間**（利用者が定義、各モード独立 trie、挙動は利用側 dispatch）であって、
  コアは特定モード名や「selection モード」のような意味を知らない。バインドの文法ロール
  （standalone か operand か）は**どのモードのキーマップに属するか**で決まる。コアに Kind タグは無く、
  唯一の特例は二重オペレータ（dd → linewise）のみ。vim の語彙・ロール名・既定キーマップは
  consumer 側のプリセットに置く（このリポジトリには持たない）。
- **行入力**: `:` `/` のような改行終端の文字列入力。これはホストの入力モード（テキスト部品）の責務。
  コアが扱うのは「キーシーケンス → 構造化 `Command`」と、単一キー引数（`AwaitsArg`）まで。

これらの不変条件は機械的にも検査する（[invariants_test.go](invariants_test.go) ＝ `make all` で実行）:
コア識別子/文字列リテラルへのエディタ語彙混入と、非 stdlib import を検知する。トリップワイヤであり
証明ではないので、**新しい概念の追加時はこの negative space に照らしてレビュー**する（禁止語リストも更新）。

## コーディング規約

- 純粋関数は意図が伝わる接頭辞で切り出す: `parse_*` 相当（Go では `parse*`/`resolve*`/`build*`）。
- godoc は公開シンボルに必ず付ける（`revive` の exported ルールで強制）。
- フォーマットは `make fmt`（gofmt + goimports, local prefix `github.com/flexphere/keyvind`）。

## よく使うコマンド

- `make all` — ローカル品質ゲート
- `make test` / `make test-race` — テスト
- `make lint` — golangci-lint (v2)
- `make cover` — カバレッジ合計
