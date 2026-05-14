package domain

import (
	"net/http"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
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
)
