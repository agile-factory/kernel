package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresCredentialRepository implements domain.CredentialRepository.
type PostgresCredentialRepository struct {
	db *pgxpool.Pool
}

// NewPostgresCredentialRepository creates a new repository backed by pgxpool.
func NewPostgresCredentialRepository(db *pgxpool.Pool) *PostgresCredentialRepository {
	return &PostgresCredentialRepository{db: db}
}

// Create inserts a new credential row.
func (r *PostgresCredentialRepository) Create(ctx context.Context, cred *domain.CustomerCredential) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO customer_credentials
			(end_customer_id, tenant_id, password_hash)
		VALUES ($1, $2, $3)
	`, cred.EndCustomerID, cred.TenantID, cred.PasswordHash)
	if err != nil {
		if isCredUniqueViolation(err) {
			return domain.ErrCredentialAlreadyExists
		}
		return err
	}
	return nil
}

// FindByEndCustomerID returns the credential for the given end-customer.
func (r *PostgresCredentialRepository) FindByEndCustomerID(ctx context.Context, endCustomerID string) (*domain.CustomerCredential, error) {
	return r.scanOne(ctx, `
		SELECT id, end_customer_id, tenant_id, password_hash,
		       last_login_at, failed_attempts, locked_until, created_at, updated_at
		FROM customer_credentials
		WHERE end_customer_id = $1
	`, endCustomerID)
}

// FindByEmail looks up a credential via the end_customers.email column (JOIN).
func (r *PostgresCredentialRepository) FindByEmail(ctx context.Context, email, tenantID string) (*domain.CustomerCredential, error) {
	return r.scanOne(ctx, `
		SELECT cc.id, cc.end_customer_id, cc.tenant_id, cc.password_hash,
		       cc.last_login_at, cc.failed_attempts, cc.locked_until, cc.created_at, cc.updated_at
		FROM customer_credentials cc
		JOIN end_customers ec ON ec.id = cc.end_customer_id
		WHERE lower(ec.email) = lower($1) AND cc.tenant_id = $2
	`, email, tenantID)
}

// Update persists all mutable fields.
func (r *PostgresCredentialRepository) Update(ctx context.Context, cred *domain.CustomerCredential) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_credentials
		SET password_hash    = $2,
		    last_login_at    = $3,
		    failed_attempts  = $4,
		    locked_until     = $5,
		    updated_at       = $6
		WHERE id = $1
	`, cred.ID, cred.PasswordHash, cred.LastLoginAt, cred.FailedAttempts, cred.LockedUntil, cred.UpdatedAt)
	return err
}

func (r *PostgresCredentialRepository) scanOne(ctx context.Context, query string, args ...any) (*domain.CustomerCredential, error) {
	row := r.db.QueryRow(ctx, query, args...)
	c := &domain.CustomerCredential{}
	var lastLoginAt, lockedUntil *time.Time
	err := row.Scan(
		&c.ID, &c.EndCustomerID, &c.TenantID, &c.PasswordHash,
		&lastLoginAt, &c.FailedAttempts, &lockedUntil,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrCredentialNotFound
		}
		return nil, err
	}
	c.LastLoginAt = lastLoginAt
	c.LockedUntil = lockedUntil
	return c, nil
}

func isCredUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
