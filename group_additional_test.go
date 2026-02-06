package safegroup

import (
	"context"
	"errors"
	"fmt"
	"testing"
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
