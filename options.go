package safegroup

import "context"

// Option mutates Group behavior at construction time.
type Option func(*config)

type config struct {
	cancelOnError bool
	cancelOnPanic bool
	captureStack  bool
	onPanic       func(context.Context, *PanicError)
	onError       func(context.Context, error)
}

func noopPanicHandler(context.Context, *PanicError) {}

func noopErrorHandler(context.Context, error) {}

func defaultConfig() config {
	return config{
		cancelOnError: true,
		cancelOnPanic: true,
		captureStack:  true,
		onPanic:       noopPanicHandler,
		onError:       noopErrorHandler,
	}
}

// CancelOnError configures whether a non-nil task error cancels the group
// context. The default is true.
func CancelOnError(enabled bool) Option {
	return func(cfg *config) {
		cfg.cancelOnError = enabled
	}
}

// CancelOnPanic configures whether a recovered panic cancels the group context.
// The default is true.
func CancelOnPanic(enabled bool) Option {
	return func(cfg *config) {
		cfg.cancelOnPanic = enabled
	}
}

// CaptureStack configures whether panic stack traces are captured in PanicError.
// The default is true.
func CaptureStack(enabled bool) Option {
	return func(cfg *config) {
		cfg.captureStack = enabled
	}
}

// OnPanic registers a hook called when a task panic is recovered.
//
// Panics in the hook itself are not recovered by Group.
func OnPanic(fn func(*PanicError)) Option {
	if fn == nil {
		return OnPanicWithContext(nil)
	}
	return OnPanicWithContext(func(_ context.Context, panicErr *PanicError) {
		fn(panicErr)
	})
}

// OnError registers a hook called when a task returns a non-nil error.
//
// Panics in the hook itself are not recovered by Group.
func OnError(fn func(error)) Option {
	if fn == nil {
		return OnErrorWithContext(nil)
	}
	return OnErrorWithContext(func(_ context.Context, err error) {
		fn(err)
	})
}

// OnPanicWithContext registers a hook called when a task panic is recovered.
//
// Panics in the hook itself are not recovered by Group.
func OnPanicWithContext(fn func(context.Context, *PanicError)) Option {
	return func(cfg *config) {
		if fn == nil {
			fn = noopPanicHandler
		}
		cfg.onPanic = fn
	}
}

// OnErrorWithContext registers a hook called when a task returns a non-nil
// error.
//
// Panics in the hook itself are not recovered by Group.
func OnErrorWithContext(fn func(context.Context, error)) Option {
	return func(cfg *config) {
		if fn == nil {
			fn = noopErrorHandler
		}
		cfg.onError = fn
	}
}
