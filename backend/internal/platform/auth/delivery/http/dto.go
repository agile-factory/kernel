package http

import (
	"errors"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/validator"
)

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID string `json:"tenantId"`
}

// Validate checks the basic rules for the RegisterRequest.
func (r *RegisterRequest) Validate() error {
	if !validator.IsValidEmail(r.Email) {
		return errors.New("invalid email format")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if !validator.IsNotEmpty(r.TenantID) {
		return errors.New("tenantId is required")
	}
	return nil
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks the basic rules for the LoginRequest.
func (r *LoginRequest) Validate() error {
	if !validator.IsValidEmail(r.Email) {
		return errors.New("invalid email format")
	}
	if len(r.Password) == 0 {
		return errors.New("password is required")
	}
	return nil
}

// AuthResponse is the payload returned upon successful auth.
type AuthResponse struct {
	Token string `json:"token"`
	User  UserDTO `json:"user"`
}

// UserDTO represents public user data.
type UserDTO struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	TenantID string `json:"tenantId"`
	Token string `json:"token"`	
}
