package infrastructure

import (
	"context"
	stderrors "errors"

	"github.com/AgileFactory/kernel/kernel/txctx"
	"github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresTenantSettingsRepository implements domain.TenantSettingsRepository.
type PostgresTenantSettingsRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTenantSettingsRepository(db *pgxpool.Pool) *PostgresTenantSettingsRepository {
	return &PostgresTenantSettingsRepository{db: db}
}

func (r *PostgresTenantSettingsRepository) Save(ctx context.Context, s *domain.TenantSettings) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryInsertTenantSettings,
		s.TenantID,
		nullableString(s.LogoURL),
		nullableString(s.PrimaryColor),
		s.Locale,
		s.Timezone,
		s.Currency,
		string(s.DataRegion),
		nullableString(s.CustomDomain),
		string(s.PaymentRegion),
		s.MerchantMode,
		s.B2B2CEnabled,
		s.CreatedAt,
		s.UpdatedAt,
	)
	return err
}

func (r *PostgresTenantSettingsRepository) Get(ctx context.Context, tenantID string) (*domain.TenantSettings, error) {
	var (
		s             domain.TenantSettings
		logoURL       *string
		primaryColor  *string
		dataRegion    string
		customDomain  *string
		paymentRegion string
	)
	err := txctx.DB(ctx, r.db).QueryRow(ctx, queryGetTenantSettings, tenantID).Scan(
		&s.TenantID,
		&logoURL,
		&primaryColor,
		&s.Locale,
		&s.Timezone,
		&s.Currency,
		&dataRegion,
		&customDomain,
		&paymentRegion,
		&s.MerchantMode,
		&s.B2B2CEnabled,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if stderrors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantSettingsNotFound
		}
		return nil, err
	}
	if logoURL != nil {
		s.LogoURL = *logoURL
	}
	if primaryColor != nil {
		s.PrimaryColor = *primaryColor
	}
	if customDomain != nil {
		s.CustomDomain = *customDomain
	}
	s.DataRegion = domain.DataRegion(dataRegion)
	s.PaymentRegion = domain.PaymentRegion(paymentRegion)
	return &s, nil
}

func (r *PostgresTenantSettingsRepository) Update(ctx context.Context, s *domain.TenantSettings) error {
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryUpdateTenantSettings,
		s.TenantID,
		nullableString(s.LogoURL),
		nullableString(s.PrimaryColor),
		s.Locale,
		s.Timezone,
		s.Currency,
		string(s.DataRegion),
		nullableString(s.CustomDomain),
		string(s.PaymentRegion),
		s.MerchantMode,
		s.B2B2CEnabled,
	)
	return err
}

// nullableString returns nil for empty strings so they land as SQL NULL.
// Avoids storing empty strings for nullable columns (logo_url, custom_domain,
// primary_color) which would conflict with the UNIQUE partial index on
// custom_domain WHERE custom_domain IS NOT NULL.
func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
