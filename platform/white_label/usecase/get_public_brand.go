package usecase

import (
	"context"

	"github.com/agile-factory/kernel/platform/white_label/domain"
)

// GetPublicBrandOutput wraps the public-facing brand settings.
type GetPublicBrandOutput struct {
	Brand *domain.BrandSettings
}

// GetPublicBrandUseCase returns brand settings for unauthenticated portal rendering.
// Lookup priority: custom domain (Host header) → tenantID query param.
// Only public fields are safe to return (colours, logo, AppName).
type GetPublicBrandUseCase struct {
	repo domain.BrandRepository
}

// NewGetPublicBrandUseCase builds the use case.
func NewGetPublicBrandUseCase(repo domain.BrandRepository) *GetPublicBrandUseCase {
	return &GetPublicBrandUseCase{repo: repo}
}

// ExecuteByDomain looks up brand settings by verified custom domain.
func (uc *GetPublicBrandUseCase) ExecuteByDomain(ctx context.Context, customDomain string) (GetPublicBrandOutput, error) {
	brand, err := uc.repo.FindByCustomDomain(ctx, customDomain)
	if err != nil {
		return GetPublicBrandOutput{}, err
	}
	return GetPublicBrandOutput{Brand: brand}, nil
}

// ExecuteByTenantID looks up brand settings by tenant ID query param.
func (uc *GetPublicBrandUseCase) ExecuteByTenantID(ctx context.Context, tenantID string) (GetPublicBrandOutput, error) {
	brand, err := uc.repo.FindByTenantID(ctx, tenantID)
	if err != nil {
		return GetPublicBrandOutput{}, err
	}
	return GetPublicBrandOutput{Brand: brand}, nil
}
