package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string
type UserStatus string

const (
	RoleTenant   UserRole = "tenant"
	RoleLandlord UserRole = "landlord"
	RoleAdmin    UserRole = "admin"

	StatusPendingVerification UserStatus = "pending_verification"
	StatusActive              UserStatus = "active"
	StatusSuspended           UserStatus = "suspended"
	StatusDeleted             UserStatus = "deleted"
)

type User struct {
	ID                   uuid.UUID  `json:"id" db:"id"`
	Email                string     `json:"email" db:"email"`
	PasswordHash         string     `json:"-" db:"password_hash"`
	FirstName            string     `json:"first_name" db:"first_name"`
	LastName             string     `json:"last_name" db:"last_name"`
	Phone                *string    `json:"phone,omitempty" db:"phone"`
	Role                 UserRole   `json:"role" db:"role"`
	Status               UserStatus `json:"status" db:"status"`
	EmailVerified        bool       `json:"email_verified" db:"email_verified"`
	PhoneVerified        bool       `json:"phone_verified" db:"phone_verified"`
	VerificationToken    *string    `json:"-" db:"verification_token"`
	ResetPasswordToken   *string    `json:"-" db:"reset_password_token"`
	ResetPasswordExpires *time.Time `json:"-" db:"reset_password_expires"`
	LastLogin            *time.Time `json:"last_login,omitempty" db:"last_login"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt            *time.Time `json:"-" db:"deleted_at"`
}

type RefreshToken struct {
	ID         uuid.UUID  `json:"id" db:"id"`
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash  string     `json:"-" db:"token_hash"`
	DeviceInfo *string    `json:"device_info,omitempty" db:"device_info"`
	IPAddress  *string    `json:"ip_address,omitempty" db:"ip_address"`
	ExpiresAt  time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
}

// DTOs for API requests/responses
type RegisterRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Password  string   `json:"password" validate:"required,min=8"`
	FirstName string   `json:"first_name" validate:"required"`
	LastName  string   `json:"last_name" validate:"required"`
	Phone     *string  `json:"phone,omitempty"`
	Role      UserRole `json:"role" validate:"required,oneof=tenant landlord"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}
