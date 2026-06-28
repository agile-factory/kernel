package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Refresh tokens ────────────────────────────────────────────────────────────

// PostgresRefreshTokenRepository implements domain.CustomerRefreshTokenRepository.
type PostgresRefreshTokenRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRefreshTokenRepository creates a new repository backed by pgxpool.
func NewPostgresRefreshTokenRepository(db *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{db: db}
}

// Create inserts a new refresh token row.
func (r *PostgresRefreshTokenRepository) Create(ctx context.Context, t *domain.CustomerRefreshToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO customer_refresh_tokens
			(end_customer_id, tenant_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, t.EndCustomerID, t.TenantID, t.TokenHash, t.ExpiresAt)
	return err
}

// FindByHash looks up a non-revoked refresh token.
func (r *PostgresRefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.CustomerRefreshToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, end_customer_id, tenant_id, token_hash, expires_at, revoked_at, created_at
		FROM customer_refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	t := &domain.CustomerRefreshToken{}
	var revokedAt *time.Time
	err := row.Scan(&t.ID, &t.EndCustomerID, &t.TenantID, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, err
	}
	t.RevokedAt = revokedAt
	return t, nil
}

// Revoke marks a specific refresh token as revoked.
func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_refresh_tokens SET revoked_at = NOW() WHERE id = $1
	`, id)
	return err
}

// RevokeAllForEndCustomer revokes all active refresh tokens for the given end-customer.
func (r *PostgresRefreshTokenRepository) RevokeAllForEndCustomer(ctx context.Context, endCustomerID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_refresh_tokens
		SET revoked_at = NOW()
		WHERE end_customer_id = $1 AND revoked_at IS NULL
	`, endCustomerID)
	return err
}

// ── Reset tokens ──────────────────────────────────────────────────────────────

// PostgresResetTokenRepository implements domain.CustomerResetTokenRepository.
type PostgresResetTokenRepository struct {
	db *pgxpool.Pool
}

// NewPostgresResetTokenRepository creates a new repository backed by pgxpool.
func NewPostgresResetTokenRepository(db *pgxpool.Pool) *PostgresResetTokenRepository {
	return &PostgresResetTokenRepository{db: db}
}

// Create inserts a new reset token row.
func (r *PostgresResetTokenRepository) Create(ctx context.Context, t *domain.CustomerResetToken) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO customer_reset_tokens
			(end_customer_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, t.EndCustomerID, t.TokenHash, t.ExpiresAt)
	return err
}

// FindByHash looks up an unconsumed reset token.
func (r *PostgresResetTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.CustomerResetToken, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, end_customer_id, token_hash, expires_at, consumed_at, created_at
		FROM customer_reset_tokens
		WHERE token_hash = $1
	`, tokenHash)
	t := &domain.CustomerResetToken{}
	var consumedAt *time.Time
	err := row.Scan(&t.ID, &t.EndCustomerID, &t.TokenHash, &t.ExpiresAt, &consumedAt, &t.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, err
	}
	t.ConsumedAt = consumedAt
	return t, nil
}

// MarkConsumed sets consumed_at on the token.
func (r *PostgresResetTokenRepository) MarkConsumed(ctx context.Context, id string, now time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_reset_tokens SET consumed_at = $2 WHERE id = $1
	`, id, now)
	return err
}

// HasRecentToken returns true when an unconsumed reset token for the given
// end-customer was created within the since window (OTP 60s rate-limit).
func (r *PostgresResetTokenRepository) HasRecentToken(ctx context.Context, endCustomerID string, since time.Time) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM customer_reset_tokens
		WHERE end_customer_id = $1
		  AND consumed_at IS NULL
		  AND created_at >= $2
	`, endCustomerID, since).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
