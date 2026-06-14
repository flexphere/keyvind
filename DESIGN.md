# keyvind — 設計ドキュメント

任意の TUI アプリケーションに **vim ライクなキーシーケンス → 機能バインディング** を提供する Go ライブラリ。

- module: `github.com/flexphere/keyvind`
- package: `keyvind`（コードは `keyvind.NewMatcher(...)`）
- 位置づけ: 「**多くの vim ユーザーに直感的な汎用キーシーケンス文法エンジン**」。
  特定エディタの意味論ではなく、キー列 → 構造化コマンドの一般的な文法のみを扱う。

> 用語: 本書で「**ホスト**」＝ keyvind を組み込む利用側アプリケーション。ハンドラ実装・モード定義・
> 解決済みコマンドの dispatch・タイムアウトのタイマ駆動を担う。エンドユーザ（アプリの利用者）と
> 区別する場合のみ「ユーザ」と書く。

## 設計原則

### 1. ライブラリは「文法エンジン」であって「エディタ」ではない

中核の責務は、入力されたキー列を **構造化された `Command`** に解決すること。
たとえば `3dw` という入力を `{Count: 3, Operator: "delete", Target: "word"}` に変換する。

「カーソルから単語末尾までの範囲を計算して削除する」といった**ドメイン処理はホストが実装**する。
これによりライブラリはアプリケーションのデータモデルから完全に独立し、
エンジン部分を純粋関数としてテストできる。

```
[]Key ──▶ Matcher (state machine) ──▶ Command ──▶ ホストの handler
```

### 2. キー列（設定）とアクション（コード）の分離

vim の `nnoremap dd delete_line` と同じ思想。

- **コード**: 名前付きアクションを宣言する（`delete`, `word`, `goto-top` …）
- **設定**: キー列 → 名前 のマッピングを宣言する（後勝ちで上書き可能）

この分離により、キーバインドの差し替え・上書き・**競合検知**が設定レイヤで完結する。

### 3. モードは任意定義。文法ロールは「どのモードに居るか」で決まる

`normal` / `insert` / `visual` を固定で持たない。
モードは「独自のキーマップ（独立した Trie）を持つ名前空間」にすぎず、ホストが任意に定義できる。

**バインドの文法ロールはタグではなく『どのキーマップに属するか』で表現する**:

- **終端バインド**（`OperandMode == ""`）: 単体でコマンドに解決する。
- **オペレータバインド**（`OperandMode != ""`）: 解決時に matcher の照合先を
  `OperandMode` のキーマップへ切り替え、そこでオペランドを1つ読み、`Operator` と
  `Target` を持つ単一の `Command` に合成する。

したがって「モーション」と「テキストオブジェクト」は**タグではなく配置**で決まる:

- アクティブモードと operand モードの**両方**にあるキー = 単体でも operand でも使える（vim の nmap+omap、例 `w`）
- operand モード**のみ**にあるキー = operand 専用（テキストオブジェクト、例 `iw`）
- アクティブモード**のみ**にあるキー = 単なるコマンド（例 `x`）

エンジンが特別扱いする例外は **二重オペレータ（`dd`/`cc`/`yy`、多キーの `gUgU` 等 → linewise）** のみ。
vim のデフォルトキーマップ（モーション/テキストオブジェクト/オペレータの語彙とロール名）は
**特定エディタの設計判断**なので、このリポジトリには持たず**ホストのプリセット**に置く。
エンジン自体は特定キーも特定モード名もハードコードしない。

### 4. フレームワーク非依存

コアは bubbletea にも特定の端末ライブラリにも依存しない（標準ライブラリのみ）。
フロントエンドとの接点は2つだけで、どちらも具体的なライブラリ型を避ける:

- **入力**: コアは自前の `Key` 型のみを扱う。各フロントエンドが「自分のキーイベント → `keyvind.Key`」変換を1つ用意する。
- **タイムアウト**: コアはタイマを持たず、曖昧時に `Result.ArmTimeout` を返すだけ。
  ホストが任意の方法でタイマを張り、期限が来たら `Matcher.Timeout()` を呼ぶ。

bubbletea 連携は別 module `adapters/teakit` に隔離し、コア単体を import しても tea を引き込まない。

## アーキテクチャ

```
keyvind/                      ── コア module（github.com/flexphere/keyvind, 依存ゼロ）
  key.go          Key 型と vim 記法パーサ（<leader> <C-x> <CR> …）          ── 純粋
  keymap.go       Binding と Trie（モードごとのキー列ストレージ）            ── 純粋
  matcher.go      状態機械。Feed(Key) → Result。operator で operand モードへ切替 ── 純粋
  command.go      Binding / Command / Result 型                            ── 純粋
  conflict.go     プレフィックス競合の静的検出（Conflicts）                 ── 純粋
  keymapper.go    Mapping / OperatorMapping / Keymapper：宣言的ビルダー       ── 純粋
  invariants_test.go  エディタ語彙の混入・非 stdlib import を AST で機械検出
  （config ファイル形式は持たない＝ホストの責務。アプリが自前 config → Mapping に変換）
  （vim の語彙・既定キーマップも持たない＝特定エディタの設計判断なのでホストのプリセットに置く）
  adapters/       ── 各 UI フレームワーク用アダプタ（それぞれ別 module）
    teakit/         bubbletea v1（…/keyvind/adapters/teakit）
    teakitv2/       bubbletea v2（charm.land/bubbletea/v2、Go 1.25+）
    tcellkit/       tcell（tview/cview も対象）
```

**各アダプタは `adapters/` 配下の独立した nested module**。コア module は依存ゼロを永続的に維持する。
keyvind は engine（core ＋ Keymapper ＋ adapters）に徹し、vim の語彙・既定キーマップ・dispatch 糖衣は
持たない（ホストの責務）。

### 宣言的ビルダーと dispatch を分離する（原則②の体現）

「キー列＝設定／アクション＝コード」の分離を **API 構造**にする:

- **`keyvind.Keymapper`（core）**: `Map`（終端）/`Operator`（オペレータ）でモード横断にバインドを宣言し、
  `Build(leader)` で per-mode keymaps を返す。**dispatch 非依存・非ジェネリック**。解決済み `Command` の
  dispatch はホストが任意に行う（switch / テーブル等）。
- **vim プリセット（ホスト側）**: コマンド名定数・ロール名ヘルパ（nmap/omap/xmap の写像）・既定キーマップは
  特定エディタの設計判断なので **ホスト側**（例: bubbles）に置く。`keyvind.Keymapper` の上に薄く載せられ、
  `Build` は engine と同じ keymaps を返すので、keyvind 自体はこれを同梱しない。

いずれもコア型＋stdlib のみ＝コアの依存ゼロを保つ。

`Mapping`/`OperatorMapping` は任意の `Desc string`（説明文）を持てる。エンジンは一切解釈せず
`Binding.Desc` → 解決後の `Command.Target`/`Command.Operator` まで透過するだけ。ホストがヘルプ画面・
which-key 風ポップアップ・ログを組むための純粋なメタデータ（標準キー記法の値なので不変条件にも抵触しない）。
全バインドの列挙は `Keymap.Bindings()`（キー列ソート）で取得できる。

## キー列の文法（operand モード切替モデル）

vim の normal モードを一般化した小さな文法。文法ロールは**モード配置**で表現する:

```
command   := count? ( operator count? operand     // 3dw, d2w
                    | operator operator            // dd, yy（二重 = linewise）
                    | count? terminal              // 3j, w, x
                    )
operand   := （operator.OperandMode のキーマップ内の終端バインド）
```

- **terminal**: `OperandMode == ""` のバインド。単体で `Command{Target}` に解決。
- **operator**: `OperandMode != ""` のバインド。解決時に照合先を operand モードへ切替え、
  そこで operand を1つ読み `Command{Operator, Target}` に合成。
- **二重オペレータ**: operator 保留中に operator 自身のキー列が**全て再入力**されたら linewise
  （`Command{Operator, Target=operator, Linewise:true}`）。単一キー（`dd`）も多キー（`gUgU`）も対応。
  operand trie と二重化シーケンスを並行追跡し、共有プレフィックス（operand の `gg` と operator `gU`）でも
  2キー目で正しく分岐する（`gUgg`=upper+goto-top, `gUgU`=linewise）。
- **count**: operator 前後の count は乗算（`2d3w` → 6）。

N 個の operator × M 個の operand は、operand を一度だけ operand モードに置けば全 operator が共有するので
`N + M` バインドで済む（`N × M` の組合せ爆発は起きない）。

### モーションとテキストオブジェクトは別キーマップに住む

このモデルの帰結として、`i`（コマンド）と `iw`（テキストオブジェクト）のようなプレフィックス衝突が
**構造的に存在しない**:

- `i`（コマンド）は **normal モード**にあり、子を持たない → 即確定（INSERT にラグなし）。
- `iw`（テキストオブジェクト）は **operand モード**にしか無い → `di` の後にだけ照合される。

両者は別キーマップなので調停すべき曖昧性が無い。matcher は mode 非依存で、
"visual"/"selection"/テキストといったエディタ概念を一切持たない（モードは単なる名前空間）。

## 曖昧性とタイムアウト

`g`（単体）と `gg`（より長い列）のように、**同一キーマップ内で**あるキー列が「完全一致」かつ
「より長い列のプレフィックス」でもある場合、vim は `timeoutlen` だけ次のキーを待つ。

keyvind はこれを **トライのカーソル位置だけ**で表現する。`g` を打つと g ノード（終端かつ子あり）に
カーソルが留まり、`Result{Pending, ArmTimeout}` を返して何も確定しない。解消は3経路:

1. **延長** (`gg`): 次キーが子にヒット → 長い方を確定、短い方は破棄
2. **逸れ** (`gx`): 次キーが子に無い → 留まった `g` を確定し、同じキーを root から再処理（1 回の Feed で 2 コマンド）
3. **時間切れ**: ホストの timer が `Matcher.Timeout()` を呼ぶ → 留まった `g` を確定

ブロッキングできない環境（bubbletea 等）でも `ArmTimeout` シグナル＋`Timeout()` で扱える。

## 文字引数コマンド（AwaitsArg）

`f`/`F`/`t`/`T`（指定文字へ移動）、`r`（文字置換）など、vim には**確定後に次の生キー1つを
引数として取る**コマンドがある。第2キーはトライ上の固定バインドではなくその場で捕捉する任意文字。

`Binding.AwaitsArg` を立てると、そのバインドが解決された瞬間に確定せず matcher が `awaiting` 状態に
入り、**次の Feed キーをトライ照合せず `Command.Arg` として捕捉**して確定する。

- terminal/operator と**直交する一様なフラグ**（`f`=モーション＋引数、`r`=アクション＋引数）。
- count / operator と合成: `3fx`・`dfx`（`Command{Operator:delete, Target:find, Arg:'x', Count}`）。
- 引数待ち中は**タイムアウトを張らない**（vim は無限に待つ）。`<Esc>` 引数は**キャンセル**（emit せず reset）。
- `SetMode` や失敗時の reset で `awaiting` も破棄。
- AwaitsArg バインドの子は**到達不能**（次キーは引数に吸われる）→ 競合検知が `ConflictArgShadow` で報告。
- 宣言: `Mapping{..., AwaitsArg: true}`。

**スコープ外**: `:`(ex) や `/`(検索) は単一文字引数ではなく**行入力**なので keyvind の責務外
（ホストが insert ライクな入力モードで扱う）。keyvind が持つのは「単一キー引数まで」。

## 競合検知

**ファイル形式のパースは持たない**（ホストの責務）。core はメモリ上の Keymap を
**エンジンの解決規則を踏まえて**静的解析する。`Keymap.Conflicts() []Conflict` /
`Matcher.Conflicts() []Conflict`（全モード横断・決定的にソート）。

検出するのは「**終端かつ子を持つノード**」＝あるキー列が、より長い列の真のプレフィックスに
なっているケース。トライを1回 walk すれば、`g`(終端)＋`gg`(子) でも、`dw` を先に登録してから
`d` を足す（非終端ノードへのバインド）でも、同じ一条件で両方向とも捕まる。挿入順非依存。

| ConflictKind | 意味 | 例 |
|---|---|---|
| `ConflictAmbiguous` | プレフィックス一致。timeout 後には確定するが遅延が生じる | `g` ＋ `gg` |
| `ConflictOperatorShadow` | 上記のうち短い側が **operator**（`OperandMode != ""`）。operator の発火が timeout 待ちになる | operator `d` ＋ `dw` |
| `ConflictArgShadow` | 短い側が **AwaitsArg**。子は引数に吸われ到達不能 | `f`(AwaitsArg) ＋ `fx` |

- **完全重複**（同一キー列に2バインド）は `Keymap.Add` が即エラーで弾く（`Conflicts` の対象外）。
- 検出は `Add` を失敗させず `Conflicts()` で報告する方針。順序非依存で、将来の上書きとも両立する。
- `[]Conflict` を返すだけで、エラー扱いにするか警告に留めるかはホストが決める。

## 設計不変条件のテスト（invariants_test.go）

コアが「エディタ概念を知らない」ことを**機械的に**担保する。`invariants_test.go` がコアルートの
非テスト `.go` を AST 解析し:

- **`TestCoreHasNoEditorVocabulary`**: 識別子・構造体フィールド・文字列リテラルに
  `selection`/`visual`/`insert`/`cursor`/`undo`/`yank`/… が現れたら fail（`mode`/`motion`/`operand` は
  単なる名前空間なので許可）。エディタ概念がコアに漏れる退行を `make all` で検出する。
- **`TestCoreImportsOnlyStdlib`**: コアが非 stdlib を import したら fail。

vim の語彙（`yank`/`insert`/…）はホストのプリセットに置けば自由に使える——この不変条件テストは
コアルートのみを走査するため。
