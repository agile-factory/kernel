package errors

import "net/http"

// ─── error code constants ───────────────────────────────────────────────────
//
// All codes follow SCREAMING_SNAKE_CASE (project convention).
// They are versioned as part of the external API contract: once published,
// a code must never be renamed — only deprecated and supplemented.

// General / infrastructure
const (
	CODE_INTERNAL_SERVER_ERROR = "INTERNAL_SERVER_ERROR"
	CODE_NOT_FOUND             = "NOT_FOUND"
	CODE_BAD_REQUEST           = "BAD_REQUEST"
	CODE_UNAUTHORIZED          = "UNAUTHORIZED"
	CODE_FORBIDDEN             = "FORBIDDEN"
	CODE_CONFLICT              = "CONFLICT"
	CODE_UNPROCESSABLE_ENTITY  = "UNPROCESSABLE_ENTITY"
	CODE_TOO_MANY_REQUESTS     = "TOO_MANY_REQUESTS"
	CODE_SERVICE_UNAVAILABLE   = "SERVICE_UNAVAILABLE"
	CODE_TIMEOUT               = "TIMEOUT"
	CODE_VALIDATION_FAILED     = "VALIDATION_FAILED"
)

// Auth / identity domain
const (
	CODE_INVALID_CREDENTIALS  = "INVALID_CREDENTIALS"
	CODE_INVALID_TOKEN        = "INVALID_TOKEN"
	CODE_TOKEN_EXPIRED        = "TOKEN_EXPIRED"
	CODE_USER_NOT_FOUND       = "USER_NOT_FOUND"
	CODE_USER_ALREADY_EXISTS  = "USER_ALREADY_EXISTS"
	CODE_INVALID_EMAIL        = "INVALID_EMAIL"
	CODE_INVALID_PASSWORD     = "INVALID_PASSWORD"
	CODE_ACCOUNT_DISABLED     = "ACCOUNT_DISABLED"
	CODE_SESSION_NOT_FOUND    = "SESSION_NOT_FOUND"
)

// Tenant / multi-tenancy domain
const (
	CODE_TENANT_NOT_FOUND       = "TENANT_NOT_FOUND"
	CODE_TENANT_SUSPENDED       = "TENANT_SUSPENDED"
	CODE_TENANT_QUOTA_EXCEEDED  = "TENANT_QUOTA_EXCEEDED"
	CODE_TENANT_ALREADY_EXISTS  = "TENANT_ALREADY_EXISTS"
)

// Billing domain
const (
	CODE_PAYMENT_REQUIRED       = "PAYMENT_REQUIRED"
	CODE_SUBSCRIPTION_INACTIVE  = "SUBSCRIPTION_INACTIVE"
	CODE_INVOICE_NOT_FOUND      = "INVOICE_NOT_FOUND"
)

// Permission / RBAC domain
const (
	CODE_PERMISSION_DENIED    = "PERMISSION_DENIED"
	CODE_ROLE_NOT_FOUND       = "ROLE_NOT_FOUND"
	CODE_INSUFFICIENT_SCOPE   = "INSUFFICIENT_SCOPE"
)

// Data / persistence domain
const (
	CODE_DB_QUERY_FAILED     = "DB_QUERY_FAILED"
	CODE_DB_CONNECTION_LOST  = "DB_CONNECTION_LOST"
	CODE_RECORD_NOT_FOUND    = "RECORD_NOT_FOUND"
	CODE_DUPLICATE_ENTRY     = "DUPLICATE_ENTRY"
	CODE_STALE_DATA          = "STALE_DATA"
)

// ─── general-purpose factory functions ─────────────────────────────────────
//
// Each factory calls newSkip so the captured stack frame 0 points to the
// caller of the factory, not to the factory itself.

// InternalServerError wraps an unexpected infrastructure error.
// The cause is logged but never surfaced to API consumers.
func InternalServerError(cause error) *AppError {
	return newSkip(CODE_INTERNAL_SERVER_ERROR, "An unexpected error occurred", http.StatusInternalServerError, cause)
}

// NotFound signals that the requested resource does not exist.
func NotFound(message string) *AppError {
	return newSkip(CODE_NOT_FOUND, message, http.StatusNotFound, nil)
}

// BadRequest signals a malformed or unacceptable client request.
func BadRequest(message string) *AppError {
	return newSkip(CODE_BAD_REQUEST, message, http.StatusBadRequest, nil)
}

// Unauthorized signals missing or invalid authentication credentials.
func Unauthorized(message string) *AppError {
	return newSkip(CODE_UNAUTHORIZED, message, http.StatusUnauthorized, nil)
}

// Forbidden signals that the authenticated principal lacks the required
// permission for this operation.
func Forbidden(message string) *AppError {
	return newSkip(CODE_FORBIDDEN, message, http.StatusForbidden, nil)
}

// Conflict signals a state conflict, e.g. duplicate resource creation.
func Conflict(message string) *AppError {
	return newSkip(CODE_CONFLICT, message, http.StatusConflict, nil)
}

// UnprocessableEntity signals that the request is syntactically valid but
// semantically incorrect (validation failures, business-rule violations).
func UnprocessableEntity(message string) *AppError {
	return newSkip(CODE_UNPROCESSABLE_ENTITY, message, http.StatusUnprocessableEntity, nil)
}

// ValidationFailed signals one or more field-level validation errors.
func ValidationFailed(message string) *AppError {
	return newSkip(CODE_VALIDATION_FAILED, message, http.StatusUnprocessableEntity, nil)
}

// TooManyRequests signals that the caller has exceeded a rate limit.
func TooManyRequests(message string) *AppError {
	return newSkip(CODE_TOO_MANY_REQUESTS, message, http.StatusTooManyRequests, nil)
}

// ServiceUnavailable signals a temporary downstream dependency failure.
func ServiceUnavailable(message string) *AppError {
	return newSkip(CODE_SERVICE_UNAVAILABLE, message, http.StatusServiceUnavailable, nil)
}

// Timeout signals that an operation exceeded its deadline.
func Timeout(message string) *AppError {
	return newSkip(CODE_TIMEOUT, message, http.StatusGatewayTimeout, nil)
}

// ─── domain-specific factory functions ─────────────────────────────────────

// InvalidCredentials is returned during login when email/password are wrong.
func InvalidCredentials() *AppError {
	return newSkip(CODE_INVALID_CREDENTIALS, "Invalid email or password", http.StatusUnauthorized, nil)
}

// InvalidToken is returned when a JWT or API token cannot be parsed or verified.
func InvalidToken(cause error) *AppError {
	return newSkip(CODE_INVALID_TOKEN, "The provided token is invalid", http.StatusUnauthorized, cause)
}

// TokenExpired is returned when a JWT has passed its expiry time.
func TokenExpired() *AppError {
	return newSkip(CODE_TOKEN_EXPIRED, "The token has expired", http.StatusUnauthorized, nil)
}

// UserNotFound is returned when a user lookup yields no result.
func UserNotFound() *AppError {
	return newSkip(CODE_USER_NOT_FOUND, "User not found", http.StatusNotFound, nil)
}

// UserAlreadyExists is returned on duplicate registration.
func UserAlreadyExists() *AppError {
	return newSkip(CODE_USER_ALREADY_EXISTS, "A user with this email already exists", http.StatusConflict, nil)
}

// TenantNotFound is returned when a tenant_id in the context cannot be resolved.
func TenantNotFound() *AppError {
	return newSkip(CODE_TENANT_NOT_FOUND, "Tenant not found", http.StatusNotFound, nil)
}

// TenantQuotaExceeded is returned when a tenant exceeds a resource limit.
func TenantQuotaExceeded(message string) *AppError {
	return newSkip(CODE_TENANT_QUOTA_EXCEEDED, message, http.StatusForbidden, nil)
}

// DBQueryFailed wraps a database-level error.
// The cause is internal — only the code and generic message reach the client.
func DBQueryFailed(cause error) *AppError {
	return newSkip(CODE_DB_QUERY_FAILED, "A database error occurred", http.StatusInternalServerError, cause)
}

// PermissionDenied is returned when RBAC evaluation denies an action.
func PermissionDenied(message string) *AppError {
	return newSkip(CODE_PERMISSION_DENIED, message, http.StatusForbidden, nil)
}
