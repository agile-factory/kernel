package domain

import (
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
)

var (
	// ErrInvalidEmail is returned when the email format is invalid.
	ErrInvalidEmail = errors.New(
		"INVALID_EMAIL",
		"The provided email is invalid",
		http.StatusBadRequest,
		nil,
	)

	// ErrUserNotFound is returned when a user cannot be found.
	ErrUserNotFound = errors.New(
		"USER_NOT_FOUND",
		"User not found with the provided credentials",
		http.StatusNotFound,
		nil,
	)

	// ErrUserAlreadyExists is returned when attempting to register with an existing email.
	ErrUserAlreadyExists = errors.New(
		"USER_ALREADY_EXISTS",
		"A user with this email already exists",
		http.StatusConflict,
		nil,
	)

	// ErrInvalidCredentials is returned for wrong password or email during login.
	ErrInvalidCredentials = errors.New(
		"INVALID_CREDENTIALS",
		"Invalid email or password",
		http.StatusUnauthorized,
		nil,
	)

	// ErrUserNotActive is returned when the user exists but is in a non-ACTIVE state
	// (PENDING_EMAIL, SUSPENDED, DELETED) and therefore cannot authenticate.
	ErrUserNotActive = errors.New(
		"USER_NOT_ACTIVE",
		"This account is not active",
		http.StatusForbidden,
		nil,
	)

	// ErrPasswordTooShort enforces the minimum length set by RFC-0018 §8.
	ErrPasswordTooShort = errors.New(
		"PASSWORD_TOO_SHORT",
		"Password must be at least 12 characters",
		http.StatusBadRequest,
		nil,
	)

	// ErrTokenNotFound is returned for an unknown verification / refresh token.
	ErrTokenNotFound = errors.New(
		"TOKEN_NOT_FOUND",
		"The provided token is not recognised",
		http.StatusBadRequest,
		nil,
	)

	// ErrTokenExpired is returned when a verification / refresh token is past
	// its expires_at.
	ErrTokenExpired = errors.New(
		"TOKEN_EXPIRED",
		"The provided token has expired",
		http.StatusBadRequest,
		nil,
	)

	// ErrTokenAlreadyConsumed signals a replay of a single-use token
	// (verification / invite). The legitimate user has either already used it
	// or someone else did.
	ErrTokenAlreadyConsumed = errors.New(
		"TOKEN_ALREADY_CONSUMED",
		"This token has already been used",
		http.StatusConflict,
		nil,
	)

	// ErrRefreshTokenRevoked is returned when a refresh token presented for
	// rotation has been explicitly revoked (logout or rolling rotation).
	ErrRefreshTokenRevoked = errors.New(
		"REFRESH_TOKEN_REVOKED",
		"This refresh token has been revoked",
		http.StatusUnauthorized,
		nil,
	)
)
