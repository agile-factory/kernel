package infrastructure

import (
	"context"
	"errors"
	"sync"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
)

// PostgresUserRepository is a mock implementation of domain.UserRepository
// Wait until the database driver (pgx/sqlx) is configured to implement the real one.
type PostgresUserRepository struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

// NewPostgresUserRepository creates a new mocked PostgresUserRepository.
func NewPostgresUserRepository() *PostgresUserRepository {
	return &PostgresUserRepository{
		users: make(map[string]*domain.User),
	}
}

// Save stores the user in the mock storage.
func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.Email.String()] = user
	return nil
}

// FindByEmail retrieves the user by email from the mock storage.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[email.String()]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

// FindByID retrieves the user by ID from the mock storage.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			return user, nil
		}
	}
	return nil, errors.New("user not found by id")
}
