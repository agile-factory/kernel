package infrastructure

import (
	"context"
	"errors"

	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEndCustomerResolver looks up end-customer contact info via SQL.
// Cross-module join: customer_notifications → end_customers.
type PostgresEndCustomerResolver struct {
	db *pgxpool.Pool
}

// NewPostgresEndCustomerResolver creates the resolver.
func NewPostgresEndCustomerResolver(db *pgxpool.Pool) *PostgresEndCustomerResolver {
	return &PostgresEndCustomerResolver{db: db}
}

// GetContact returns the end-customer's email, phone, and name.
func (r *PostgresEndCustomerResolver) GetContact(ctx context.Context, endCustomerID, tenantID string) (*domain.EndCustomerContact, error) {
	row := r.db.QueryRow(ctx, `
		SELECT email, coalesce(phone, ''), coalesce(first_name, ''), coalesce(last_name, '')
		FROM end_customers
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
	`, endCustomerID, tenantID)
	c := &domain.EndCustomerContact{}
	if err := row.Scan(&c.Email, &c.Phone, &c.FirstName, &c.LastName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEndCustomerNotFound
		}
		return nil, err
	}
	return c, nil
}

// PostgresBrandResolver looks up tenant brand data via SQL.
// Cross-module join: customer_notifications → brand_settings.
type PostgresBrandResolver struct {
	db *pgxpool.Pool
}

// NewPostgresBrandResolver creates the resolver.
func NewPostgresBrandResolver(db *pgxpool.Pool) *PostgresBrandResolver {
	return &PostgresBrandResolver{db: db}
}

// GetBrand returns the tenant's branding fields needed for templates.
func (r *PostgresBrandResolver) GetBrand(ctx context.Context, tenantID string) (*domain.BrandData, error) {
	row := r.db.QueryRow(ctx, `
		SELECT coalesce(app_name, ''), coalesce(logo_url, ''),
		       coalesce(primary_color, '#000000'), coalesce(support_email, '')
		FROM brand_settings
		WHERE tenant_id = $1
	`, tenantID)
	b := &domain.BrandData{}
	if err := row.Scan(&b.AppName, &b.LogoURL, &b.PrimaryColor, &b.SupportEmail); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Brand not configured — use empty defaults.
			return &domain.BrandData{}, nil
		}
		return nil, err
	}
	return b, nil
}
