package domain

import (
	"time"
)

// Role defines the authorization level of a user.
type Role string

const (
	ROLE_ADMIN  Role = "ROLE_ADMIN"
	ROLE_AGENT  Role = "ROLE_AGENT"
	ROLE_VIEWER Role = "ROLE_VIEWER"
)

// UserStatus is the lifecycle state introduced by RFC-0018: new signups land
// in PENDING_EMAIL and transition to ACTIVE upon email verification.
type UserStatus string

const (
	USER_STATUS_PENDING_EMAIL UserStatus = "PENDING_EMAIL"
	USER_STATUS_ACTIVE        UserStatus = "ACTIVE"
	USER_STATUS_SUSPENDED     UserStatus = "SUSPENDED"
	USER_STATUS_DELETED       UserStatus = "DELETED"
)

// User represents the central authentication entity. Tenant membership is
// tracked separately in the memberships table.
type User struct {
	ID             string
	Email          Email
	HashedPassword string
	Role           Role
	EmailVerified  bool
	Status         UserStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewUser creates a new User in PENDING_EMAIL until the verification token is
// redeemed. Use NewVerifiedUser for the accept-invite path where the invite
// token has already proven the user controls the email.
func NewUser(id string, email Email, hashedPassword string, role Role) *User {
	now := time.Now()
	return &User{
		ID:             id,
		Email:          email,
		HashedPassword: hashedPassword,
		Role:           role,
		EmailVerified:  false,
		Status:         USER_STATUS_PENDING_EMAIL,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewVerifiedUser creates a user that bypasses email verification — used by
// AcceptInvite, where the invite token already proved address ownership.
func NewVerifiedUser(id string, email Email, hashedPassword string, role Role) *User {
	u := NewUser(id, email, hashedPassword, role)
	u.EmailVerified = true
	u.Status = USER_STATUS_ACTIVE
	return u
}

// MarkEmailVerified flips the user to ACTIVE; idempotent.
func (u *User) MarkEmailVerified() {
	u.EmailVerified = true
	if u.Status == USER_STATUS_PENDING_EMAIL {
		u.Status = USER_STATUS_ACTIVE
	}
	u.UpdatedAt = time.Now()
}

// IsActive returns true when the user can authenticate.
func (u *User) IsActive() bool {
	return u.Status == USER_STATUS_ACTIVE
}
