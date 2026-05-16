package infrastructure

import (
	"context"
	stderrors "errors"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/membership/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresInviteTokenRepository persists single-use, 7-day invite tokens
// (RFC-0018 §6). The raw token IS the secret; we store it directly because
// it is short-lived and bound to a specific (tenant, email, role) triple.
type PostgresInviteTokenRepository struct {
	db *pgxpool.Pool
}

func NewPostgresInviteTokenRepository(db *pgxpool.Pool) *PostgresInviteTokenRepository {
	return &PostgresInviteTokenRepository{db: db}
}

func (r *PostgresInviteTokenRepository) Save(ctx context.Context, t *domain.InviteToken) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryInsertInviteToken,
		t.ID,
		t.TenantID,
		t.InvitedBy,
		t.Email,
		string(t.Role),
		t.Token,
		t.ExpiresAt,
		t.CreatedAt,
	)
	return err
}

func (r *PostgresInviteTokenRepository) FindByToken(ctx context.Context, token string) (*domain.InviteToken, error) {
	return r.scanOne(txctx.DB(ctx, r.db).QueryRow(ctx, queryGetInviteTokenByToken, token))
}

func (r *PostgresInviteTokenRepository) FindPendingByTenantAndEmail(ctx context.Context, tenantID, email string) (*domain.InviteToken, error) {
	return r.scanOne(txctx.DB(ctx, r.db).QueryRow(ctx, queryGetPendingInviteByTenantAndEmail, tenantID, email))
}

func (r *PostgresInviteTokenRepository) Consume(ctx context.Context, id string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryConsumeInviteToken, id)
	return err
}

func (r *PostgresInviteTokenRepository) Revoke(ctx context.Context, id string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryRevokeInviteToken, id)
	return err
}

func (r *PostgresInviteTokenRepository) scanOne(row pgx.Row) (*domain.InviteToken, error) {
	var (
		t    domain.InviteToken
		role string
	)
	err := row.Scan(
		&t.ID,
		&t.TenantID,
		&t.InvitedBy,
		&t.Email,
		&role,
		&t.Token,
		&t.ExpiresAt,
		&t.ConsumedAt,
		&t.RevokedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInviteTokenNotFound
		}
		return nil, err
	}
	t.Role = domain.MemberRole(role)
	return &t, nil
}
