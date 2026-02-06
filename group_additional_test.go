package safegroup

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestSetLimitZeroRemovesLimit(t *testing.T) {
	group, _ := WithContext(context.Background())
	group.SetLimit(1)

	block := make(chan struct{})
	started := make(chan struct{})
	group.Go(func(context.Context) error {
		close(started)
		<-block
		return nil
	})
	<-started

	if group.TryGo(func(context.Context) error { return nil }) {
		t.Fatal("expected TryGo to fail while limit is full")
	}

	close(block)
	if err := group.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}

	group.SetLimit(0)
	if !group.TryGo(func(context.Context) error { return nil }) {
		t.Fatal("expected TryGo to succeed after removing limit")
	}
	if !group.TryGo(func(context.Context) error { return nil }) {
		t.Fatal("expected TryGo to succeed without limit")
	}
	if err := group.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
}

func TestCaptureStackDisabledOmitsStack(t *testing.T) {
	group, _ := WithContext(context.Background(), CaptureStack(false))
	group.Go(func(context.Context) error {
		panic("boom")
	})

	err := group.Wait()
	panicErr := AsPanic(err)
	if panicErr == nil {
		t.Fatal("expected panic error")
	}
	if len(panicErr.Stack) != 0 {
		t.Fatalf("expected no stack, got %d frames", len(panicErr.Stack))
	}
}

func TestNilHooksFallbackToNoop(t *testing.T) {
	group, _ := WithContext(context.Background(), OnError(nil), OnPanic(nil))
	group.Go(func(context.Context) error { return errors.New("x") })
	group.Go(func(context.Context) error { panic("y") })

	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}
}

func TestTryGoLabelPreservesLabelOnPanic(t *testing.T) {
	group, _ := WithContext(context.Background())
	if !group.TryGoLabel("worker-a", func(context.Context) error {
		panic("boom")
	}) {
		t.Fatal("expected TryGoLabel to start task")
	}

	err := group.Wait()
	panicErr := AsPanic(err)
	if panicErr == nil {
		t.Fatal("expected panic error")
	}
	if panicErr.Label != "worker-a" {
		t.Fatalf("unexpected label: %q", panicErr.Label)
	}
}

func TestWithContextParentAlreadyCanceled(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()

	group, ctx := WithContext(parent)
	group.Go(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return nil
		default:
			return fmt.Errorf("expected canceled context")
		}
	})

	if err := group.Wait(); err != nil {
		t.Fatalf("unexpected wait error: %v", err)
	}
	if ctx.Err() == nil {
		t.Fatal("expected derived context to be canceled")
	}
}

func TestPanicErrorFormatVerbs(t *testing.T) {
	panicErr := &PanicError{
		Label: "job-1",
		Value: "boom",
		Stack: StackTrace{1},
	}

	if got := fmt.Sprintf("%s", panicErr); got == "" {
		t.Fatalf("expected %%s output")
	}
	if got := fmt.Sprintf("%q", panicErr); got == "" {
		t.Fatalf("expected %%q output")
	}
	if got := fmt.Sprintf("%v", panicErr); got == "" {
		t.Fatalf("expected %%v output")
	}
	if got := fmt.Sprintf("%+v", panicErr); got == "" {
		t.Fatalf("expected %%+v output")
	}
}

func TestPanicErrorFormatNilReceiver(t *testing.T) {
	var panicErr *PanicError

	if got := fmt.Sprintf("%v", panicErr); got != "<nil>" {
		t.Fatalf("unexpected %%v output: %q", got)
	}
	if got := fmt.Sprintf("%+v", panicErr); got != "<nil>" {
		t.Fatalf("unexpected %%+v output: %q", got)
	}
}

func TestGoLabelReturnsOnContextCancelWhileWaitingLimit(t *testing.T) {
	group, _ := WithContext(context.Background())
	group.SetLimit(1)

	panicNow := make(chan struct{})
	started := make(chan struct{})
	group.Go(func(context.Context) error {
		close(started)
		<-panicNow
		panic("boom")
	})
	<-started

	blockedReturned := make(chan struct{})
	secondRan := make(chan struct{})
	go func() {
		group.GoLabel("blocked", func(context.Context) error {
			close(secondRan)
			return nil
		})
		close(blockedReturned)
	}()

	close(panicNow)

	select {
	case <-blockedReturned:
	case <-time.After(1 * time.Second):
		t.Fatal("GoLabel did not return after group context cancellation")
	}

	if err := group.Wait(); AsPanic(err) == nil {
		t.Fatalf("expected panic error, got: %v", err)
	}

	select {
	case <-secondRan:
		t.Fatal("second task should not run when canceled while waiting for limit")
	default:
	}
}

func TestWaitCanBeCalledMultipleTimes(t *testing.T) {
	group, _ := WithContext(context.Background())
	taskErr := errors.New("task failed")
	group.Go(func(context.Context) error {
		return taskErr
	})

	first := group.Wait()
	second := group.Wait()
	if first == nil || second == nil {
		t.Fatalf("expected non-nil errors, got first=%v second=%v", first, second)
	}
	if first.Error() != second.Error() {
		t.Fatalf("expected equivalent results, got first=%q second=%q", first.Error(), second.Error())
	}
}

func TestErrorsAndPanicsSnapshotBeforeWait(t *testing.T) {
	group, _ := WithContext(context.Background())
	release := make(chan struct{})
	group.Go(func(context.Context) error {
		<-release
		return errors.New("done")
	})

	if got := group.Errors(); len(got) != 0 {
		t.Fatalf("expected no collected errors before completion, got %d", len(got))
	}
	if got := group.Panics(); len(got) != 0 {
		t.Fatalf("expected no collected panics before completion, got %d", len(got))
	}

	close(release)
	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}
	if got := group.Errors(); len(got) != 1 {
		t.Fatalf("expected one collected error after Wait, got %d", len(got))
	}
}

func TestErrorsKeepAppendOrderWithMixedFailures(t *testing.T) {
	group, _ := WithContext(
		context.Background(),
		CancelOnError(false),
		CancelOnPanic(false),
		CaptureStack(false),
	)
	group.SetLimit(1)

	firstErr := errors.New("first")
	thirdErr := errors.New("third")
	group.GoLabel("err-1", func(context.Context) error {
		return firstErr
	})
	group.GoLabel("panic-2", func(context.Context) error {
		panic("second")
	})
	group.GoLabel("err-3", func(context.Context) error {
		return thirdErr
	})

	if err := group.Wait(); err == nil {
		t.Fatal("expected joined error")
	}

	failures := group.Errors()
	if len(failures) != 3 {
		t.Fatalf("expected 3 failures, got %d", len(failures))
	}
	if !errors.Is(failures[0], firstErr) {
		t.Fatalf("unexpected first failure: %v", failures[0])
	}
	panicErr, ok := failures[1].(*PanicError)
	if !ok || panicErr.Label != "panic-2" {
		t.Fatalf("unexpected second failure: %T %v", failures[1], failures[1])
	}
	if !errors.Is(failures[2], thirdErr) {
		t.Fatalf("unexpected third failure: %v", failures[2])
	}
}

func TestSetLimitNegativePanics(t *testing.T) {
	group, _ := WithContext(context.Background())

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if got, ok := recovered.(string); !ok || got != "safegroup: negative limit" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	group.SetLimit(-1)
}

func TestGoNilTaskPanics(t *testing.T) {
	group, _ := WithContext(context.Background())

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if got, ok := recovered.(string); !ok || got != "safegroup: nil task" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	group.Go(nil)
}

func TestTryGoNilTaskPanics(t *testing.T) {
	group, _ := WithContext(context.Background())

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		if got, ok := recovered.(string); !ok || got != "safegroup: nil task" {
			t.Fatalf("unexpected panic: %v", recovered)
		}
	}()

	group.TryGo(nil)
}

func TestPanicErrorNilReceiverMethods(t *testing.T) {
	var panicErr *PanicError

	if got := panicErr.Error(); got != "<nil>" {
		t.Fatalf("unexpected Error output: %q", got)
	}
	if got := panicErr.Unwrap(); got != nil {
		t.Fatalf("unexpected Unwrap output: %v", got)
	}
}

func TestAllPanicsNil(t *testing.T) {
	if got := AllPanics(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
