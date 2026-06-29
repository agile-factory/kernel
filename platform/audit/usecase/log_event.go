package usecase

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// auditLogPayload is the typed contract stored in the event_outbox payload.
// Using a typed struct (rather than ad-hoc map helpers) ensures the publisher
// and consumer agree on field names and types at compile time.
type auditLogPayload struct {
	TenantID   string         `json:"tenant_id"`
	ActorID    string         `json:"actor_id"`
	Action     string         `json:"action"`
	TargetType string         `json:"target_type"`
	TargetID   string         `json:"target_id"`
	Metadata   map[string]any `json:"metadata"`
	OccurredAt time.Time      `json:"occurred_at"`
}

// LogEventUseCase publishes an audit event to the outbox bus.
// Execute does not return an error: callers MUST NOT branch on audit success.
// Publish failures are logged; the business operation is never blocked.
type LogEventUseCase struct {
	db        *pgxpool.Pool
	publisher events.Publisher
}

// NewLogEventUseCase creates a LogEventUseCase backed by the outbox publisher.
func NewLogEventUseCase(db *pgxpool.Pool, publisher events.Publisher) *LogEventUseCase {
	return &LogEventUseCase{db: db, publisher: publisher}
}

// Execute publishes the audit event to event_outbox in a dedicated transaction.
// The at-least-once dispatcher (RFC-0015) delivers it to ConsumeAuditLogUseCase,
// which persists it to audit_logs with ON CONFLICT (outbox_event_id) DO NOTHING.
func (uc *LogEventUseCase) Execute(ctx context.Context, params LogEventParams) {
	payload := auditLogPayload{
		TenantID:   params.TenantID,
		ActorID:    params.ActorID,
		Action:     string(params.Action),
		TargetType: params.TargetType,
		TargetID:   params.TargetID,
		Metadata:   params.Metadata,
		OccurredAt: time.Now().UTC(),
	}

	err := txctx.ExecInTx(ctx, uc.db, func(ctx context.Context) error {
		m, err := toPayloadMap(payload)
		if err != nil {
			return err
		}
		return uc.publisher.Publish(ctx, events.Event{
			Type:    "audit.log_created",
			Payload: m,
			Meta: events.EventMeta{
				TenantID:   params.TenantID,
				ActorID:    params.ActorID,
				OccurredAt: payload.OccurredAt,
			},
		})
	})
	if err != nil {
		slog.Error("audit: outbox publish failed",
			"error", err,
			"action", params.Action,
			"tenant_id", params.TenantID,
		)
	}
}

// toPayloadMap JSON-round-trips the typed struct to map[string]any so it can
// be stored in events.Event.Payload (which is map[string]any by contract).
func toPayloadMap(p auditLogPayload) (map[string]any, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}
