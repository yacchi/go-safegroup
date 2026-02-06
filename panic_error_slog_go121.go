//go:build go1.21

package safegroup

import "log/slog"

// LogValue returns a structured slog value for PanicError.
//
// The output includes a summary message, label (when set), panic value,
// and stack trace (when captured).
func (pe *PanicError) LogValue() slog.Value {
	if pe == nil {
		return slog.AnyValue(nil)
	}

	attrs := []slog.Attr{
		slog.String("message", pe.Error()),
		slog.Any("value", pe.Value),
	}
	if pe.Label != "" {
		attrs = append(attrs, slog.String("label", pe.Label))
	}
	if len(pe.Stack) > 0 {
		attrs = append(attrs, slog.Any("stack", pe.Stack))
	}
	return slog.GroupValue(attrs...)
}

// LogValue returns the stack trace as a single string for slog output.
func (st StackTrace) LogValue() slog.Value {
	if len(st) == 0 {
		return slog.StringValue("")
	}
	return slog.StringValue(st.String())
}
