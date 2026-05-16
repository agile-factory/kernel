package usecase

import (
	"context"

	"github.com/agile-factory/kernel/platform/tenant/domain"
)

// FindTenantUseCase handles retrieval of a tenant by ID or slug.
type FindTenantUseCase struct {
	repo domain.TenantRepository
}

// NewFindTenantUseCase creates a new FindTenantUseCase.
func NewFindTenantUseCase(repo domain.TenantRepository) *FindTenantUseCase {
	return &FindTenantUseCase{repo: repo}
}

// ByID retrieves a tenant by its UUID.
func (uc *FindTenantUseCase) ByID(ctx context.Context, id string) (*domain.Tenant, error) {
	tenant, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, domain.ErrTenantNotFound
	}
	return tenant, nil
}

// BySlug retrieves a tenant by its slug.
func (uc *FindTenantUseCase) BySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	normalizedSlug, err := domain.NewSlug(slug)
	if err != nil {
		return nil, err
	}

	tenant, err := uc.repo.FindBySlug(ctx, normalizedSlug)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, domain.ErrTenantNotFound
	}
	return tenant, nil
}
