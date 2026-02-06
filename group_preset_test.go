package safegroup

import (
	"context"
	"errors"
	"testing"
	"time"
)

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

	preset.Go(func() error {
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

	preset.GoLabel("worker-a", func() error {
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
	preset.Go(func() error {
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
	preset.GoLabel("worker-a", func() error {
		close(taskDone)
		return nil
	})

	select {
	case <-taskDone:
	case <-time.After(1 * time.Second):
		t.Fatal("task did not run")
	}
}
