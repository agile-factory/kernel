package infrastructure

import (
	"context"
	stderrors "errors"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRefreshTokenRepository persists refresh tokens (RFC-0018 §5). Only
// the SHA-256 hash of the opaque token lives in the table; the raw token is
// returned to the caller exactly once at issuance and discarded server-side.
type PostgresRefreshTokenRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRefreshTokenRepository(db *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{db: db}
}

func (r *PostgresRefreshTokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
	var tenantID any
	if t.TenantID != "" {
		tenantID = t.TenantID
	}
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryInsertRefreshToken,
		t.ID,
		t.UserID,
		tenantID,
		t.TokenHash,
		t.ExpiresAt,
		t.CreatedAt,
	)
	return err
}

func (r *PostgresRefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	var (
		t        domain.RefreshToken
		tenantID *string
	)
	err := txctx.DB(ctx, r.db).QueryRow(ctx, queryGetRefreshTokenByHash, hash).Scan(
		&t.ID,
		&t.UserID,
		&tenantID,
		&t.TokenHash,
		&t.ExpiresAt,
		&t.RevokedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, err
	}
	if tenantID != nil {
		t.TenantID = *tenantID
	}
	return &t, nil
}

func (r *PostgresRefreshTokenRepository) Revoke(ctx context.Context, id string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryRevokeRefreshToken, id)
	return err
}
