// Package infrastructure provides the Postgres implementation of domain.BrandRepository.
package infrastructure

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/platform/white_label/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresBrandRepository is the Postgres-backed implementation of domain.BrandRepository.
type PostgresBrandRepository struct {
	db *pgxpool.Pool
}

// NewPostgresBrandRepository builds the repository.
func NewPostgresBrandRepository(db *pgxpool.Pool) *PostgresBrandRepository {
	return &PostgresBrandRepository{db: db}
}

// Upsert inserts or replaces the brand settings for the tenant.
func (r *PostgresBrandRepository) Upsert(ctx context.Context, b *domain.BrandSettings) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO brand_settings (
			tenant_id, app_name, logo_url, favicon_url,
			primary_color, secondary_color, accent_color,
			font_family, custom_font_url,
			custom_domain, custom_domain_status, custom_domain_token, custom_domain_verified_at,
			email_footer_html, email_logo_url,
			terms_url, privacy_url, support_email, support_phone,
			updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
		ON CONFLICT (tenant_id) DO UPDATE SET
			app_name               = EXCLUDED.app_name,
			logo_url               = EXCLUDED.logo_url,
			favicon_url            = EXCLUDED.favicon_url,
			primary_color          = EXCLUDED.primary_color,
			secondary_color        = EXCLUDED.secondary_color,
			accent_color           = EXCLUDED.accent_color,
			font_family            = EXCLUDED.font_family,
			custom_font_url        = EXCLUDED.custom_font_url,
			custom_domain          = EXCLUDED.custom_domain,
			custom_domain_status   = EXCLUDED.custom_domain_status,
			custom_domain_token    = EXCLUDED.custom_domain_token,
			custom_domain_verified_at = EXCLUDED.custom_domain_verified_at,
			email_footer_html      = EXCLUDED.email_footer_html,
			email_logo_url         = EXCLUDED.email_logo_url,
			terms_url              = EXCLUDED.terms_url,
			privacy_url            = EXCLUDED.privacy_url,
			support_email          = EXCLUDED.support_email,
			support_phone          = EXCLUDED.support_phone,
			updated_at             = EXCLUDED.updated_at
	`,
		b.TenantID, b.AppName, nullableStr(b.LogoURL), nullableStr(b.FaviconURL),
		b.PrimaryColor, b.SecondaryColor, b.AccentColor,
		b.FontFamily, nullableStr(b.CustomFontURL),
		nullableStr(b.CustomDomain), string(b.CustomDomainStatus), nullableStr(b.CustomDomainToken), b.CustomDomainVerifiedAt,
		nullableStr(b.EmailFooterHTML), nullableStr(b.EmailLogoURL),
		nullableStr(b.TermsURL), nullableStr(b.PrivacyURL), nullableStr(b.SupportEmail), nullableStr(b.SupportPhone),
		b.UpdatedAt,
	)
	return err
}

// FindByTenantID returns the brand settings for the given tenant.
func (r *PostgresBrandRepository) FindByTenantID(ctx context.Context, tenantID string) (*domain.BrandSettings, error) {
	return r.scanOne(ctx, `
		SELECT tenant_id, app_name, logo_url, favicon_url,
		       primary_color, secondary_color, accent_color,
		       font_family, custom_font_url,
		       custom_domain, custom_domain_status, custom_domain_token, custom_domain_verified_at,
		       email_footer_html, email_logo_url,
		       terms_url, privacy_url, support_email, support_phone,
		       updated_at
		FROM brand_settings
		WHERE tenant_id = $1
	`, tenantID)
}

// FindByCustomDomain returns the brand settings for the given verified domain.
func (r *PostgresBrandRepository) FindByCustomDomain(ctx context.Context, d string) (*domain.BrandSettings, error) {
	return r.scanOne(ctx, `
		SELECT tenant_id, app_name, logo_url, favicon_url,
		       primary_color, secondary_color, accent_color,
		       font_family, custom_font_url,
		       custom_domain, custom_domain_status, custom_domain_token, custom_domain_verified_at,
		       email_footer_html, email_logo_url,
		       terms_url, privacy_url, support_email, support_phone,
		       updated_at
		FROM brand_settings
		WHERE custom_domain = $1 AND custom_domain_status = 'VERIFIED'
	`, d)
}

// UpdateDomainStatus persists only the status and verified_at columns.
func (r *PostgresBrandRepository) UpdateDomainStatus(ctx context.Context, tenantID string, status domain.DomainVerificationStatus, verifiedAt *time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE brand_settings
		SET custom_domain_status = $2, custom_domain_verified_at = $3, updated_at = NOW()
		WHERE tenant_id = $1
	`, tenantID, string(status), verifiedAt)
	return err
}

func (r *PostgresBrandRepository) scanOne(ctx context.Context, query string, args ...any) (*domain.BrandSettings, error) {
	row := r.db.QueryRow(ctx, query, args...)
	b := &domain.BrandSettings{}
	var (
		logoURL, faviconURL, customFontURL     *string
		customDomain, customDomainToken        *string
		emailFooterHTML, emailLogoURL          *string
		termsURL, privacyURL, supportEmail, supportPhone *string
		statusStr                              string
	)
	err := row.Scan(
		&b.TenantID, &b.AppName, &logoURL, &faviconURL,
		&b.PrimaryColor, &b.SecondaryColor, &b.AccentColor,
		&b.FontFamily, &customFontURL,
		&customDomain, &statusStr, &customDomainToken, &b.CustomDomainVerifiedAt,
		&emailFooterHTML, &emailLogoURL,
		&termsURL, &privacyURL, &supportEmail, &supportPhone,
		&b.UpdatedAt,
	)
	if err != nil {
		return nil, domain.ErrBrandNotFound
	}
	derefStr := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	b.LogoURL = derefStr(logoURL)
	b.FaviconURL = derefStr(faviconURL)
	b.CustomFontURL = derefStr(customFontURL)
	b.CustomDomain = derefStr(customDomain)
	b.CustomDomainToken = derefStr(customDomainToken)
	b.EmailFooterHTML = derefStr(emailFooterHTML)
	b.EmailLogoURL = derefStr(emailLogoURL)
	b.TermsURL = derefStr(termsURL)
	b.PrivacyURL = derefStr(privacyURL)
	b.SupportEmail = derefStr(supportEmail)
	b.SupportPhone = derefStr(supportPhone)
	b.CustomDomainStatus = domain.DomainVerificationStatus(statusStr)
	return b, nil
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
