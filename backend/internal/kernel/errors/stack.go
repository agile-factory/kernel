package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// maxStackDepth is the maximum number of frames captured per error.
// 64 frames gives full goroutine call depth in virtually all real-world scenarios.
const maxStackDepth = 64

// StackFrame represents a single frame in a call stack.
type StackFrame struct {
	// Function is the fully-qualified function name, e.g.
	// "github.com/org/repo/internal/platform/auth/usecase.(*LoginUseCase).Execute".
	Function string `json:"function"`

	// File is the absolute source file path at compile time.
	File string `json:"file"`

	// Line is the line number within File.
	Line int `json:"line"`
}

// String renders the frame as a single human-readable line.
//
//	usecase.(*LoginUseCase).Execute
//	    /home/zone01/…/usecase/login.go:44
func (f StackFrame) String() string {
	return fmt.Sprintf("%s\n\t%s:%d", f.Function, f.File, f.Line)
}

// ShortFile returns the last two path segments of the file, e.g.
// "usecase/login.go". Useful in log output where full paths are noisy.
func (f StackFrame) ShortFile() string {
	parts := strings.Split(f.File, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return f.File
}

// ShortFunction returns only the package-relative function name, e.g.
// "usecase.(*LoginUseCase).Execute" (drops the module path prefix).
func (f StackFrame) ShortFunction() string {
	if idx := strings.LastIndex(f.Function, "/"); idx >= 0 {
		return f.Function[idx+1:]
	}
	return f.Function
}

// StackTrace is an ordered slice of StackFrame, outermost call first (index 0
// is the direct caller of New / Wrap / captureStack).
type StackTrace []StackFrame

// String formats the full stack trace in the style of Go's runtime/debug package.
func (st StackTrace) String() string {
	if len(st) == 0 {
		return "(no stack trace)"
	}
	var b strings.Builder
	for i, f := range st {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.String())
	}
	return b.String()
}

// captureStack records up to maxStackDepth frames, skipping `skip` frames from
// the bottom of the stack.
//
// Callers inside this package use skip=3 so that:
//
//	frame 0 → captureStack       (skipped)
//	frame 1 → New / Wrap / ...   (skipped)
//	frame 2 → the kernel helper  (skipped when called via a named factory)
//	frame 3 → actual application call site  ← frame 0 of the returned slice
//
// For direct calls to New() from application code, use skip=2.
func captureStack(skip int) StackTrace {
	pcs := make([]uintptr, maxStackDepth)
	// runtime.Callers itself occupies 1 frame, so add 1 to skip.
	n := runtime.Callers(skip+1, pcs)
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	trace := make(StackTrace, 0, n)

	for {
		frame, more := frames.Next()

		// Skip frames that are internal to the Go runtime or testing harness,
		// but keep everything above that so we don't lose context.
		if frame.Function == "" {
			if !more {
				break
			}
			continue
		}

		trace = append(trace, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})

		if !more {
			break
		}
	}

	return trace
}
