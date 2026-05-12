// Package errors provides the centralised, structured error type for the
// Agile Factory backend. Every error produced inside the system is either an
// *AppError or wraps one, so that:
//
//   - the HTTP delivery layer can map it to the correct status code,
//   - the logger receives a fully-structured slog.Value (no string parsing),
//   - post-mortem debugging has a precise stack trace pointing to the exact
//     line where the fault originated (not just where it was caught).
//
// Usage — creating a new error:
//
//	return errors.New("USER_NOT_FOUND", "user not found", http.StatusNotFound, nil)
//
// Usage — wrapping an infrastructure error:
//
//	return errors.Wrap(pgErr, "DB_QUERY_FAILED", "query failed", http.StatusInternalServerError)
//
// Usage — structured logging (automatic via slog.LogValuer):
//
//	slog.Error("request failed", "error", appErr)
package errors

import (
	"fmt"
	"log/slog"
	"time"
)

// AppError is the canonical error type for the entire application.
//
// Field visibility:
//   - Code, Message, HTTPCode: public — they form the external API contract.
//   - Cause: public — needed by errors.Unwrap / errors.As chains.
//   - Stack, OccurredAt, TraceID: public — needed by response / logger layers,
//     but NEVER serialised into production API responses (only logged).
type AppError struct {
	// Code is a stable, machine-readable identifier versioned as part of the
	// API contract (e.g. "USER_NOT_FOUND", "TENANT_QUOTA_EXCEEDED").
	Code string `json:"code"`

	// Message is a human-readable description safe to expose to API consumers.
	Message string `json:"message"`

	// HTTPCode is the HTTP status code this error maps to.
	HTTPCode int `json:"http_code"`

	// Cause is the underlying error that triggered this AppError, if any.
	// It is walked by errors.As / errors.Is via Unwrap().
	Cause error `json:"-"`

	// Stack is the call stack captured at the moment New() or Wrap() was called.
	// It points to the application call site, not to this package.
	Stack StackTrace `json:"-"`

	// OccurredAt is the wall-clock time at which the error was constructed.
	// Useful for log correlation without relying on log timestamp precision.
	OccurredAt time.Time `json:"-"`

	// TraceID is the distributed trace / request identifier, optionally
	// populated by the HTTP delivery layer from the incoming X-Request-ID header
	// or a tracing context. Empty string means not set.
	TraceID string `json:"-"`
}

// ─── core interface implementations ────────────────────────────────────────

// Error implements the built-in error interface.
// The string includes Code, Message, and the cause chain — never the stack
// (stacks appear only in structured logs, not in string-formatted errors).
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped cause, enabling errors.Is / errors.As to walk
// the entire error chain transparently.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// ─── slog integration ──────────────────────────────────────────────────────

// LogValue implements slog.LogValuer so that passing an *AppError directly to
// any slog call produces a fully-structured group — no manual field extraction.
//
//	slog.Error("handler failed", "error", appErr)
//
// Produces (JSON handler):
//
//	{
//	  "error": {
//	    "code":        "USER_NOT_FOUND",
//	    "message":     "user not found",
//	    "http_code":   404,
//	    "trace_id":    "req-abc-123",
//	    "occurred_at": "2026-04-30T16:04:09Z",
//	    "cause":       "…",
//	    "stack":       "usecase.(*LoginUseCase).Execute\n\t/…/login.go:44\n…"
//	  }
//	}
func (e *AppError) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.String("code", e.Code),
		slog.String("message", e.Message),
		slog.Int("http_code", e.HTTPCode),
	}

	if e.TraceID != "" {
		attrs = append(attrs, slog.String("trace_id", e.TraceID))
	}

	if !e.OccurredAt.IsZero() {
		attrs = append(attrs, slog.Time("occurred_at", e.OccurredAt))
	}

	if e.Cause != nil {
		attrs = append(attrs, slog.String("cause", e.Cause.Error()))
	}

	if len(e.Stack) > 0 {
		attrs = append(attrs, slog.String("stack", e.Stack.String()))
	}

	return slog.GroupValue(attrs...)
}

// ─── builder methods (fluent, return *AppError for chaining) ───────────────

// WithCause attaches or replaces the underlying cause.
// Returns the receiver for fluent chaining.
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithTraceID sets the request/trace identifier on the error.
// Typically called by the HTTP delivery layer:
//
//	appErr.WithTraceID(r.Header.Get("X-Request-ID"))
func (e *AppError) WithTraceID(id string) *AppError {
	e.TraceID = id
	return e
}

// WithMessage overrides the human-readable message (useful when a generic
// sentinel is reused with a more specific description).
func (e *AppError) WithMessage(msg string) *AppError {
	e.Message = msg
	return e
}

// ─── constructors ──────────────────────────────────────────────────────────

// New creates a new *AppError, capturing the call stack at the call site.
//
// The `cause` parameter accepts nil for errors with no underlying cause.
// skip=2 strips: captureStack + New itself, leaving the caller as frame 0.
//
// Backward-compatible signature: accepts the same four arguments as the
// previous implementation so that existing call sites (domain/errors.go,
// recovery.go, etc.) compile without changes.
func New(code, message string, httpCode int, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPCode:   httpCode,
		Cause:      cause,
		Stack:      captureStack(2),
		OccurredAt: time.Now().UTC(),
	}
}

// newSkip is an internal constructor used by named factory functions (e.g.
// NotFound, InternalServerError). It adds one extra skip level so that the
// factory function itself is not frame 0 of the trace — the application call
// site is.
func newSkip(code, message string, httpCode int, cause error) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPCode:   httpCode,
		Cause:      cause,
		Stack:      captureStack(3), // skip: captureStack + newSkip + named factory
		OccurredAt: time.Now().UTC(),
	}
}
