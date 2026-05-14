package errors

import (
	"errors"
	"net/http"
	"time"
)

// ─── wrapping helpers ────────────────────────────────────────────────────────
//
// Use these when converting a lower-level error (stdlib, pgx, bcrypt…) into a
// typed *AppError without losing the original error in the chain.
//
// The stack captured here points to the application call site (the function
// calling Wrap), not to this file, because newSkip strips two extra frames.

// Wrap converts any error into an *AppError with the given classification.
// The original error is preserved as Cause and walked by errors.As / errors.Is.
//
//	pgErr := db.QueryRow(…)
//	return errors.Wrap(pgErr, errors.CODE_DB_QUERY_FAILED, "failed to fetch user", http.StatusInternalServerError)
func Wrap(err error, code, message string, httpCode int) *AppError {
	if err == nil {
		return nil
	}
	// If the error is already an *AppError, preserve its stack; only update
	// the outer classification so the caller's intent is recorded.
	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:       code,
			Message:    message,
			HTTPCode:   httpCode,
			Cause:      appErr,
			Stack:      captureStack(2), // call site is the direct caller of Wrap
			OccurredAt: appErr.OccurredAt, // preserve original timestamp
		}
	}
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPCode:   httpCode,
		Cause:      err,
		Stack:      captureStack(2),
		OccurredAt: nowUTC(),
	}
}

// WrapInternal wraps any unexpected infrastructure error as INTERNAL_SERVER_ERROR.
// The original error is attached as Cause for internal logging but never
// surfaced in API responses.
//
//	if err != nil {
//	    return errors.WrapInternal(err)
//	}
func WrapInternal(err error) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       CODE_INTERNAL_SERVER_ERROR,
		Message:    "An unexpected error occurred",
		HTTPCode:   http.StatusInternalServerError,
		Cause:      err,
		Stack:      captureStack(2),
		OccurredAt: nowUTC(),
	}
}

// WrapNotFound wraps an error as NOT_FOUND with a custom message.
func WrapNotFound(err error, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       CODE_NOT_FOUND,
		Message:    message,
		HTTPCode:   http.StatusNotFound,
		Cause:      err,
		Stack:      captureStack(2),
		OccurredAt: nowUTC(),
	}
}

// WrapBadRequest wraps an error as BAD_REQUEST with a custom message.
func WrapBadRequest(err error, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       CODE_BAD_REQUEST,
		Message:    message,
		HTTPCode:   http.StatusBadRequest,
		Cause:      err,
		Stack:      captureStack(2),
		OccurredAt: nowUTC(),
	}
}

// WrapUnauthorized wraps an error as UNAUTHORIZED.
func WrapUnauthorized(err error, message string) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       CODE_UNAUTHORIZED,
		Message:    message,
		HTTPCode:   http.StatusUnauthorized,
		Cause:      err,
		Stack:      captureStack(2),
		OccurredAt: nowUTC(),
	}
}

// WrapDBError wraps a database driver error as DB_QUERY_FAILED.
// Use this at the infrastructure boundary so the domain never sees raw DB errors.
//
//	row, err := db.QueryRow(ctx, query, args...)
//	if err != nil {
//	    return nil, errors.WrapDBError(err)
//	}
func WrapDBError(err error) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{
		Code:       CODE_DB_QUERY_FAILED,
		Message:    "A database error occurred",
		HTTPCode:   http.StatusInternalServerError,
		Cause:      err,
		Stack:      captureStack(2),
		OccurredAt: nowUTC(),
	}
}

// ─── inspection helpers ─────────────────────────────────────────────────────

// IsAppError reports whether err (or any error in its chain) is an *AppError
// and returns it if found.
//
//	if appErr, ok := errors.IsAppError(err); ok {
//	    log.Error("app error", "error", appErr)
//	}
func IsAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// HTTPCode returns the HTTP status code for any error.
// If err is (or wraps) an *AppError its HTTPCode is returned;
// otherwise http.StatusInternalServerError is returned.
func HTTPCode(err error) int {
	if appErr, ok := IsAppError(err); ok {
		return appErr.HTTPCode
	}
	return http.StatusInternalServerError
}

// Code returns the stable error code string for any error.
// Falls back to CODE_INTERNAL_SERVER_ERROR for plain errors.
func Code(err error) string {
	if appErr, ok := IsAppError(err); ok {
		return appErr.Code
	}
	return CODE_INTERNAL_SERVER_ERROR
}

// ─── internal helpers ────────────────────────────────────────────────────────

// nowUTC is a thin wrapper so tests can trivially intercept time if needed.
func nowUTC() time.Time {
	return time.Now().UTC()
}
