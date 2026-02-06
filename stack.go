package safegroup

import (
	"fmt"
	"runtime"
	"strings"
)

// StackTrace holds program counters captured from runtime.Callers.
type StackTrace []uintptr

func captureStack(skip int) StackTrace {
	programCounters := make([]uintptr, 64)
	count := runtime.Callers(skip+2, programCounters)
	if count == 0 {
		return nil
	}
	stack := make(StackTrace, count)
	copy(stack, programCounters[:count])
	return stack
}

// String formats the stack as newline-separated frames.
func (st StackTrace) String() string {
	if len(st) == 0 {
		return ""
	}

	frames := runtime.CallersFrames(st)
	lines := make([]string, 0, len(st)*2)
	for {
		frame, more := frames.Next()
		lines = append(lines, frame.Function)
		lines = append(lines, fmt.Sprintf("\t%s:%d", frame.File, frame.Line))
		if !more {
			break
		}
	}

	return strings.Join(lines, "\n")
}
