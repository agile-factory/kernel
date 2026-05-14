package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/platform/audit"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	"github.com/AgileFactory/kernel/platform/tenant/domain"
)

// CreateTenantUseCase handles the creation of a new tenant.
type CreateTenantUseCase struct {
	repo        domain.TenantRepository
	auditLogger audit.EventLogger
}

// NewCreateTenantUseCase creates a new CreateTenantUseCase.
func NewCreateTenantUseCase(repo domain.TenantRepository, auditLogger audit.EventLogger) *CreateTenantUseCase {
	return &CreateTenantUseCase{
		repo:        repo,
		auditLogger: auditLogger,
	}
}

// Execute creates a new tenant from the given name and slug.
// It enforces slug uniqueness before persisting.
func (uc *CreateTenantUseCase) Execute(ctx context.Context, id, name, slug string) (*domain.Tenant, error) {
	// Validate and normalize the slug
	normalizedSlug, err := domain.NewSlug(slug)
	if err != nil {
		return nil, err
	}

	// Enforce uniqueness
	exists, err := uc.repo.ExistsBySlug(ctx, normalizedSlug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, domain.ErrTenantAlreadyExists
	}

	// Build the aggregate
	tenant, err := domain.NewTenant(id, name, normalizedSlug)
	if err != nil {
		return nil, err
	}

	// Persist
	if err := uc.repo.Save(ctx, tenant); err != nil {
		return nil, errors.WrapInternal(err)
	}

	uc.auditLogger.Execute(ctx, audit.LogEventParams{
		TenantID:   tenant.ID,
		ActorID:    "", // no actor in this standalone context
		Action:     auditdomain.ACTION_TENANT_CREATED,
		TargetType: "tenant",
		TargetID:   tenant.ID,
		Metadata: map[string]any{
			"name": tenant.Name,
		},
	})

	return tenant, nil
}
