package safegroup

import (
	"fmt"
	"io"
)

// PanicError represents a recovered panic from a task.
type PanicError struct {
	// Label is copied from GoLabel/TryGoLabel.
	Label string
	// Value is the recovered panic value.
	Value any
	// Stack is the captured stack trace when CaptureStack(true).
	Stack StackTrace
}

// Error returns the panic summary text.
func (pe *PanicError) Error() string {
	if pe == nil {
		return "<nil>"
	}
	if pe.Label == "" {
		return fmt.Sprintf("panic: %v", pe.Value)
	}
	return fmt.Sprintf("panic in %q: %v", pe.Label, pe.Value)
}

// Unwrap returns Value when it is an error; otherwise nil.
func (pe *PanicError) Unwrap() error {
	if pe == nil {
		return nil
	}
	unwrapped, ok := pe.Value.(error)
	if !ok {
		return nil
	}
	return unwrapped
}

// Format prints panic details.
//
// `%v` and `%s` print the summary text. `%+v` also prints the stack trace.
func (pe *PanicError) Format(state fmt.State, verb rune) {
	if pe == nil {
		_, _ = io.WriteString(state, "<nil>")
		return
	}

	switch verb {
	case 'v':
		if state.Flag('+') {
			_, _ = io.WriteString(state, pe.Error())
			if len(pe.Stack) > 0 {
				_, _ = io.WriteString(state, "\n")
				_, _ = io.WriteString(state, pe.Stack.String())
			}
			return
		}
		_, _ = io.WriteString(state, pe.Error())
	case 's':
		_, _ = io.WriteString(state, pe.Error())
	case 'q':
		_, _ = fmt.Fprintf(state, "%q", pe.Error())
	default:
		_, _ = io.WriteString(state, pe.Error())
	}
}
