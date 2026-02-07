package safegroup

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func TestWaitJoinsAllPanics(t *testing.T) {
	group, _ := WithContext(context.Background(), CancelOnPanic(false))

	group.GoLabel("task-1", func(context.Context) error {
		panic("first")
	})
	group.GoLabel("task-2", func(context.Context) error {
		panic("second")
	})

	err := group.Wait()
	if err == nil {
		t.Fatal("expected joined error")
	}

	panics := AllPanics(err)
	if len(panics) != 2 {
		t.Fatalf("expected 2 panics, got %d", len(panics))
	}
}

func TestErrorsAsFindsPanicError(t *testing.T) {
	group, _ := WithContext(context.Background())

	group.Go(func(context.Context) error {
		panic("boom")
	})

	err := group.Wait()
	if err == nil {
		t.Fatal("expected panic converted to error")
	}

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatal("expected errors.As to find PanicError")
	}
	if panicErr.Label != "" {
		t.Fatalf("unexpected label: %q", panicErr.Label)
	}
}

func TestAllPanicsReturnsEveryPanic(t *testing.T) {
	group, _ := WithContext(context.Background(), CancelOnPanic(false))

	group.Go(func(context.Context) error {
		panic(errors.New("wrapped"))
	})
	group.Go(func(context.Context) error {
		panic("text panic")
	})

	err := group.Wait()
	panics := AllPanics(err)
	if len(panics) != 2 {
		t.Fatalf("expected 2 panics, got %d", len(panics))
	}
	if panics[0].Unwrap() == nil && panics[1].Unwrap() == nil {
		t.Fatal("expected one panic to unwrap to error")
	}
}

func TestTryGoWithSetLimit(t *testing.T) {
	group, _ := WithContext(context.Background())
	group.SetLimit(1)

	blocker := make(chan struct{})
	started := make(chan struct{})
	group.Go(func(context.Context) error {
		close(started)
		<-blocker
		return nil
	})
	<-started

	if ok := group.TryGo(func(context.Context) error { return nil }); ok {
		t.Fatal("expected TryGo to fail when limit is full")
	}

	close(blocker)
	if err := group.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestCancelOptionsControlContextCancellation(t *testing.T) {
	t.Run("CancelOnError", func(t *testing.T) {
		group, ctx := WithContext(context.Background(), CancelOnError(true))
		group.Go(func(context.Context) error {
			return errors.New("failure")
		})
		group.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		})

		_ = group.Wait()
		select {
		case <-ctx.Done():
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected context cancellation on error")
		}
	})

	t.Run("DisableCancelOnError", func(t *testing.T) {
		group, ctx := WithContext(context.Background(), CancelOnError(false))
		released := make(chan struct{})
		finished := make(chan struct{})
		group.Go(func(context.Context) error {
			return errors.New("failure")
		})
		group.Go(func(ctx context.Context) error {
			defer close(finished)
			select {
			case <-released:
				return nil
			case <-ctx.Done():
				return fmt.Errorf("unexpected cancellation: %w", ctx.Err())
			}
		})

		time.Sleep(50 * time.Millisecond)
		select {
		case <-ctx.Done():
			t.Fatal("did not expect early cancellation")
		default:
		}
		close(released)
		if err := group.Wait(); err == nil {
			t.Fatal("expected joined error")
		}
		<-finished
	})
}

func TestHooksCalledForPanicAndError(t *testing.T) {
	var panicCalls int32
	var errorCalls int32

	group, _ := WithContext(
		context.Background(),
		CancelOnError(false),
		CancelOnPanic(false),
		OnPanic(func(*PanicError) { atomic.AddInt32(&panicCalls, 1) }),
		OnError(func(error) { atomic.AddInt32(&errorCalls, 1) }),
	)

	group.Go(func(context.Context) error {
		return errors.New("regular")
	})
	group.Go(func(context.Context) error {
		panic("panic")
	})

	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}
	if got := atomic.LoadInt32(&panicCalls); got != 1 {
		t.Fatalf("expected one panic hook call, got %d", got)
	}
	if got := atomic.LoadInt32(&errorCalls); got != 1 {
		t.Fatalf("expected one error hook call, got %d", got)
	}
}

func TestContextHooksCalledForPanicAndError(t *testing.T) {
	var panicCalls int32
	var errorCalls int32

	group, _ := WithContext(
		context.Background(),
		CancelOnError(false),
		CancelOnPanic(false),
		OnPanicWithContext(func(context.Context, *PanicError) { atomic.AddInt32(&panicCalls, 1) }),
		OnErrorWithContext(func(context.Context, error) { atomic.AddInt32(&errorCalls, 1) }),
	)

	group.Go(func(context.Context) error {
		return errors.New("regular")
	})
	group.Go(func(context.Context) error {
		panic("panic")
	})

	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}
	if got := atomic.LoadInt32(&panicCalls); got != 1 {
		t.Fatalf("expected one panic hook call, got %d", got)
	}
	if got := atomic.LoadInt32(&errorCalls); got != 1 {
		t.Fatalf("expected one error hook call, got %d", got)
	}
}

func TestOnErrorWithContextRunsBeforeSelfCancel(t *testing.T) {
	hookCtxErr := make(chan error, 1)
	group, _ := WithContext(
		context.Background(),
		CancelOnError(true),
		OnErrorWithContext(func(ctx context.Context, _ error) {
			hookCtxErr <- ctx.Err()
		}),
	)

	group.Go(func(context.Context) error {
		return errors.New("task failed")
	})
	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}

	select {
	case err := <-hookCtxErr:
		if err != nil {
			t.Fatalf("expected hook context not canceled by same error yet, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("error hook was not called")
	}
}

func TestOnErrorPanicIsNotRecovered(t *testing.T) {
	group, _ := WithContext(
		context.Background(),
		CancelOnError(false),
		CancelOnPanic(false),
		OnError(func(error) { panic("hook panic") }),
	)

	taskErr := errors.New("task failure")

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if got, ok := recovered.(string); !ok || got != "hook panic" {
			t.Fatalf("unexpected panic: %v", recovered)
		}

		failures := group.Errors()
		if len(failures) != 1 {
			t.Fatalf("expected only task error to be recorded, got %d", len(failures))
		}
		if !errors.Is(failures[0], taskErr) {
			t.Fatalf("unexpected recorded failure: %v", failures[0])
		}
		if got := group.Panics(); len(got) != 0 {
			t.Fatalf("expected no recorded panic errors, got %d", len(got))
		}
	}()

	group.waitGroup.Add(1)
	group.runTask("worker-a", func(context.Context) error {
		return taskErr
	})
}

func TestOnErrorWithContextPanicStillCancelsGroup(t *testing.T) {
	group, ctx := WithContext(
		context.Background(),
		CancelOnError(true),
		OnErrorWithContext(func(context.Context, error) { panic("hook panic") }),
	)

	taskErr := errors.New("task failure")

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if got, ok := recovered.(string); !ok || got != "hook panic" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
		if ctx.Err() == nil {
			t.Fatal("expected context to be canceled")
		}
	}()

	group.waitGroup.Add(1)
	group.runTask("worker-a", func(context.Context) error {
		return taskErr
	})
}
