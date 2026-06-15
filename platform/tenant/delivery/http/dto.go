package http

import (
	"errors"
	"strings"

	"github.com/agile-factory/kernel/platform/tenant/domain"
)

// ─── Create (admin) ──────────────────────────────────────────────────────────

// CreateTenantRequest is the payload to create a new tenant. The endpoint
// is super-admin only (POST /admin/v1/tenants) per RFC-0018 §9.
type CreateTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (r *CreateTenantRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(r.Slug) == "" {
		return errors.New("slug is required")
	}
	return nil
}

// ─── Update ──────────────────────────────────────────────────────────────────

// UpdateTenantRequest is the payload for PATCH /api/v1/tenants/{id}. Only
// the human-readable name is mutable; slug is immutable.
type UpdateTenantRequest struct {
	Name string `json:"name"`
}

func (r *UpdateTenantRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	return nil
}

// ─── Settings ────────────────────────────────────────────────────────────────

// UpdateSettingsRequest is the PATCH payload for /api/v1/tenants/{id}/settings.
// All fields are optional pointers so the caller can omit them — only
// supplied fields are applied (sparse update).
type UpdateSettingsRequest struct {
	LogoURL       *string `json:"logoUrl"`
	PrimaryColor  *string `json:"primaryColor"`
	Locale        *string `json:"locale"`
	Timezone      *string `json:"timezone"`
	Currency      *string `json:"currency"`
	DataRegion    *string `json:"dataRegion"`
	CustomDomain  *string `json:"customDomain"`
	PaymentRegion *string `json:"paymentRegion"`
	MerchantMode  *bool   `json:"merchantMode"`
	B2B2CEnabled  *bool   `json:"b2b2cEnabled"`
}

// ToPatch converts the wire shape into a domain.SettingsPatch. Enum-typed
// fields (DataRegion, PaymentRegion) are wrapped with their domain types.
func (r *UpdateSettingsRequest) ToPatch() domain.SettingsPatch {
	patch := domain.SettingsPatch{
		LogoURL:      r.LogoURL,
		PrimaryColor: r.PrimaryColor,
		Locale:       r.Locale,
		Timezone:     r.Timezone,
		Currency:     r.Currency,
		CustomDomain: r.CustomDomain,
		MerchantMode: r.MerchantMode,
		B2B2CEnabled: r.B2B2CEnabled,
	}
	if r.DataRegion != nil {
		dr := domain.DataRegion(*r.DataRegion)
		patch.DataRegion = &dr
	}
	if r.PaymentRegion != nil {
		pr := domain.PaymentRegion(*r.PaymentRegion)
		patch.PaymentRegion = &pr
	}
	return patch
}

// SettingsResponse mirrors the persisted shape, less the timestamps the
// client usually does not need.
type SettingsResponse struct {
	TenantID      string `json:"tenantId"`
	LogoURL       string `json:"logoUrl"`
	PrimaryColor  string `json:"primaryColor"`
	Locale        string `json:"locale"`
	Timezone      string `json:"timezone"`
	Currency      string `json:"currency"`
	DataRegion    string `json:"dataRegion"`
	CustomDomain  string `json:"customDomain"`
	PaymentRegion string `json:"paymentRegion"`
	MerchantMode  bool   `json:"merchantMode"`
	B2B2CEnabled  bool   `json:"b2b2cEnabled"`
}

func toSettingsResponse(s *domain.TenantSettings) SettingsResponse {
	return SettingsResponse{
		TenantID:      s.TenantID,
		LogoURL:       s.LogoURL,
		PrimaryColor:  s.PrimaryColor,
		Locale:        s.Locale,
		Timezone:      s.Timezone,
		Currency:      s.Currency,
		DataRegion:    string(s.DataRegion),
		CustomDomain:  s.CustomDomain,
		PaymentRegion: string(s.PaymentRegion),
		MerchantMode:  s.MerchantMode,
		B2B2CEnabled:  s.B2B2CEnabled,
	}
}

// ─── Tenant ──────────────────────────────────────────────────────────────────

// TenantResponse is the public representation of a Tenant.
type TenantResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}
