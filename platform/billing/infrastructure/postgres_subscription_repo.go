package infrastructure

import (
	"context"
	stderrors "errors"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/billing/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSubscriptionRepository implements domain.SubscriptionRepository.
type PostgresSubscriptionRepository struct {
	db *pgxpool.Pool
}

// NewPostgresSubscriptionRepository creates a Postgres-backed subscription repository.
func NewPostgresSubscriptionRepository(db *pgxpool.Pool) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{db: db}
}

const subColumns = `
	id, tenant_id, plan_id, status,
	current_period_start, current_period_end, canceled_at,
	payment_method, coalesce(external_id, ''), coalesce(mobile_money_ref, ''), created_at`

// FindByTenantID returns the subscription for the given tenant.
// Returns kerrors.NotFound when no subscription exists yet.
func (r *PostgresSubscriptionRepository) FindByTenantID(ctx context.Context, tenantID string) (*domain.Subscription, error) {
	row := r.db.QueryRow(ctx,
		"SELECT "+subColumns+" FROM subscriptions WHERE tenant_id = $1", tenantID)
	s := &domain.Subscription{}
	var status string
	err := row.Scan(
		&s.ID, &s.TenantID, &s.PlanID, &status,
		&s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CanceledAt,
		&s.PaymentMethod, &s.ExternalID, &s.MobileMoneyRef, &s.CreatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, kerrors.NotFound("subscription not found")
		}
		return nil, kerrors.WrapInternal(err)
	}
	s.Status = domain.SubscriptionStatus(status)
	return s, nil
}

// Upsert inserts or updates the subscription for the given tenant.
// Keyed on tenant_id (UNIQUE constraint in DB).
func (r *PostgresSubscriptionRepository) Upsert(ctx context.Context, sub *domain.Subscription) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO subscriptions
			(tenant_id, plan_id, status, current_period_start, current_period_end,
			 canceled_at, payment_method, external_id, mobile_money_ref)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_id) DO UPDATE SET
			plan_id              = EXCLUDED.plan_id,
			status               = EXCLUDED.status,
			current_period_start = EXCLUDED.current_period_start,
			current_period_end   = EXCLUDED.current_period_end,
			canceled_at          = EXCLUDED.canceled_at,
			payment_method       = EXCLUDED.payment_method,
			external_id          = EXCLUDED.external_id,
			mobile_money_ref     = EXCLUDED.mobile_money_ref
	`,
		sub.TenantID, sub.PlanID, string(sub.Status),
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CanceledAt,
		sub.PaymentMethod, nullableString(sub.ExternalID), nullableString(sub.MobileMoneyRef),
	)
	if err != nil {
		return kerrors.WrapInternal(err)
	}
	return nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
