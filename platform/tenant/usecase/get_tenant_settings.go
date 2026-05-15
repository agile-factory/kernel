package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/platform/tenant/domain"
)

// GetTenantSettingsUseCase reads the tenant_settings row. Permission gating
// (active membership in the tenant) is enforced at the HTTP handler.
type GetTenantSettingsUseCase struct {
	repo domain.TenantSettingsRepository
}

func NewGetTenantSettingsUseCase(repo domain.TenantSettingsRepository) *GetTenantSettingsUseCase {
	return &GetTenantSettingsUseCase{repo: repo}
}

func (uc *GetTenantSettingsUseCase) Execute(ctx context.Context, tenantID string) (*domain.TenantSettings, error) {
	return uc.repo.Get(ctx, tenantID)
}
