package infrastructure

import (
	"context"
	"sync"

	"github.com/AgileFactory/kernel/platform/auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository implements domain.UserRepository using pgxpool.
type PostgresUserRepository struct {
	mu sync.RWMutex
	db *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// Save persists a new user to the database.
func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.db.Exec(ctx, queryInsertUser,
		user.ID,
		user.Email,
		user.HashedPassword,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	); err != nil {
		return err
	}
	return nil
}

// FindByEmail retrieves the user by email.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var user domain.User
	if err := r.db.QueryRow(ctx, queryGetUserByEmail, email.String()).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID retrieves the user by ID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var user domain.User
	if err := r.db.QueryRow(ctx, queryGetUserByID, id).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &user, nil
}

