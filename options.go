package safegroup

// Option mutates Group behavior at construction time.
type Option func(*config)

type config struct {
	cancelOnError bool
	cancelOnPanic bool
	captureStack  bool
	onPanic       func(*PanicError)
	onError       func(error)
}

func noopPanicHandler(*PanicError) {}

func noopErrorHandler(error) {}

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
func OnPanic(fn func(*PanicError)) Option {
	return func(cfg *config) {
		if fn == nil {
			fn = noopPanicHandler
		}
		cfg.onPanic = fn
	}
}

// OnError registers a hook called when a task returns a non-nil error.
func OnError(fn func(error)) Option {
	return func(cfg *config) {
		if fn == nil {
			fn = noopErrorHandler
		}
		cfg.onError = fn
	}
}
