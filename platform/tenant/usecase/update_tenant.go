package usecase

import (
	"context"
	"strings"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/audit"
	auditdomain "github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/agile-factory/kernel/platform/tenant/domain"
)

// UpdateTenantUseCase renames a tenant. Slug is immutable. Permission gating
// (active OWNER/ADMIN membership) happens at the HTTP handler — the use case
// itself trusts its caller.
type UpdateTenantUseCase struct {
	repo        domain.TenantRepository
	auditLogger audit.EventLogger
}

func NewUpdateTenantUseCase(repo domain.TenantRepository, auditLogger audit.EventLogger) *UpdateTenantUseCase {
	return &UpdateTenantUseCase{repo: repo, auditLogger: auditLogger}
}

// Execute renames the tenant. ActorID is used only for the audit log entry.
func (uc *UpdateTenantUseCase) Execute(ctx context.Context, id, actorID, name string) (*domain.Tenant, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, domain.ErrInvalidTenantName
	}

	tenant, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateName(ctx, id, name); err != nil {
		return nil, errors.WrapInternal(err)
	}
	tenant.Name = name

	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   tenant.ID,
			ActorID:    actorID,
			Action:     auditdomain.ACTION_TENANT_UPDATED,
			TargetType: "tenant",
			TargetID:   tenant.ID,
			Metadata: map[string]any{
				"name": tenant.Name,
			},
		})
	}

	return tenant, nil
}
