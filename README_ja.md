# safegroup

[![CI](https://github.com/yacchi/go-safegroup/actions/workflows/ci.yml/badge.svg)](https://github.com/yacchi/go-safegroup/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/yacchi/go-safegroup/graph/badge.svg?token=ARU7BCiEar)](https://codecov.io/gh/yacchi/go-safegroup)
[![Go Reference](https://pkg.go.dev/badge/github.com/yacchi/go-safegroup.svg)](https://pkg.go.dev/github.com/yacchi/go-safegroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/yacchi/go-safegroup)](https://goreportcard.com/report/github.com/yacchi/go-safegroup)
[![License](https://img.shields.io/github/license/yacchi/go-safegroup)](LICENSE)

`safegroup` は Go 1.20+ 向けのパニック安全・Join-first goroutine グループです。

- ワーカー goroutine のパニックを自動的にリカバリ
- 各パニックを型付き `*PanicError` に変換
- すべての失敗を収集し、`Wait()` から `errors.Join` で返却

このパッケージは `errgroup` と互換性のない返却セマンティクスを意図的に採用しています: `Wait()` は最初のエラーだけでなく、
収集したすべての失敗を返します。

## インストール

```bash
go get github.com/yacchi/go-safegroup
```

## 動作要件

- Go 1.20+
- `mise`（マルチバージョンテスト用）

## クイックスタート

```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/yacchi/go-safegroup"
)

func main() {
	group, _ := safegroup.WithContext(context.Background())

	group.GoLabel("tenant=A/job=1", func(context.Context) error {
		return errors.New("regular error")
	})
	group.GoLabel("tenant=A/job=2", func(context.Context) error {
		panic("unexpected")
	})

	if err := group.Wait(); err != nil {
		fmt.Printf("joined error: %v\n", err)

		for _, panicErr := range safegroup.AllPanics(err) {
			fmt.Printf("panic detail:\n%+v\n", panicErr)
		}
	}
}
```

## API 概要

- コンストラクタ: `WithContext`
- タスク API: `Go`, `GoLabel`, `TryGo`, `TryGoLabel`, `SetLimit`
- 結果取得 API: `Wait`, `Errors`, `Panics`
- パニックヘルパー: `IsPanic`, `AsPanic`, `AllPanics`

正式な API ドキュメントは `pkg.go.dev` で公開されています:

- `https://pkg.go.dev/github.com/yacchi/go-safegroup`

## デフォルト動作

- `CancelOnError(true)`
- `CancelOnPanic(true)`
- `CaptureStack(true)`

`WithContext(...)` のオプションで動作を変更できます。

## PanicError

`PanicError` の保持する情報:

- `Label`: `GoLabel`/`TryGoLabel` で設定したタスクラベル
- `Value`: リカバリされたパニック値 (`any`)
- `Stack`: `CaptureStack(true)` 時に取得されるスタックトレース

`PanicError` が実装するインターフェース:

- `error`
- `fmt.Formatter`（`%+v` でスタックトレース付き出力）
- `Unwrap() error`（`Value` が `error` 型の場合のみ）

## 注意事項

- `runtime.Goexit` は `recover` でリカバリできないため、サポート対象外です。
- このパッケージ自体はログを出力しません。メトリクスやログには `OnError` / `OnPanic` フックを使用してください。
- `SetLimit` 使用時の `GoLabel` は、スロットが空くかグループコンテキストがキャンセルされるまで待機します。待機中にキャンセルされた場合、タスクは起動されません。
- `Wait` は複数回呼び出せます。各呼び出しは一貫した失敗スナップショットを返します。
- `Wait` はグループの終端操作です。`Wait` の返却後は `Go`/`GoLabel` は panic し、`TryGo`/`TryGoLabel` は `false` を返します。
- `OnError` / `OnPanic` フック内で発生した panic は `safegroup` では recover されません。

## タスクランナー

`Makefile` がタスクランナーです。

```bash
make test
make test-matrix
```

`make test-matrix` は `mise` を使用して Go `1.20` から `1.24` でテストを実行します。
`mise` が信頼設定の警告を表示した場合は、このリポジトリで `mise trust` を一度実行してください。

特定バージョンを直接指定して実行することもできます:

```bash
mise x go@1.20 -- go test ./...
```

## ライセンス

Apache License 2.0
