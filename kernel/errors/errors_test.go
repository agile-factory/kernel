package errors_test

import (
	stderrors "errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
)

// ─── AppError core ──────────────────────────────────────────────────────────

func TestNew_BasicFields(t *testing.T) {
	err := errors.New("MY_CODE", "my message", http.StatusBadRequest, nil)

	if err.Code != "MY_CODE" {
		t.Errorf("Code: got %q, want %q", err.Code, "MY_CODE")
	}
	if err.Message != "my message" {
		t.Errorf("Message: got %q, want %q", err.Message, "my message")
	}
	if err.HTTPCode != http.StatusBadRequest {
		t.Errorf("HTTPCode: got %d, want %d", err.HTTPCode, http.StatusBadRequest)
	}
	if err.Cause != nil {
		t.Errorf("Cause: expected nil, got %v", err.Cause)
	}
	if err.OccurredAt.IsZero() {
		t.Error("OccurredAt should not be zero")
	}
}

func TestNew_WithCause(t *testing.T) {
	root := stderrors.New("root cause")
	err := errors.New("WRAPPED", "wrap msg", http.StatusInternalServerError, root)

	if !stderrors.Is(err, root) {
		t.Error("errors.Is should find the root cause through Unwrap()")
	}
}

func TestNew_ErrorString(t *testing.T) {
	err := errors.New("CODE", "message", 400, nil)
	s := err.Error()
	if !strings.Contains(s, "CODE") || !strings.Contains(s, "message") {
		t.Errorf("Error() missing code or message: %q", s)
	}
}

func TestNew_ErrorStringWithCause(t *testing.T) {
	root := stderrors.New("disk full")
	err := errors.New("DB_FAILED", "query failed", 500, root)
	s := err.Error()
	if !strings.Contains(s, "disk full") {
		t.Errorf("Error() should include cause: %q", s)
	}
}

// ─── Stack accuracy ─────────────────────────────────────────────────────────

func TestNew_StackNonNil(t *testing.T) {
	err := errors.New("X", "x", 400, nil)
	if len(err.Stack) == 0 {
		t.Fatal("Stack should not be empty")
	}
}

func TestNew_StackFrame0IsCallSite(t *testing.T) {
	err := errors.New("X", "x", 400, nil)
	frame := err.Stack[0]

	// Frame 0 must point to THIS test file, not to errors.go or stack.go.
	if !strings.Contains(frame.File, "errors_test.go") {
		t.Errorf("Stack frame 0 expected to be in errors_test.go, got %q", frame.File)
	}
}

func TestFactory_StackFrame0IsCallSite(t *testing.T) {
	// Named factories (NotFound, InternalServerError, etc.) add an extra skip.
	// Frame 0 must still point to HERE, not to codes.go.
	err := errors.NotFound("not found")
	frame := err.Stack[0]

	if !strings.Contains(frame.File, "errors_test.go") {
		t.Errorf("Factory stack frame 0 expected errors_test.go, got %q", frame.File)
	}
}

func TestWrap_StackFrame0IsCallSite(t *testing.T) {
	root := stderrors.New("pg: connection refused")
	err := errors.WrapDBError(root)
	frame := err.Stack[0]

	if !strings.Contains(frame.File, "errors_test.go") {
		t.Errorf("Wrap stack frame 0 expected errors_test.go, got %q", frame.File)
	}
}

// ─── Builder methods ─────────────────────────────────────────────────────────

func TestWithTraceID(t *testing.T) {
	err := errors.New("X", "x", 400, nil).WithTraceID("req-abc-123")
	if err.TraceID != "req-abc-123" {
		t.Errorf("TraceID: got %q, want %q", err.TraceID, "req-abc-123")
	}
}

func TestWithMessage(t *testing.T) {
	err := errors.New("X", "original", 400, nil).WithMessage("overridden")
	if err.Message != "overridden" {
		t.Errorf("Message: got %q, want %q", err.Message, "overridden")
	}
}

func TestWithCause_Builder(t *testing.T) {
	root := stderrors.New("root")
	err := errors.New("X", "x", 500, nil).WithCause(root)
	if !stderrors.Is(err, root) {
		t.Error("WithCause: errors.Is should find root")
	}
}

// ─── errors.As / errors.Is ─────────────────────────────────────────────────

func TestErrorsAs_DirectAppError(t *testing.T) {
	orig := errors.NotFound("item")
	var appErr *errors.AppError
	if !stderrors.As(orig, &appErr) {
		t.Fatal("errors.As should succeed for *AppError")
	}
	if appErr.Code != errors.CODE_NOT_FOUND {
		t.Errorf("Code: got %q, want %q", appErr.Code, errors.CODE_NOT_FOUND)
	}
}

func TestErrorsAs_WrappedInPlainError(t *testing.T) {
	appErr := errors.Forbidden("access denied")
	wrapped := stderrors.Join(stderrors.New("outer"), appErr)

	var found *errors.AppError
	if !stderrors.As(wrapped, &found) {
		t.Fatal("errors.As should find *AppError inside joined error")
	}
}

func TestErrorsIs_SentinelByCode(t *testing.T) {
	err := errors.NotFound("specific message")

	if !stderrors.Is(err, errors.ErrNotFound) {
		t.Error("errors.Is should match ErrNotFound by code regardless of message")
	}
}

func TestErrorsIs_DoesNotCrossMatch(t *testing.T) {
	err := errors.Forbidden("no access")

	if stderrors.Is(err, errors.ErrNotFound) {
		t.Error("errors.Is(Forbidden, ErrNotFound) should be false")
	}
}

// ─── Wrap helpers ────────────────────────────────────────────────────────────

func TestWrapInternal_NilReturnsNil(t *testing.T) {
	if errors.WrapInternal(nil) != nil {
		t.Error("WrapInternal(nil) should return nil")
	}
}

func TestWrapInternal_PreservesOriginal(t *testing.T) {
	root := stderrors.New("io: broken pipe")
	appErr := errors.WrapInternal(root)

	if appErr.HTTPCode != http.StatusInternalServerError {
		t.Errorf("HTTPCode: got %d, want 500", appErr.HTTPCode)
	}
	if !stderrors.Is(appErr, root) {
		t.Error("Cause chain should include original error")
	}
}

func TestWrap_AppErrorPreservesTimestamp(t *testing.T) {
	inner := errors.NotFound("first")
	outer := errors.Wrap(inner, errors.CODE_DB_QUERY_FAILED, "db failed", 500)

	if outer.OccurredAt != inner.OccurredAt {
		t.Error("Wrap of AppError should preserve the original OccurredAt")
	}
}

func TestIsAppError_Found(t *testing.T) {
	root := stderrors.New("pg error")
	wrapped := errors.WrapDBError(root)

	appErr, ok := errors.IsAppError(wrapped)
	if !ok {
		t.Fatal("IsAppError should return true")
	}
	if appErr.Code != errors.CODE_DB_QUERY_FAILED {
		t.Errorf("Code: got %q, want %q", appErr.Code, errors.CODE_DB_QUERY_FAILED)
	}
}

func TestIsAppError_PlainError(t *testing.T) {
	_, ok := errors.IsAppError(stderrors.New("plain"))
	if ok {
		t.Error("IsAppError should return false for plain errors")
	}
}

// ─── HTTPCode / Code helpers ─────────────────────────────────────────────────

func TestHTTPCode_AppError(t *testing.T) {
	err := errors.Unauthorized("no token")
	if errors.HTTPCode(err) != http.StatusUnauthorized {
		t.Errorf("HTTPCode: got %d, want 401", errors.HTTPCode(err))
	}
}

func TestHTTPCode_PlainError(t *testing.T) {
	plain := stderrors.New("plain")
	if errors.HTTPCode(plain) != http.StatusInternalServerError {
		t.Errorf("HTTPCode(plain): got %d, want 500", errors.HTTPCode(plain))
	}
}

func TestCode_PlainError(t *testing.T) {
	plain := stderrors.New("plain")
	if errors.Code(plain) != errors.CODE_INTERNAL_SERVER_ERROR {
		t.Errorf("Code(plain): got %q", errors.Code(plain))
	}
}

// ─── StackFrame helpers ──────────────────────────────────────────────────────

func TestStackFrame_ShortFile(t *testing.T) {
	err := errors.New("X", "x", 400, nil)
	if len(err.Stack) == 0 {
		t.Skip("no stack frames")
	}
	short := err.Stack[0].ShortFile()
	if strings.Contains(short, "/home") {
		t.Errorf("ShortFile should not contain full path prefix, got %q", short)
	}
}

func TestStackTrace_String(t *testing.T) {
	err := errors.New("X", "x", 400, nil)
	s := err.Stack.String()
	if s == "(no stack trace)" {
		t.Error("Stack.String() should not be empty for a freshly created error")
	}
	if !strings.Contains(s, ".go:") {
		t.Errorf("Stack.String() should contain file:line, got: %q", s)
	}
}

// ─── slog.LogValuer ─────────────────────────────────────────────────────────

func TestLogValue_NotPanics(t *testing.T) {
	err := errors.New("X", "x", 400, stderrors.New("cause")).WithTraceID("tid-1")
	// LogValue must not panic; shape is validated by log/slog itself.
	val := err.LogValue()
	if val.Kind().String() == "" {
		t.Error("LogValue should return a non-empty slog.Value")
	}
}
