package http

import (
	"errors"
)

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID string `json:"tenantId"` // The tenant to join on registration
	Role     string `json:"role"`     // Optional: ROLE_OWNER|ROLE_ADMIN|ROLE_AGENT|ROLE_VIEWER (default: ROLE_VIEWER)
}

// Validate checks the basic rules for the RegisterRequest.
func (r *RegisterRequest) Validate() error {
	if len(r.Email) == 0 {
		return errors.New("email is required")
	}
	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if len(r.TenantID) == 0 {
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
	if len(r.Email) == 0 {
		return errors.New("email is required")
	}
	if len(r.Password) == 0 {
		return errors.New("password is required")
	}
	return nil
}

// AuthResponse is the payload returned upon successful auth.
type AuthResponse struct {
	Token string  `json:"token"`
	User  UserDTO `json:"user"`
}

// MembershipSummary is a compact view of a membership embedded in auth responses.
type MembershipSummary struct {
	MembershipID string `json:"membershipId"`
	TenantID     string `json:"tenantId"`
	Role         string `json:"role"`
	Status       string `json:"status"`
}

// UserDTO represents public user data returned on register/login.
type UserDTO struct {
	ID          string              `json:"id"`
	Email       string              `json:"email"`
	Role        string              `json:"role"`
	Memberships []MembershipSummary `json:"memberships"`
	Token       string              `json:"token,omitempty"`
}

