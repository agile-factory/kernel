package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/white_label/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxAppNameLen       = 50
	maxEmailFooterLen   = 2000
)

var hexColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

var validFontFamilies = map[string]bool{
	"inter": true, "roboto": true, "lato": true, "custom": true,
}

// UpdateBrandSettingsInput carries the mutable fields of BrandSettings.
// Empty string means "keep existing value" for string fields.
type UpdateBrandSettingsInput struct {
	AppName         string
	LogoURL         string
	FaviconURL      string
	PrimaryColor    string
	SecondaryColor  string
	AccentColor     string
	FontFamily      string
	CustomFontURL   string
	CustomDomain    string
	EmailFooterHTML string
	EmailLogoURL    string
	TermsURL        string
	PrivacyURL      string
	SupportEmail    string
	SupportPhone    string
}

// UpdateBrandSettingsUseCase persists brand settings for the authenticated tenant.
type UpdateBrandSettingsUseCase struct {
	db        *pgxpool.Pool
	repo      domain.BrandRepository
	checker   permission.Checker
	publisher events.Publisher
}

// NewUpdateBrandSettingsUseCase builds the use case.
func NewUpdateBrandSettingsUseCase(
	db *pgxpool.Pool,
	repo domain.BrandRepository,
	checker permission.Checker,
	publisher events.Publisher,
) *UpdateBrandSettingsUseCase {
	return &UpdateBrandSettingsUseCase{db: db, repo: repo, checker: checker, publisher: publisher}
}

// Execute validates and persists brand settings, then publishes white_label.brand_updated.
func (uc *UpdateBrandSettingsUseCase) Execute(ctx context.Context, input UpdateBrandSettingsInput) (*domain.BrandSettings, error) {
	if err := uc.checker.Require(ctx, permission.BrandWrite); err != nil {
		return nil, err
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	actorID := kernelauth.UserIDFromContext(ctx)

	if err := validateBrandInput(input); err != nil {
		return nil, err
	}

	// Load existing or start fresh.
	existing, err := uc.repo.FindByTenantID(ctx, tenantID)
	if err == domain.ErrBrandNotFound {
		existing = &domain.BrandSettings{
			TenantID:           tenantID,
			PrimaryColor:       "#000000",
			SecondaryColor:     "#FFFFFF",
			AccentColor:        "#0066FF",
			FontFamily:         "inter",
			CustomDomainStatus: domain.DomainNotSet,
		}
	} else if err != nil {
		return nil, err
	}

	// Apply non-empty fields.
	applyString := func(dst *string, src string) {
		if src != "" {
			*dst = src
		}
	}
	applyString(&existing.AppName, input.AppName)
	applyString(&existing.LogoURL, input.LogoURL)
	applyString(&existing.FaviconURL, input.FaviconURL)
	applyString(&existing.PrimaryColor, input.PrimaryColor)
	applyString(&existing.SecondaryColor, input.SecondaryColor)
	applyString(&existing.AccentColor, input.AccentColor)
	applyString(&existing.FontFamily, input.FontFamily)
	applyString(&existing.CustomFontURL, input.CustomFontURL)
	applyString(&existing.EmailFooterHTML, input.EmailFooterHTML)
	applyString(&existing.EmailLogoURL, input.EmailLogoURL)
	applyString(&existing.TermsURL, input.TermsURL)
	applyString(&existing.PrivacyURL, input.PrivacyURL)
	applyString(&existing.SupportEmail, input.SupportEmail)
	applyString(&existing.SupportPhone, input.SupportPhone)

	// Custom domain change: generate verification token, reset to PENDING.
	if input.CustomDomain != "" && input.CustomDomain != existing.CustomDomain {
		existing.CustomDomain = input.CustomDomain
		existing.CustomDomainToken = "_kernel-verify=" + uuid.New().String()
		existing.CustomDomainStatus = domain.DomainPending
		existing.CustomDomainVerifiedAt = nil
	}

	now := time.Now().UTC()
	existing.UpdatedAt = now

	err = execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.repo.Upsert(ctx, existing); err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, events.Event{
			Type:    "white_label.brand_updated",
			Payload: map[string]any{"tenant_id": tenantID},
			Meta:    events.EventMeta{TenantID: tenantID, ActorID: actorID, OccurredAt: now},
		})
	})
	if err != nil {
		return nil, err
	}

	return existing, nil
}

func validateBrandInput(input UpdateBrandSettingsInput) error {
	if input.AppName != "" && len(input.AppName) > maxAppNameLen {
		return apperrors.BadRequest(fmt.Sprintf("app_name must be at most %d characters", maxAppNameLen))
	}

	for name, val := range map[string]string{
		"primary_color":   input.PrimaryColor,
		"secondary_color": input.SecondaryColor,
		"accent_color":    input.AccentColor,
	} {
		if val != "" && !hexColorRe.MatchString(val) {
			return apperrors.BadRequest(fmt.Sprintf("%s must be a valid hex color (#RRGGBB)", name))
		}
	}

	httpsFields := map[string]string{
		"logo_url":       input.LogoURL,
		"favicon_url":    input.FaviconURL,
		"custom_font_url": input.CustomFontURL,
		"email_logo_url": input.EmailLogoURL,
		"terms_url":      input.TermsURL,
		"privacy_url":    input.PrivacyURL,
	}
	for name, val := range httpsFields {
		if val != "" && !strings.HasPrefix(val, "https://") {
			return apperrors.BadRequest(fmt.Sprintf("%s must start with https://", name))
		}
	}

	if input.EmailFooterHTML != "" && len(input.EmailFooterHTML) > maxEmailFooterLen {
		return apperrors.BadRequest(fmt.Sprintf("email_footer_html must be at most %d characters", maxEmailFooterLen))
	}

	if input.FontFamily != "" {
		if !validFontFamilies[input.FontFamily] {
			return apperrors.BadRequest("font_family must be one of: inter, roboto, lato, custom")
		}
	}

	if input.CustomFontURL != "" && input.FontFamily != "custom" && input.FontFamily != "" {
		return apperrors.BadRequest("custom_font_url requires font_family to be 'custom'")
	}

	return nil
}
