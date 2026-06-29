package domain

import (
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
)

var (
	// ErrInvalidCredentials is returned for any auth failure (email not found,
	// wrong password). Never distinguish the two — prevents email enumeration.
	ErrInvalidCredentials = &errors.AppError{
		Code:     "INVALID_CREDENTIALS",
		Message:  "invalid email or password",
		HTTPCode: http.StatusUnauthorized,
	}

	// ErrAccountLocked is returned when failed_attempts >= 5.
	ErrAccountLocked = &errors.AppError{
		Code:     "ACCOUNT_LOCKED",
		Message:  "account is temporarily locked — retry in 15 minutes",
		HTTPCode: http.StatusTooManyRequests,
	}

	// ErrCredentialNotFound is returned when no credential row exists for the end-customer.
	// Used internally — callers should surface ErrInvalidCredentials to clients.
	ErrCredentialNotFound = &errors.AppError{
		Code:     "CREDENTIAL_NOT_FOUND",
		Message:  "no credential found for this customer",
		HTTPCode: http.StatusNotFound,
	}

	// ErrCredentialAlreadyExists is returned when attempting to register twice.
	ErrCredentialAlreadyExists = &errors.AppError{
		Code:     "CREDENTIAL_ALREADY_EXISTS",
		Message:  "this customer already has a password set",
		HTTPCode: http.StatusConflict,
	}

	// ErrTokenNotFound is returned for an unknown refresh or reset token.
	ErrTokenNotFound = &errors.AppError{
		Code:     "TOKEN_NOT_FOUND",
		Message:  "token not found",
		HTTPCode: http.StatusUnauthorized,
	}

	// ErrTokenExpired is returned when a token is past its expires_at.
	ErrTokenExpired = &errors.AppError{
		Code:     "TOKEN_EXPIRED",
		Message:  "token has expired",
		HTTPCode: http.StatusUnauthorized,
	}

	// ErrTokenConsumed is returned when a single-use reset token is replayed.
	ErrTokenConsumed = &errors.AppError{
		Code:     "TOKEN_CONSUMED",
		Message:  "this token has already been used",
		HTTPCode: http.StatusConflict,
	}

	// ErrOTPRateLimit is returned when a second SMS OTP is requested within 60 seconds.
	ErrOTPRateLimit = &errors.AppError{
		Code:     "OTP_RATE_LIMIT",
		Message:  "please wait 60 seconds before requesting a new code",
		HTTPCode: http.StatusTooManyRequests,
	}

	// ErrPasswordTooShort enforces the 12-character minimum (RFC-0018 §8).
	ErrPasswordTooShort = &errors.AppError{
		Code:     "PASSWORD_TOO_SHORT",
		Message:  "password must be at least 12 characters",
		HTTPCode: http.StatusBadRequest,
	}

	// ErrB2B2CDisabled is returned when tenant_settings.b2b2c_enabled == false.
	ErrB2B2CDisabled = &errors.AppError{
		Code:     "B2B2C_DISABLED",
		Message:  "customer portal is not enabled for this tenant",
		HTTPCode: http.StatusForbidden,
	}
)
