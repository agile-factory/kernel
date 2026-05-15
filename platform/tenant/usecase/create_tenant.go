package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/kernel/txctx"
	"github.com/AgileFactory/kernel/platform/audit"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	"github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CreateTenantUseCase handles the atomic creation of a tenant + its default
// tenant_settings row. Per RFC-0018 §9, this use case is no longer exposed
// publicly: it is invoked by SignupOwner (founding owner self-serve) and by
// AdminCreateTenant (super-admin onboarding).
type CreateTenantUseCase struct {
	pool         *pgxpool.Pool
	repo         domain.TenantRepository
	settingsRepo domain.TenantSettingsRepository
	auditLogger  audit.EventLogger
}

// NewCreateTenantUseCase wires the use case. pool/settingsRepo are required
// so the tenant + its default settings land in the same transaction.
func NewCreateTenantUseCase(
	pool *pgxpool.Pool,
	repo domain.TenantRepository,
	settingsRepo domain.TenantSettingsRepository,
	auditLogger audit.EventLogger,
) *CreateTenantUseCase {
	return &CreateTenantUseCase{
		pool:         pool,
		repo:         repo,
		settingsRepo: settingsRepo,
		auditLogger:  auditLogger,
	}
}

// Execute creates a tenant + its default settings row in a single
// transaction. Slug uniqueness is enforced inside the tx so a concurrent
// signup with the same slug is rejected against the same snapshot.
func (uc *CreateTenantUseCase) Execute(ctx context.Context, id, name, slug string) (*domain.Tenant, error) {
	normalizedSlug, err := domain.NewSlug(slug)
	if err != nil {
		return nil, err
	}

	var tenant *domain.Tenant
	txErr := txctx.ExecInTx(ctx, uc.pool, func(ctx context.Context) error {
		exists, err := uc.repo.ExistsBySlug(ctx, normalizedSlug)
		if err != nil {
			return err
		}
		if exists {
			return domain.ErrTenantAlreadyExists
		}

		tenant, err = domain.NewTenant(id, name, normalizedSlug)
		if err != nil {
			return err
		}
		if err := uc.repo.Save(ctx, tenant); err != nil {
			return errors.WrapInternal(err)
		}
		if err := uc.settingsRepo.Save(ctx, domain.DefaultSettings(tenant.ID)); err != nil {
			return errors.WrapInternal(err)
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   tenant.ID,
			Action:     auditdomain.ACTION_TENANT_CREATED,
			TargetType: "tenant",
			TargetID:   tenant.ID,
			Metadata: map[string]any{
				"name": tenant.Name,
				"slug": tenant.Slug,
			},
		})
	}

	return tenant, nil
}
