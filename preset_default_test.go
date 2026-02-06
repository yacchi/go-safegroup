package safegroup

import (
	"errors"
	"testing"
	"time"
)

func TestPackageGoUsesDefaultPreset(t *testing.T) {
	old := DefaultPreset
	t.Cleanup(func() {
		DefaultPreset = old
	})

	errorHook := make(chan error, 1)
	DefaultPreset = NewGroupPreset().
		WithOptions(
			CancelOnError(false),
			OnError(func(err error) { errorHook <- err }),
		)

	taskErr := errors.New("task failed")
	Go(func() error {
		return taskErr
	})

	select {
	case err := <-errorHook:
		if !errors.Is(err, taskErr) {
			t.Fatalf("unexpected hook error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("error hook was not called")
	}
}

func TestGroupPresetWithOptionsConfigureBehavior(t *testing.T) {
	panicHook := make(chan *PanicError, 1)
	preset := NewGroupPreset().
		WithOptions(
			CaptureStack(false),
			OnPanic(func(panicErr *PanicError) { panicHook <- panicErr }),
		)

	preset.GoLabel("worker-a", func() error {
		panic("boom")
	})

	select {
	case panicErr := <-panicHook:
		if panicErr.Label != "worker-a" {
			t.Fatalf("unexpected label: %q", panicErr.Label)
		}
		if len(panicErr.Stack) != 0 {
			t.Fatalf("expected no stack, got %d frames", len(panicErr.Stack))
		}
	case <-time.After(1 * time.Second):
		t.Fatal("panic hook was not called")
	}
}
