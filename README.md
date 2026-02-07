# safegroup

[![CI](https://github.com/yacchi/go-safegroup/actions/workflows/ci.yml/badge.svg)](https://github.com/yacchi/go-safegroup/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/yacchi/go-safegroup/graph/badge.svg?token=ARU7BCiEar)](https://codecov.io/gh/yacchi/go-safegroup)
[![Go Reference](https://pkg.go.dev/badge/github.com/yacchi/go-safegroup.svg)](https://pkg.go.dev/github.com/yacchi/go-safegroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/yacchi/go-safegroup)](https://goreportcard.com/report/github.com/yacchi/go-safegroup)
[![License](https://img.shields.io/github/license/yacchi/go-safegroup)](LICENSE)

`safegroup` is a panic-safe, join-first goroutine group for Go 1.20+.

- Recovers panics in worker goroutines.
- Converts each panic into a typed `*PanicError`.
- Collects all failures and returns them from `Wait()` via `errors.Join`.

This package is intentionally not `errgroup`-compatible in return semantics: `Wait()` returns all collected failures,
not only the first one.

## Installation

```bash
go get github.com/yacchi/go-safegroup
```

## Requirements

- Go 1.20+
- `mise` (for multi-version test tasks)

## Quick Start

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

## API Overview

- Constructor: `WithContext`
- Preset: `NewGroupPreset`, `GroupPreset.Group`
- Task APIs: `Go`, `GoLabel`, `TryGo`, `TryGoLabel`, `SetLimit`
- Detached task APIs: `GroupPreset.Go`, `GroupPreset.GoContext`, `GroupPreset.GoLabel`, `GroupPreset.GoLabelContext`
- Package helper APIs: `DefaultPreset`, `safegroup.DefaultGroup`, `safegroup.Go`, `safegroup.GoContext`, `safegroup.GoLabel`, `safegroup.GoLabelContext`
- Result APIs: `Wait`, `Errors`, `Panics`
- Panic helpers: `IsPanic`, `AsPanic`, `AllPanics`

Canonical API docs are published on `pkg.go.dev`:

- `https://pkg.go.dev/github.com/yacchi/go-safegroup`

## Default Behavior

- `CancelOnError(true)`
- `CancelOnPanic(true)`
- `CaptureStack(true)`

Use options in `WithContext(...)` to change behavior.

Hook options:

- `OnError`, `OnPanic`
- `OnErrorWithContext`, `OnPanicWithContext`

## GroupPreset

Use `GroupPreset` when you want to reuse the same options across calls.

```go
preset := safegroup.NewGroupPreset(
	safegroup.OnError(func(err error) { log.Printf("task error: %v", err) }),
	safegroup.OnPanic(func(pe *safegroup.PanicError) { log.Printf("panic: %+v", pe) }),
)

preset.Go(ctx, func() error {
	// detached async work
	return nil
})

preset.GoContext(ctx, func(ctx context.Context) error {
	_ = ctx
	return nil
})

group, groupCtx := preset.Group(ctx)
group.Go(func(ctx context.Context) error {
	_ = groupCtx
	return nil
})
_ = group.Wait()
```

For the most minimal detached use, call package-level helpers:

```go
type requestIDKey struct{}

requestID := requestIDKey{}

safegroup.DefaultPreset.
	WithOptions(
		safegroup.OnErrorWithContext(func(ctx context.Context, err error) {
			id, _ := ctx.Value(requestID).(string)
			log.Printf("request_id=%s task error: %v", id, err)
		}),
		safegroup.OnPanic(func(pe *safegroup.PanicError) { log.Printf("panic: %+v", pe) }),
	)

reqCtx := context.WithValue(context.Background(), requestID, "req-1")
safegroup.Go(reqCtx, func() error {
	return errors.New("failed")
})

safegroup.GoLabel(reqCtx, "worker-a", func() error {
	return errors.New("failed again")
})

safegroup.GoLabelContext(reqCtx, "worker-b", func(context.Context) error {
	return errors.New("failed with context")
})
// OnErrorWithContext can read request-id from this context and include it in logs.
```

## PanicError

`PanicError` stores:

- `Label`: task label from `GoLabel`/`TryGoLabel`
- `Value`: recovered panic value (`any`)
- `Stack`: captured stack trace when `CaptureStack(true)`

`PanicError` implements:

- `error`
- `fmt.Formatter` (`%+v` includes stack trace)
- `slog.LogValuer` on Go 1.21+ (`stack` is emitted as a structured field)
- `Unwrap() error` only when `Value` is an `error`

## Notes

- `runtime.Goexit` is not supported because it is not recoverable with `recover`.
- This package does not log by itself. Use `OnError` / `OnPanic` hooks for metrics or logging.
- When `CancelOnError(true)` / `CancelOnPanic(true)` is enabled, hooks are called before cancellation triggered by that same failure.
- Slow hooks can delay cancellation propagation for that same failure.
- `GoLabel` with `SetLimit` waits for a slot or group context cancellation. If canceled while waiting, the task is not started.
- Detached helpers (`Go*`/`GoLabel*` on presets and package-level APIs) start one worker task goroutine and one internal waiter goroutine (`group.Wait()`); each detached call therefore adds two goroutines until the task finishes.
- The internal waiter is used to guarantee final context cancellation on task completion and to release context-related resources even when the caller does not call `Wait`.
- `Wait` can be called multiple times and returns a consistent failure snapshot.
- `Wait` is terminal for the group: after `Wait` returns, `Go`/`GoLabel` panic and `TryGo`/`TryGoLabel` return `false`.
- Panics inside `OnError` / `OnPanic` / `OnErrorWithContext` / `OnPanicWithContext` hooks are not recovered by `safegroup`.

## Task Runner

`Makefile` is the primary task runner.

```bash
make test
make test-matrix
```

`make test-matrix` runs tests across Go `1.20` to `1.24` using `mise`.
If `mise` shows an untrusted config warning, run `mise trust` once in this repository.

You can also run a specific version directly with `mise`:

```bash
mise x go@1.20 -- go test ./...
```

## License

Apache License 2.0
