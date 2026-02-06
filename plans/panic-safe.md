# Panic-Safe Group OSS ライブラリ設計（改訂案）

## 概要

goroutine 内の **panic を recover して error 化**し、**すべての error / panic を `errors.Join` で統合**して返す “並列実行グループ” ライブラリ。
`golang.org/x/sync/errgroup` の「最初の 1 件だけ返す」契約から脱却し、**Go1.20+ の error 取り回し（Join/Is/As/Unwrap）を前提**にした現代的 API を提供する。

---

## 背景

* `golang.org/x/sync/errgroup` は「最初の error を返す」設計で、複数失敗の集約や詳細解析（panic 含む）に不向き
* 過去の `errgroup` の panic 伝播実装は、デバッグ/スタックの扱い等の事情で継続されなかった経緯がある
* 既存の “panic 対応 errgroup 互換” OSS は、互換性のために **単一エラー返却**に縛られ、`errors.Join` を前提にした運用と噛み合いにくい

> 本ライブラリは「互換」ではなく **“Join-first の並列実行ライブラリ”** として設計する。

---

## 価値提案（差別化ポイント）

### 1) Join-first（複数失敗の第一級サポート）

* `Wait()` は **常に `errors.Join`** を返す（0件なら `nil`）
* `errors.Is/As` で目的の失敗を抽出できる

### 2) Panic を構造化して解析可能に

* `PanicError` として **panic 値 + スタック**を保持
* `errors.As(err, *PanicError)` が使える
* `fmt.Formatter` を実装し、`%+v` で詳細（スタック）を出せる

### 3) 開発・運用で “どのタスクが死んだか” が分かる

* タスクに `label`（tenant/job/task名）を付与可能
* `PanicError` / 通常 error にも label を保持して可観測性を上げる

### 4) errgroup の便利機能は取り込む

* `SetLimit(n)` / `TryGo(...)` を提供（並列数制御）
* ただし戻り値は Join-first のまま

---

## 比較表（更新）

| 機能              | 標準 errgroup | “互換 panic errgroup”系 | **本ライブラリ**         |
| --------------- | ----------- | -------------------- | ------------------ |
| panic 回復        | ✗           | ○（実装差あり）             | **○（全 panic を捕捉）** |
| Wait の返却        | 最初の1件       | 最初の1件                | **全件 Join**        |
| errors.As/Is 前提 | △           | △                    | **◎（Join-first）**  |
| スタック            | N/A         | まちまち                 | **構造化 + `%+v`**    |
| ラベル（どのタスクか）     | ✗           | ✗/△                  | **○**              |
| SetLimit/TryGo  | ○           | ✗/△                  | **○**              |

---

## パッケージ構成

```
github.com/yacchi/safegroup/
├── group.go            # Group 実装（Go/TryGo/SetLimit/Wait）
├── panic_error.go      # PanicError（値・スタック・ラベル）
├── stack.go            # StackTrace（capture/format）
├── options.go          # Option と既定値
├── helpers.go          # AsPanic/IsPanic 等
├── group_test.go
├── example_test.go
├── go.mod              # go 1.20+
├── LICENSE             # MIT
└── README.md
```

---

## API 設計（改訂）

### 重要な設計変更点

* `Go(func() error)` ではなく **`Go(func(context.Context) error)`** を推奨
  → cancel/timeout を自然に伝播できる（errgroup の良さを残す）
* **label 付き API** をコアにする（マルチテナント用途で効く）

### PanicError

```go
type PanicError struct {
	Label string      // 任意: task識別子（tenant/job名など）
	Value any         // panic に渡された値
	Stack StackTrace  // スタック
	// Go routine ID は "非推奨/オプション"（安定取得が難しい・不要なことが多い）
}

func (e *PanicError) Error() string
func (e *PanicError) Format(s fmt.State, verb rune) // %v / %+v
```

#### 重要：Unwrap 方針

`panic` は通常 `error` ではないので、`Unwrap() error` を付けるなら設計方針を明確にします。

* **推奨**：`Unwrap() error` は実装しない（panic 値は `Value any` で保持）
* ただし `Value` が `error` の場合にだけ `Unwrap() error` を返すオプションはアリ
  → `panic(err)` を `errors.Is` で追えるのは便利

改訂案：

```go
func (e *PanicError) Unwrap() error {
	if v, ok := e.Value.(error); ok {
		return v
	}
	return nil
}
```

### Group

```go
type Group struct { /* ... */ }

func WithContext(ctx context.Context, opts ...Option) (*Group, context.Context)

// label なし版も用意するが、label版が主
func (g *Group) Go(f func(context.Context) error)
func (g *Group) GoLabel(label string, f func(context.Context) error)

func (g *Group) TryGo(f func(context.Context) error) bool
func (g *Group) TryGoLabel(label string, f func(context.Context) error) bool

func (g *Group) SetLimit(n int)

// Wait は Join-first
func (g *Group) Wait() error

// 取り出し API（任意だが便利）
func (g *Group) Errors() []error       // 通常 error + PanicError 全部
func (g *Group) Panics() []*PanicError // panic のみ
```

### ヘルパー関数

```go
func IsPanic(err error) bool
func AsPanic(err error) *PanicError // Join 内から最初の PanicError を拾う
func AllPanics(err error) []*PanicError // Join 内の panic 全部
```

> `AsPanic` は 1 個だけ返す版と、`AllPanics` で全件返す版を両方用意すると直感的です。

---

## オプション設計（明確化）

### Option 一覧（提案）

* `CancelOnError(bool)`

  * **デフォルト: true**（errgroup に近い）
* `CancelOnPanic(bool)`

  * **デフォルト: true**
* `CaptureStack(bool)`

  * **デフォルト: true**（panic からの復旧用途なので）
* `OnPanic(func(*PanicError))` / `OnError(func(error))`

  * メトリクス/ログのフック（ただし “ライブラリがログしない” を徹底）

> ステートレス・テナント継続だと「panic を捕まえて job を失敗扱いにして次へ」が主なので、cancel は **job 内並列にだけ影響**する、という意味で `true` が便利です。

---

## runtime.Goexit の扱い（整理）

* **対応しない**（ドラフトの結論を維持）
* README に「`Goexit` は recover で捕捉不可であり、本ライブラリの目的外」と明記

---

## Go バージョン

* **Go 1.20+**

  * `errors.Join` を必須利用
  * `context.WithCancelCause` は “将来の拡張” として検討（panic を cause にする等）

    * ただしまずはシンプルに `WithCancel` で良い

---

## 挙動仕様（明文化）

### Wait の返却

* 通常 error / panic error を **発生順に収集**（順序保持）
* `Wait()` は `errors.Join(errs...)` を返す
* 0件なら `nil`

### panic の取り扱い

* すべて recover して `PanicError` に変換して収集
* `CancelOnPanic=true` なら context を cancel（他タスクの停止を促す）
* “panic を握りつぶす” のではなく「**error として呼び出し側に返す**」が目的

---

## 使用例（改訂：label と AllPanics）

```go
g, ctx := safegroup.WithContext(context.Background())

g.GoLabel("tenant=A/job=1", func(ctx context.Context) error {
	return errors.New("regular error")
})

g.GoLabel("tenant=A/job=2", func(ctx context.Context) error {
	panic("unexpected")
})

if err := g.Wait(); err != nil {
	// Join された複合エラー
	fmt.Printf("Error: %v\n", err)

	// panic をすべて取得
	for _, pe := range safegroup.AllPanics(err) {
		fmt.Printf("Panic: %+v\n", pe) // %+v でスタック
	}
}
_ = ctx
```

---

## 実装ステップ（更新）

### Phase 0: API 固定

* [ ] `GoLabel` / `TryGoLabel` の追加
* [ ] `PanicError.Unwrap` 方針（Value が error のときだけ unwrap）を決定

### Phase 1: リポジトリ準備

* [ ] `go mod init github.com/yacchi/safegroup`（go 1.20）

### Phase 2: コア実装

* [ ] `stack.go` - `type StackTrace []uintptr` 方式推奨（`debug.Stack()`より構造化しやすい）
* [ ] `panic_error.go` - `PanicError`（Format, Unwrap）
* [ ] `group.go` - 内部は errgroup の semaphore と同等の仕組み（SetLimit/TryGo）
* [ ] `helpers.go` - `IsPanic/AsPanic/AllPanics`

### Phase 3: テスト（重要ケース）

* [ ] 複数 panic が全部 Join に入る
* [ ] `errors.As(joined, *PanicError)` が成立する
* [ ] `AllPanics` で全件取れる
* [ ] SetLimit / TryGo の挙動（制限超過で false）
* [ ] CancelOnPanic/CancelOnError の有無で ctx.Done が変わる

### Phase 4: ドキュメント

* [ ] “互換ではない” の宣言（Join-first）
* [ ] 典型パターン（マルチテナントの job 内並列集約）
* [ ] スタック出力方法（`%+v`）

### Phase 5: リリース

* [ ] v0.1.0

---

## 追加提案（公開時に強いポイント）

### 1) 互換 shim を別パッケージに切る（必要なら）

互換を入れたくなったら、コアを汚さず

* `safegroup/compat/errgroup` みたいに薄いアダプタ
* `Wait()` は最初の1件だけ返す（互換用途限定）

を後から足せます。コアは Join-first を守るのが価値です。

### 2) “エラーのサイズ” と “スタックの扱い” を README で言及

* stack が巨大化するケースがある（大量 panic）
* その場合は `CaptureStack(false)` を選べる
* もしくは `StackTrace` を “圧縮（上位Nフレーム）” オプション

---

## 決定事項（改訂）

* **パッケージ名**: `safegroup`
* **リポジトリ**: `github.com/yacchi/safegroup`
* **最小 Go**: 1.20
* **runtime.Goexit**: 対応しない
* **非互換方針**: Join-first（errgroup 互換を目的にしない）
