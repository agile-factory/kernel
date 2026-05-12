package infrastructure

import (
	"context"
	"sync"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository is a mock implementation of domain.UserRepository
// Wait until the database driver (pgx/sqlx) is configured to implement the real one.
type PostgresUserRepository struct {
	mu    sync.RWMutex
	db *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// Save stores the user in the mock storage.
func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.db.Exec(ctx, queryInsertUser, user.ID, user.TenantID, user.Email, user.HashedPassword, user.Role, user.CreatedAt, user.UpdatedAt); err != nil {
		return err
	}
	return nil
}

// FindByEmail retrieves the user by email from the mock storage.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var user domain.User
	if err := r.db.QueryRow(ctx, queryGetUserByEmail, email.String()).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID retrieves the user by ID from the mock storage.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var user domain.User
	if err := r.db.QueryRow(ctx, queryGetUserByID, id).Scan(&user.ID, &user.Email, &user.HashedPassword, &user.Role, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return nil, err
	}
	return &user, nil
}
