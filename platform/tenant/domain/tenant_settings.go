package domain

import (
	"context"
	"time"
)

// PaymentRegion gates which billing adapters are exposed to the tenant
// (RFC-0020). INTERNATIONAL → Stripe only ; CEDEAO → Stripe + Wave + Orange
// Money + Free Money.
type PaymentRegion string

const (
	PAYMENT_REGION_INTERNATIONAL PaymentRegion = "INTERNATIONAL"
	PAYMENT_REGION_CEDEAO        PaymentRegion = "CEDEAO"
)

// DataRegion controls geographic residency of tenant data (RFC-0027 GDPR
// pinning). Persisted in `tenant_settings.data_region`.
type DataRegion string

const (
	DATA_REGION_EU   DataRegion = "eu"
	DATA_REGION_US   DataRegion = "us"
	DATA_REGION_APAC DataRegion = "apac"
)

// TenantSettings carries the configurable, mostly cosmetic / commercial knobs
// a tenant administrator can flip without involving the platform team. One
// row per tenant (1:1, PK = tenant_id).
type TenantSettings struct {
	TenantID       string
	LogoURL        string
	PrimaryColor   string // hex "#RRGGBB"
	Locale         string
	Timezone       string
	Currency       string // ISO 4217
	DataRegion     DataRegion
	CustomDomain   string // empty when not set
	PaymentRegion  PaymentRegion
	MerchantMode   bool // B2B2C opt-in
	B2B2CEnabled   bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// DefaultSettings returns the row inserted alongside a freshly created
// tenant — INTERNATIONAL profile, no white-label, B2B2C disabled.
func DefaultSettings(tenantID string) *TenantSettings {
	now := time.Now().UTC()
	return &TenantSettings{
		TenantID:      tenantID,
		Locale:        "en-US",
		Timezone:      "UTC",
		Currency:      "USD",
		DataRegion:    DATA_REGION_EU,
		PaymentRegion: PAYMENT_REGION_INTERNATIONAL,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// SettingsPatch is a sparse update — only non-nil fields are applied. Using
// pointers lets the HTTP layer distinguish "not provided" from "explicitly
// set to zero value" (e.g. clearing custom_domain to empty string).
type SettingsPatch struct {
	LogoURL       *string
	PrimaryColor  *string
	Locale        *string
	Timezone      *string
	Currency      *string
	DataRegion    *DataRegion
	CustomDomain  *string
	PaymentRegion *PaymentRegion
	MerchantMode  *bool
	B2B2CEnabled  *bool
}

// Apply mutates s in place with the patch fields.
func (s *TenantSettings) Apply(p SettingsPatch) {
	if p.LogoURL != nil {
		s.LogoURL = *p.LogoURL
	}
	if p.PrimaryColor != nil {
		s.PrimaryColor = *p.PrimaryColor
	}
	if p.Locale != nil {
		s.Locale = *p.Locale
	}
	if p.Timezone != nil {
		s.Timezone = *p.Timezone
	}
	if p.Currency != nil {
		s.Currency = *p.Currency
	}
	if p.DataRegion != nil {
		s.DataRegion = *p.DataRegion
	}
	if p.CustomDomain != nil {
		s.CustomDomain = *p.CustomDomain
	}
	if p.PaymentRegion != nil {
		s.PaymentRegion = *p.PaymentRegion
	}
	if p.MerchantMode != nil {
		s.MerchantMode = *p.MerchantMode
	}
	if p.B2B2CEnabled != nil {
		s.B2B2CEnabled = *p.B2B2CEnabled
	}
	s.UpdatedAt = time.Now().UTC()
}

// TenantSettingsRepository persists tenant settings.
type TenantSettingsRepository interface {
	// Save inserts a brand-new row (used at tenant creation).
	Save(ctx context.Context, s *TenantSettings) error
	// Get retrieves the settings for tenantID. Returns ErrTenantSettingsNotFound
	// when no row exists yet (this can only happen for legacy tenants).
	Get(ctx context.Context, tenantID string) (*TenantSettings, error)
	// Update persists the full row's mutable fields back. The caller must
	// have called Apply on the loaded aggregate first.
	Update(ctx context.Context, s *TenantSettings) error
}
