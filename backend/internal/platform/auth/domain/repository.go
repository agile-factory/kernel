package domain

import "context"

// UserRepository defines the persistence methods for the User entity.
type UserRepository interface {
	// Save persists a new user to the database.
	Save(ctx context.Context, user *User) error

	// FindByEmail retrieves a user by their email address.
	FindByEmail(ctx context.Context, email Email) (*User, error)

	// FindByID retrieves a user by their unique identifier.
	FindByID(ctx context.Context, id string) (*User, error)
}
