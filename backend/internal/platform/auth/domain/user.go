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

// User represents the central authentication entity.
type User struct {
	ID             string
	TenantID       string
	Email          Email
	HashedPassword string
	Role           Role
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewUser creates a new User instance.
func NewUser(id, tenantId string, email Email, hashedPassword string, role Role) *User {
	now := time.Now()
	return &User{
		ID:             id,
		TenantID:       tenantId,
		Email:          email,
		HashedPassword: hashedPassword,
		Role:           role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
