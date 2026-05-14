package infrastructure

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTenantRepository is the Postgres implementation of domain.TenantRepository.
type PostgresTenantRepository struct {
	mu sync.RWMutex
	db *pgxpool.Pool
}

// NewPostgresTenantRepository creates a new PostgresTenantRepository.
func NewPostgresTenantRepository(db *pgxpool.Pool) *PostgresTenantRepository {
	return &PostgresTenantRepository{db: db}
}

// Save persists a new tenant to the database.
func (r *PostgresTenantRepository) Save(ctx context.Context, t *domain.Tenant) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(ctx, queryInsertTenant,
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
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.scanTenant(r.db.QueryRow(ctx, queryGetTenantByID, id))
}

// FindBySlug retrieves a tenant by its unique slug.
func (r *PostgresTenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.scanTenant(r.db.QueryRow(ctx, queryGetTenantBySlug, slug))
}

// ExistsBySlug returns true when a tenant with the given slug already exists.
func (r *PostgresTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var exists bool
	if err := r.db.QueryRow(ctx, queryTenantExistsBySlug, slug).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
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
