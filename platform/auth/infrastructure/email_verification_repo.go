package infrastructure

import (
	"context"
	stderrors "errors"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEmailVerificationTokenRepository persists single-use email
// verification tokens (RFC-0018 §4). Only the SHA-256 of the raw token is
// stored (migration 0006), like refresh tokens: a database read must not be
// enough to verify someone else's email.
type PostgresEmailVerificationTokenRepository struct {
	db *pgxpool.Pool
}

func NewPostgresEmailVerificationTokenRepository(db *pgxpool.Pool) *PostgresEmailVerificationTokenRepository {
	return &PostgresEmailVerificationTokenRepository{db: db}
}

func (r *PostgresEmailVerificationTokenRepository) Save(ctx context.Context, t *domain.EmailVerificationToken) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryInsertEmailVerificationToken,
		t.ID,
		t.UserID,
		domain.HashToken(t.Token),
		t.ExpiresAt,
		t.CreatedAt,
	)
	return err
}

// FindByToken looks the token up by its hash. The returned Token field holds
// the stored hash, not the raw token.
func (r *PostgresEmailVerificationTokenRepository) FindByToken(ctx context.Context, token string) (*domain.EmailVerificationToken, error) {
	var t domain.EmailVerificationToken
	err := txctx.DB(ctx, r.db).QueryRow(ctx, queryGetEmailVerificationTokenByToken, domain.HashToken(token)).Scan(
		&t.ID,
		&t.UserID,
		&t.Token,
		&t.ExpiresAt,
		&t.ConsumedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTokenNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *PostgresEmailVerificationTokenRepository) Consume(ctx context.Context, id string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryConsumeEmailVerificationToken, id)
	return err
}
