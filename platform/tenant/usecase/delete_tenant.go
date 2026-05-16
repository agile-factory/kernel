package usecase

import (
	"context"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/audit"
	auditdomain "github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/agile-factory/kernel/platform/tenant/domain"
)

// DeleteTenantUseCase soft-deletes a tenant (is_active = FALSE). The row
// stays in the table so audit logs and memberships remain valid; a separate
// GDPR hard-delete flow (RFC-0027) will physically remove rows. Permission
// gating (ROLE_OWNER only) is enforced at the HTTP handler.
type DeleteTenantUseCase struct {
	repo        domain.TenantRepository
	auditLogger audit.EventLogger
}

func NewDeleteTenantUseCase(repo domain.TenantRepository, auditLogger audit.EventLogger) *DeleteTenantUseCase {
	return &DeleteTenantUseCase{repo: repo, auditLogger: auditLogger}
}

// Execute soft-deletes the tenant.
func (uc *DeleteTenantUseCase) Execute(ctx context.Context, id, actorID string) error {
	tenant, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !tenant.IsActive {
		return nil // idempotent
	}

	if err := uc.repo.SoftDelete(ctx, id); err != nil {
		return errors.WrapInternal(err)
	}

	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   tenant.ID,
			ActorID:    actorID,
			Action:     auditdomain.ACTION_TENANT_DELETED,
			TargetType: "tenant",
			TargetID:   tenant.ID,
			Metadata: map[string]any{
				"slug": tenant.Slug,
			},
		})
	}
	return nil
}
