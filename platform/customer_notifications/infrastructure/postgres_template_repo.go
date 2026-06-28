package infrastructure

import (
	"context"
	"errors"

	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresCustomerTemplateRepository implements domain.CustomerTemplateRepository.
type PostgresCustomerTemplateRepository struct {
	db *pgxpool.Pool
}

// NewPostgresCustomerTemplateRepository creates a new repository backed by pgxpool.
func NewPostgresCustomerTemplateRepository(db *pgxpool.Pool) *PostgresCustomerTemplateRepository {
	return &PostgresCustomerTemplateRepository{db: db}
}

// Upsert inserts or replaces a template override.
func (r *PostgresCustomerTemplateRepository) Upsert(ctx context.Context, t *domain.CustomerTemplate) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO customer_templates
			(tenant_id, key, channel, locale, subject_tmpl, body_tmpl)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, key, channel, locale) DO UPDATE SET
			subject_tmpl = EXCLUDED.subject_tmpl,
			body_tmpl    = EXCLUDED.body_tmpl,
			updated_at   = NOW()
	`, t.TenantID, t.Key, string(t.Channel), t.Locale,
		nullableStr(t.SubjectTmpl), t.BodyTmpl)
	return err
}

// Find returns a tenant's template override for the given key/channel/locale.
func (r *PostgresCustomerTemplateRepository) Find(ctx context.Context, tenantID, key string, ch domain.Channel, locale string) (*domain.CustomerTemplate, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, key, channel, locale,
		       coalesce(subject_tmpl, ''), body_tmpl,
		       created_at, updated_at
		FROM customer_templates
		WHERE tenant_id = $1 AND key = $2 AND channel = $3 AND locale = $4
	`, tenantID, key, string(ch), locale)
	t := &domain.CustomerTemplate{}
	var chStr string
	err := row.Scan(
		&t.ID, &t.TenantID, &t.Key, &chStr, &t.Locale,
		&t.SubjectTmpl, &t.BodyTmpl,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTemplateNotFound
		}
		return nil, err
	}
	t.Channel = domain.Channel(chStr)
	return t, nil
}

// Delete removes a template override.
func (r *PostgresCustomerTemplateRepository) Delete(ctx context.Context, tenantID, key string, ch domain.Channel, locale string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM customer_templates
		WHERE tenant_id = $1 AND key = $2 AND channel = $3 AND locale = $4
	`, tenantID, key, string(ch), locale)
	return err
}

// ListByTenant returns all template overrides for the tenant.
func (r *PostgresCustomerTemplateRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.CustomerTemplate, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, tenant_id, key, channel, locale,
		       coalesce(subject_tmpl, ''), body_tmpl,
		       created_at, updated_at
		FROM customer_templates
		WHERE tenant_id = $1
		ORDER BY key, channel, locale
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []*domain.CustomerTemplate
	for rows.Next() {
		t := &domain.CustomerTemplate{}
		var chStr string
		if err := rows.Scan(
			&t.ID, &t.TenantID, &t.Key, &chStr, &t.Locale,
			&t.SubjectTmpl, &t.BodyTmpl,
			&t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, err
		}
		t.Channel = domain.Channel(chStr)
		templates = append(templates, t)
	}
	return templates, rows.Err()
}
