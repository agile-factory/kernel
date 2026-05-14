package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/AgileFactory/kernel/platform/audit/domain"
	"github.com/google/uuid"
)

// auditWriteTimeout caps how long a single fire-and-forget Save may run after
// the originating request returns. Long enough to absorb a transient pool wait,
// short enough that pathological writes don't pile up forever.
const auditWriteTimeout = 5 * time.Second

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

// Execute persists the audit event without blocking the caller's request path.
// The save uses a context derived from the caller's context (preserving values
// such as request ID / trace ID) but with cancellation detached, so the write
// is not aborted when the HTTP handler returns.
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

	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), auditWriteTimeout)
	go func() {
		defer cancel()
		if err := uc.repo.Save(bgCtx, log); err != nil {
			slog.Error("audit: best-effort save failed",
				"error", err,
				"tenant_id", params.TenantID,
				"actor_id", params.ActorID,
				"action", params.Action,
			)
		}
	}()
}
