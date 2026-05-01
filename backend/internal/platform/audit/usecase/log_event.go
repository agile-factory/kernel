package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"
	"github.com/google/uuid"
)

// LogEventUseCase persists a single audit log entry.
// It is intentionally fire-and-forget: callers MUST NOT branch on its error.
type LogEventUseCase struct {
	repo domain.AuditRepository
}

func NewLogEventUseCase(repo domain.AuditRepository) *LogEventUseCase {
	return &LogEventUseCase{
		repo: repo,
	}
}

func (uc *LogEventUseCase) Execute(ctx context.Context, params LogEventParams) {
	log := domain.NewAuditLog(
		uuid.NewString(),
		params.TenantID,
		params.ActorID,
		params.Action,
		params.TargetType,
		params.TargetID,
		params.Metadata,
	)
	_ = uc.repo.Save(ctx, log) // best-effort, deliberate ignore of error
}
