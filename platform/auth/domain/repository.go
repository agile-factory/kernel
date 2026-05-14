package domain

import "context"

// UserRepository defines the persistence methods for the User entity.
type UserRepository interface {
	// Save persists a new user.
	Save(ctx context.Context, user *User) error

	// FindByEmail retrieves a user by their email address.
	FindByEmail(ctx context.Context, email Email) (*User, error)

	// FindByID retrieves a user by their unique identifier.
	FindByID(ctx context.Context, id string) (*User, error)

	// UpdateEmailVerified flips email_verified=TRUE and bumps status from
	// PENDING_EMAIL to ACTIVE. Idempotent; safe to call on already-verified users.
	UpdateEmailVerified(ctx context.Context, userID string) error
}
