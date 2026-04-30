package errors

import "fmt"

// AppError represents a domain error.
type AppError struct {
	Code     string // "USER_NOT_FOUND", "TENANT_QUOTA_EXCEEDED"
	Message  string
	HTTPCode int
	Cause    error
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// New creates a new AppError.
func New(code, message string, httpCode int, cause error) *AppError {
	return &AppError{
		Code:     code,
		Message:  message,
		HTTPCode: httpCode,
		Cause:    cause,
	}
}
