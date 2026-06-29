package usecase

import (
	"context"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/white_label/domain"
)

// GetBrandSettingsOutput wraps the brand settings for the authenticated tenant.
type GetBrandSettingsOutput struct {
	Brand *domain.BrandSettings
}

// GetBrandSettingsUseCase returns the brand settings for the authenticated tenant.
// When no settings row exists yet, it returns a BrandSettings with default values.
type GetBrandSettingsUseCase struct {
	repo    domain.BrandRepository
	checker permission.Checker
}

// NewGetBrandSettingsUseCase builds the use case.
func NewGetBrandSettingsUseCase(repo domain.BrandRepository, checker permission.Checker) *GetBrandSettingsUseCase {
	return &GetBrandSettingsUseCase{repo: repo, checker: checker}
}

// Execute returns the brand settings, falling back to defaults when absent.
func (uc *GetBrandSettingsUseCase) Execute(ctx context.Context) (GetBrandSettingsOutput, error) {
	if err := uc.checker.Require(ctx, permission.BrandRead); err != nil {
		return GetBrandSettingsOutput{}, err
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)

	brand, err := uc.repo.FindByTenantID(ctx, tenantID)
	if err == domain.ErrBrandNotFound {
		// Return sensible defaults — no row yet, not an error.
		brand = &domain.BrandSettings{
			TenantID:           tenantID,
			PrimaryColor:       "#000000",
			SecondaryColor:     "#FFFFFF",
			AccentColor:        "#0066FF",
			FontFamily:         "inter",
			CustomDomainStatus: domain.DomainNotSet,
		}
		return GetBrandSettingsOutput{Brand: brand}, nil
	}
	if err != nil {
		return GetBrandSettingsOutput{}, err
	}

	return GetBrandSettingsOutput{Brand: brand}, nil
}
