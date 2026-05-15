package infrastructure

import (
	"context"
	"errors"
	"time"

	"github.com/AgileFactory/kernel/kernel/txctx"
	"github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTenantRepository is the Postgres implementation of
// domain.TenantRepository. All operations honour an in-context pgx.Tx via
// txctx.DB so the repo can be enrolled in an outer transaction.
type PostgresTenantRepository struct {
	db *pgxpool.Pool
}

// NewPostgresTenantRepository creates a new PostgresTenantRepository.
func NewPostgresTenantRepository(db *pgxpool.Pool) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

// Save persists a new tenant.
func (r *PostgresTenantRepository) Save(ctx context.Context, t *domain.Tenant) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryInsertTenant,
		t.ID,
		t.Name,
		t.Slug,
		t.IsActive,
		t.CreatedAt,
		t.UpdatedAt,
	)
	return err
}

// FindByID retrieves a tenant by its UUID.
func (r *PostgresTenantRepository) FindByID(ctx context.Context, id string) (*domain.Tenant, error) {
	return r.scanTenant(txctx.DB(ctx, r.db).QueryRow(ctx, queryGetTenantByID, id))
}

// FindBySlug retrieves a tenant by its unique slug.
func (r *PostgresTenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	return r.scanTenant(txctx.DB(ctx, r.db).QueryRow(ctx, queryGetTenantBySlug, slug))
}

// ExistsBySlug returns true when a tenant with the given slug already exists.
func (r *PostgresTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var exists bool
	if err := txctx.DB(ctx, r.db).QueryRow(ctx, queryTenantExistsBySlug, slug).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

// UpdateName changes the tenant's display name.
func (r *PostgresTenantRepository) UpdateName(ctx context.Context, id, name string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryUpdateTenantName, id, name)
	return err
}

// SoftDelete flips is_active to FALSE. Tenant row is preserved so that
// historical references (audit logs, memberships) stay valid.
func (r *PostgresTenantRepository) SoftDelete(ctx context.Context, id string) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, querySoftDeleteTenant, id)
	return err
}

// scanTenant maps a pgx row into a Tenant value object.
func (r *PostgresTenantRepository) scanTenant(row pgx.Row) (*domain.Tenant, error) {
	var t domain.Tenant
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&t.ID,
		&t.Name,
		&t.Slug,
		&t.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, err
	}

	t.CreatedAt = createdAt
	t.UpdatedAt = updatedAt
	return &t, nil
}
