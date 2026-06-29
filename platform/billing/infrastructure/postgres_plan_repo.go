// Package infrastructure provides Postgres-backed repository implementations
// for the billing platform.
package infrastructure

import (
	"context"
	stderrors "errors"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/billing/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPlanRepository implements domain.PlanRepository.
type PostgresPlanRepository struct {
	db *pgxpool.Pool
}

// NewPostgresPlanRepository creates a Postgres-backed plan repository.
func NewPostgresPlanRepository(db *pgxpool.Pool) *PostgresPlanRepository {
	return &PostgresPlanRepository{db: db}
}

const planCols = `id, name, price_cents, currency, interval,
	max_users, max_storage, max_api_calls,
	can_export, can_white_label, can_merchant, created_at`

func scanOnePlan(row pgx.Row) (*domain.Plan, error) {
	p := &domain.Plan{}
	var interval string
	err := row.Scan(
		&p.ID, &p.Name, &p.PriceCents, &p.Currency, &interval,
		&p.MaxUsers, &p.MaxStorage, &p.MaxAPICalls,
		&p.CanExport, &p.CanWhiteLabel, &p.CanMerchant, &p.CreatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, kerrors.NotFound("plan not found")
		}
		return nil, kerrors.WrapInternal(err)
	}
	p.Interval = interval
	return p, nil
}

// FindByID returns the plan with the given ID.
func (r *PostgresPlanRepository) FindByID(ctx context.Context, id string) (*domain.Plan, error) {
	row := r.db.QueryRow(ctx, "SELECT "+planCols+" FROM plans WHERE id = $1", id)
	return scanOnePlan(row)
}

// FindAll returns all plans ordered by price ascending.
func (r *PostgresPlanRepository) FindAll(ctx context.Context) ([]*domain.Plan, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+planCols+" FROM plans ORDER BY price_cents ASC, name ASC")
	if err != nil {
		return nil, kerrors.WrapInternal(err)
	}
	defer rows.Close()

	var plans []*domain.Plan
	for rows.Next() {
		p := &domain.Plan{}
		var interval string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.PriceCents, &p.Currency, &interval,
			&p.MaxUsers, &p.MaxStorage, &p.MaxAPICalls,
			&p.CanExport, &p.CanWhiteLabel, &p.CanMerchant, &p.CreatedAt,
		); err != nil {
			return nil, kerrors.WrapInternal(err)
		}
		p.Interval = interval
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		return nil, kerrors.WrapInternal(err)
	}
	return plans, nil
}
