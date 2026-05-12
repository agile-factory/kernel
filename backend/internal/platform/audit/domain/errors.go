package domain

import "errors"

var (
	ErrAuditSaveFailed = errors.New("audit: failed to persist audit log")

	// ErrInvalidCursor is returned when the opaque cursor token cannot be decoded
	// (malformed Base64url, invalid JSON, or missing required fields). It is a
	// client-side error and must be mapped to HTTP 400.
	ErrInvalidCursor = errors.New("audit: invalid cursor token")
)
