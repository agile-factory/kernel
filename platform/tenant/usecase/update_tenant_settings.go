package usecase

import (
	"context"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/audit"
	auditdomain "github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/agile-factory/kernel/platform/tenant/domain"
)

// UpdateTenantSettingsUseCase applies a sparse SettingsPatch to a tenant's
// settings row. Permission gating (active OWNER/ADMIN membership) is at the
// HTTP handler.
type UpdateTenantSettingsUseCase struct {
	repo        domain.TenantSettingsRepository
	auditLogger audit.EventLogger
}

func NewUpdateTenantSettingsUseCase(repo domain.TenantSettingsRepository, auditLogger audit.EventLogger) *UpdateTenantSettingsUseCase {
	return &UpdateTenantSettingsUseCase{repo: repo, auditLogger: auditLogger}
}

// Execute loads the current settings, applies the patch, persists. Returns
// the post-patch aggregate so the handler can serialise it back to the
// client without a second round-trip.
func (uc *UpdateTenantSettingsUseCase) Execute(ctx context.Context, tenantID, actorID string, patch domain.SettingsPatch) (*domain.TenantSettings, error) {
	current, err := uc.repo.Get(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	current.Apply(patch)
	if err := uc.repo.Update(ctx, current); err != nil {
		return nil, errors.WrapInternal(err)
	}

	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   tenantID,
			ActorID:    actorID,
			Action:     auditdomain.ACTION_SETTINGS_UPDATED,
			TargetType: "tenant_settings",
			TargetID:   tenantID,
			Metadata:   nil, // intentionally empty — full snapshot would be noisy
		})
	}

	return current, nil
}
