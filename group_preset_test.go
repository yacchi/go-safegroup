package safegroup

import (
	"context"
	"errors"
	"testing"
	"time"
)

type contextKey string

func TestGroupPresetGoCallsOnError(t *testing.T) {
	errorHook := make(chan error, 1)
	preset := NewGroupPreset(
		CancelOnError(false),
		OnError(func(err error) {
			errorHook <- err
		}),
	)

	taskDone := make(chan struct{})
	taskErr := errors.New("task failed")

	preset.Go(context.Background(), func() error {
		close(taskDone)
		return taskErr
	})

	select {
	case <-taskDone:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not run")
	}

	select {
	case err := <-errorHook:
		if !errors.Is(err, taskErr) {
			t.Fatalf("unexpected hook error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("error hook was not called")
	}
}

func TestGroupPresetGoLabelPreservesPanicLabel(t *testing.T) {
	panicHook := make(chan *PanicError, 1)
	preset := NewGroupPreset(
		CaptureStack(false),
		OnPanic(func(panicErr *PanicError) {
			panicHook <- panicErr
		}),
	)

	preset.GoLabel(context.Background(), "worker-a", func() error {
		panic("boom")
	})

	select {
	case panicErr := <-panicHook:
		if panicErr.Label != "worker-a" {
			t.Fatalf("unexpected label: %q", panicErr.Label)
		}
		if panicErr.Value != "boom" {
			t.Fatalf("unexpected panic value: %v", panicErr.Value)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("panic hook was not called")
	}
}

func TestGroupPresetGroupAppliesConfiguredOptions(t *testing.T) {
	errorHook := make(chan error, 1)
	preset := NewGroupPreset(
		CancelOnError(false),
		OnError(func(err error) {
			errorHook <- err
		}),
	)
	group, _ := preset.Group(context.Background())
	taskErr := errors.New("task failed")

	group.Go(func(context.Context) error {
		return taskErr
	})

	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}

	select {
	case err := <-errorHook:
		if !errors.Is(err, taskErr) {
			t.Fatalf("unexpected hook error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("error hook was not called")
	}
}

func TestNilGroupPresetGroupUsesDefaultOptions(t *testing.T) {
	var preset *GroupPreset

	group, _ := preset.Group(context.Background())
	taskErr := errors.New("task failed")
	group.Go(func(context.Context) error {
		return taskErr
	})

	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}
}

func TestNilGroupPresetGoRunsTask(t *testing.T) {
	var preset *GroupPreset

	taskDone := make(chan struct{})
	preset.Go(context.Background(), func() error {
		close(taskDone)
		return nil
	})

	select {
	case <-taskDone:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not run")
	}
}

func TestNilGroupPresetGoLabelRunsTask(t *testing.T) {
	var preset *GroupPreset

	taskDone := make(chan struct{})
	preset.GoLabel(context.Background(), "worker-a", func() error {
		close(taskDone)
		return nil
	})

	select {
	case <-taskDone:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not run")
	}
}

func TestGroupPresetGoPassesContextToErrorHook(t *testing.T) {
	const key contextKey = "request-id"

	hookValue := make(chan string, 1)
	preset := NewGroupPreset(
		CancelOnError(false),
		OnErrorWithContext(func(ctx context.Context, _ error) {
			value, _ := ctx.Value(key).(string)
			hookValue <- value
		}),
	)

	ctx := context.WithValue(context.Background(), key, "req-1")
	preset.Go(ctx, func() error {
		return errors.New("task failed")
	})

	select {
	case value := <-hookValue:
		if value != "req-1" {
			t.Fatalf("unexpected context value: %q", value)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("error hook was not called")
	}
}

func TestGroupPresetGoLabelPassesContextToPanicHook(t *testing.T) {
	const key contextKey = "request-id"

	hookValue := make(chan string, 1)
	preset := NewGroupPreset(
		CaptureStack(false),
		OnPanicWithContext(func(ctx context.Context, _ *PanicError) {
			value, _ := ctx.Value(key).(string)
			hookValue <- value
		}),
	)

	ctx := context.WithValue(context.Background(), key, "req-2")
	preset.GoLabel(ctx, "worker-a", func() error {
		panic("boom")
	})

	select {
	case value := <-hookValue:
		if value != "req-2" {
			t.Fatalf("unexpected context value: %q", value)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("panic hook was not called")
	}
}

func TestGroupPresetGoContextPassesContextToErrorHook(t *testing.T) {
	const key contextKey = "request-id"

	hookValue := make(chan string, 1)
	preset := NewGroupPreset(
		CancelOnError(false),
		OnErrorWithContext(func(ctx context.Context, _ error) {
			value, _ := ctx.Value(key).(string)
			hookValue <- value
		}),
	)

	ctx := context.WithValue(context.Background(), key, "req-ctx-1")
	preset.GoContext(ctx, func(context.Context) error {
		return errors.New("task failed")
	})

	select {
	case value := <-hookValue:
		if value != "req-ctx-1" {
			t.Fatalf("unexpected context value: %q", value)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("error hook was not called")
	}
}

func TestGroupPresetGoLabelContextPassesContextToPanicHook(t *testing.T) {
	const key contextKey = "request-id"

	hookValue := make(chan string, 1)
	preset := NewGroupPreset(
		CaptureStack(false),
		OnPanicWithContext(func(ctx context.Context, _ *PanicError) {
			value, _ := ctx.Value(key).(string)
			hookValue <- value
		}),
	)

	ctx := context.WithValue(context.Background(), key, "req-ctx-2")
	preset.GoLabelContext(ctx, "worker-a", func(context.Context) error {
		panic("boom")
	})

	select {
	case value := <-hookValue:
		if value != "req-ctx-2" {
			t.Fatalf("unexpected context value: %q", value)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("panic hook was not called")
	}
}
