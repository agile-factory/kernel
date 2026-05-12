package errors

import "net/http"

// ─── sentinel errors ────────────────────────────────────────────────────────
//
// Sentinels are pre-constructed *AppError values for use with errors.Is.
//
// Because AppError implements Unwrap(), the standard errors.Is / errors.As
// functions walk the chain automatically. Comparison is by pointer identity:
//
//	if errors.Is(err, ErrNotFound) { … }   // works if err is or wraps ErrNotFound
//
// IMPORTANT: sentinels are not captured with a call-site stack (their stack
// points here, inside the kernel). When a sentinel is *returned* from domain
// code it should be wrapped via Wrap() or compared with errors.Is — never
// re-used as the actual returned value if you need accurate location info.
//
// Pattern:
//
//	// domain/errors.go — define a domain sentinel
//	var ErrUserNotFound = errors.New("USER_NOT_FOUND", "user not found", http.StatusNotFound, nil)
//
//	// usecase/login.go — return a *fresh* error with accurate stack
//	return errors.WrapNotFound(err, "user not found during login")
//
//	// handler — check type
//	if errors.Is(err, ErrNotFound) { … }

// ErrNotFound is the generic resource-not-found sentinel.
var ErrNotFound = &AppError{
	Code:     CODE_NOT_FOUND,
	Message:  "resource not found",
	HTTPCode: http.StatusNotFound,
}

// ErrUnauthorized is the generic unauthenticated sentinel.
var ErrUnauthorized = &AppError{
	Code:     CODE_UNAUTHORIZED,
	Message:  "unauthorized",
	HTTPCode: http.StatusUnauthorized,
}

// ErrForbidden is the generic insufficient-permissions sentinel.
var ErrForbidden = &AppError{
	Code:     CODE_FORBIDDEN,
	Message:  "forbidden",
	HTTPCode: http.StatusForbidden,
}

// ErrBadRequest is the generic malformed-request sentinel.
var ErrBadRequest = &AppError{
	Code:     CODE_BAD_REQUEST,
	Message:  "bad request",
	HTTPCode: http.StatusBadRequest,
}

// ErrConflict is the generic state-conflict sentinel.
var ErrConflict = &AppError{
	Code:     CODE_CONFLICT,
	Message:  "conflict",
	HTTPCode: http.StatusConflict,
}

// ErrInternal is the generic internal-server-error sentinel.
var ErrInternal = &AppError{
	Code:     CODE_INTERNAL_SERVER_ERROR,
	Message:  "internal server error",
	HTTPCode: http.StatusInternalServerError,
}

// ErrServiceUnavailable is the generic downstream-unavailable sentinel.
var ErrServiceUnavailable = &AppError{
	Code:     CODE_SERVICE_UNAVAILABLE,
	Message:  "service unavailable",
	HTTPCode: http.StatusServiceUnavailable,
}

// ErrTooManyRequests is the generic rate-limit sentinel.
var ErrTooManyRequests = &AppError{
	Code:     CODE_TOO_MANY_REQUESTS,
	Message:  "too many requests",
	HTTPCode: http.StatusTooManyRequests,
}

// ─── Is / As helpers for sentinel matching ──────────────────────────────────

// Is reports whether target and err have the same Code.
// This makes errors.Is(err, ErrNotFound) match any AppError whose Code is
// CODE_NOT_FOUND, regardless of message or stack.
//
// Go's errors.Is calls this method automatically when it is defined.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}
