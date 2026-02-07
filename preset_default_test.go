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

	expectHookValues(t, hookValues, "plain", "with-context")
}

func TestPackageDefaultGroupDelegatesToDefaultPreset(t *testing.T) {
	old := DefaultPreset
	t.Cleanup(func() {
		DefaultPreset = old
	})

	group, _ := DefaultGroup(context.Background())
	group.Go(func(context.Context) error {
		return errors.New("task failed")
	})

	if err := group.Wait(); err == nil {
		t.Fatal("Wait() = nil, want non-nil error")
	}

	DefaultPreset = NewGroupPreset().WithOptions(CancelOnError(false))
	groupNoCancel, _ := DefaultGroup(context.Background())
	groupNoCancel.Go(func(context.Context) error {
		return errors.New("task failed")
	})
	groupNoCancel.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return nil
		}
	})

	err := groupNoCancel.Wait()
	if err == nil {
		t.Fatal("Wait() = nil, want non-nil error")
	}
	if errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected context cancellation in joined error: %v", err)
	}
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

	expectHookValues(t, hookValues, "plain-label", "context-label")
}

func expectHookValues(t *testing.T, ch <-chan string, wants ...string) {
	t.Helper()

	remaining := make(map[string]int, len(wants))
	for _, want := range wants {
		remaining[want]++
	}

	for range wants {
		select {
		case got := <-ch:
			if remaining[got] == 0 {
				t.Fatalf("unexpected context value: got %q", got)
			}
			remaining[got]--
		case <-time.After(1 * time.Second):
			t.Fatal("hook was not called")
		}
	}

	for want, count := range remaining {
		if count != 0 {
			t.Fatalf("missing context value: %q", want)
		}
	}
}
