package safegroup

import "errors"

// IsPanic reports whether err contains a PanicError.
func IsPanic(err error) bool {
	var panicErr *PanicError
	return errors.As(err, &panicErr)
}

// AsPanic returns the first PanicError found in err, or nil if none exists.
func AsPanic(err error) *PanicError {
	var panicErr *PanicError
	if errors.As(err, &panicErr) {
		return panicErr
	}
	return nil
}

// AllPanics returns all PanicError values found in err.
//
// It traverses both single unwrap chains and multi-error trees such as
// errors.Join.
func AllPanics(err error) []*PanicError {
	if err == nil {
		return nil
	}
	var result []*PanicError
	collectPanics(err, &result)
	return result
}

func collectPanics(err error, result *[]*PanicError) {
	if err == nil {
		return
	}

	if panicErr, ok := err.(*PanicError); ok {
		*result = append(*result, panicErr)
	}

	type manyUnwrapper interface {
		Unwrap() []error
	}
	if unwrappedMany, ok := err.(manyUnwrapper); ok {
		for _, child := range unwrappedMany.Unwrap() {
			collectPanics(child, result)
		}
		return
	}

	type oneUnwrapper interface {
		Unwrap() error
	}
	if unwrappedOne, ok := err.(oneUnwrapper); ok {
		collectPanics(unwrappedOne.Unwrap(), result)
	}
}
