package infrastructure

import (
	"context"
	stderrors "errors"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserRepository implements domain.UserRepository using pgxpool. All
// write paths honour an in-context pgx.Tx via txctx.DB, so the repo enlists
// into an outer transaction (e.g. SignupOwner's atomic tenant+user+membership
// insert) when one is present, and falls back to the pool otherwise.
type PostgresUserRepository struct {
	db *pgxpool.Pool
}

// NewPostgresUserRepository creates a new PostgresUserRepository.
func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Save persists a new user.
func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryInsertUser,
		user.ID,
		user.Email,
		user.HashedPassword,
		user.Role,
		user.EmailVerified,
		string(user.Status),
		user.CreatedAt,
		user.UpdatedAt,
	)
	return err
}

// FindByEmail retrieves the user by email.
func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email domain.Email) (*domain.User, error) {
	var (
		user   domain.User
		status string
	)
	err := txctx.DB(ctx, r.db).QueryRow(ctx, queryGetUserByEmail, email.String()).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.Role,
		&user.EmailVerified,
		&status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	user.Status = domain.UserStatus(status)
	return &user, nil
}

// FindByID retrieves the user by ID.
func (r *PostgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var (
		user   domain.User
		status string
	)
	err := txctx.DB(ctx, r.db).QueryRow(ctx, queryGetUserByID, id).Scan(
		&user.ID,
		&user.Email,
		&user.HashedPassword,
		&user.Role,
		&user.EmailVerified,
		&status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	user.Status = domain.UserStatus(status)
	return &user, nil
}

// UpdateEmailVerified flips email_verified=TRUE and (when applicable) bumps
// status from PENDING_EMAIL to ACTIVE.
func (r *PostgresUserRepository) UpdateEmailVerified(ctx context.Context, userID string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryMarkUserEmailVerified, userID)
	return err
}
