package domain

import (
	"context"
	"time"
)

// BrandRepository is the persistence contract for BrandSettings.
// Implementations live in infrastructure/; the domain never imports them.
type BrandRepository interface {
	// Upsert inserts or replaces the brand settings for the tenant.
	Upsert(ctx context.Context, b *BrandSettings) error

	// FindByTenantID returns the brand settings for the given tenant.
	// Returns ErrBrandNotFound when no row exists.
	FindByTenantID(ctx context.Context, tenantID string) (*BrandSettings, error)

	// FindByCustomDomain returns the brand settings for the given verified domain.
	// Returns ErrBrandNotFound when no match exists.
	FindByCustomDomain(ctx context.Context, domain string) (*BrandSettings, error)

	// UpdateDomainStatus persists only the custom_domain_status and
	// custom_domain_verified_at columns for the given tenant.
	UpdateDomainStatus(ctx context.Context, tenantID string, status DomainVerificationStatus, verifiedAt *time.Time) error
}
