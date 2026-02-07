package safegroup

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPackageGoAndGoContextDelegateToDefaultPreset(t *testing.T) {
	old := DefaultPreset
	t.Cleanup(func() {
		DefaultPreset = old
	})

	type contextKey string
	const key contextKey = "request-id"

	hookValues := make(chan string, 2)
	DefaultPreset = NewGroupPreset().
		WithOptions(
			CancelOnError(false),
			OnErrorWithContext(func(ctx context.Context, _ error) {
				value, _ := ctx.Value(key).(string)
				hookValues <- value
			}),
		)

	plainCtx := context.WithValue(context.Background(), key, "plain")
	Go(plainCtx, func() error {
		return errors.New("task failed")
	})

	contextCtx := context.WithValue(context.Background(), key, "with-context")
	GoContext(contextCtx, func(context.Context) error {
		return errors.New("task failed")
	})

	expectHookValue(t, hookValues, "plain")
	expectHookValue(t, hookValues, "with-context")
}

func TestPackageGoLabelAndGoLabelContextDelegateToDefaultPreset(t *testing.T) {
	old := DefaultPreset
	t.Cleanup(func() {
		DefaultPreset = old
	})

	type contextKey string
	const key contextKey = "request-id"

	hookValues := make(chan string, 2)
	DefaultPreset = NewGroupPreset().
		WithOptions(
			CaptureStack(false),
			OnPanicWithContext(func(ctx context.Context, _ *PanicError) {
				value, _ := ctx.Value(key).(string)
				hookValues <- value
			}),
		)

	plainCtx := context.WithValue(context.Background(), key, "plain-label")
	GoLabel(plainCtx, "worker-a", func() error {
		panic("boom")
	})

	contextCtx := context.WithValue(context.Background(), key, "context-label")
	GoLabelContext(contextCtx, "worker-b", func(context.Context) error {
		panic("boom")
	})

	expectHookValue(t, hookValues, "plain-label")
	expectHookValue(t, hookValues, "context-label")
}

func expectHookValue(t *testing.T, ch <-chan string, want string) {
	t.Helper()

	select {
	case got := <-ch:
		if got != want {
			t.Fatalf("unexpected context value: got %q, want %q", got, want)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("hook was not called")
	}
}
