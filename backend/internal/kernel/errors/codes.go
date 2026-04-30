package errors

import "net/http"

// Commonly used error codes
const (
	CODE_INTERNAL_SERVER_ERROR = "INTERNAL_SERVER_ERROR"
	CODE_NOT_FOUND            = "NOT_FOUND"
	CODE_BAD_REQUEST          = "BAD_REQUEST"
	CODE_UNAUTHORIZED        = "UNAUTHORIZED"
	CODE_FORBIDDEN           = "FORBIDDEN"
)

// InternalServerError returns a default internal server error
func InternalServerError(cause error) *AppError {
	return New(CODE_INTERNAL_SERVER_ERROR, "An unexpected error occurred", http.StatusInternalServerError, cause)
}

// NotFound returns a default not found error
func NotFound(message string) *AppError {
	return New(CODE_NOT_FOUND, message, http.StatusNotFound, nil)
}

// BadRequest returns a default bad request error
func BadRequest(message string) *AppError {
	return New(CODE_BAD_REQUEST, message, http.StatusBadRequest, nil)
}

// Unauthorized returns a default unauthorized error
func Unauthorized(message string) *AppError {
	return New(CODE_UNAUTHORIZED, message, http.StatusUnauthorized, nil)
}

// Forbidden returns a default forbidden error
func Forbidden(message string) *AppError {
	return New(CODE_FORBIDDEN, message, http.StatusForbidden, nil)
}
