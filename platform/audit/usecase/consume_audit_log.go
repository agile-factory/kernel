package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/google/uuid"
)

// ConsumeAuditLogUseCase is the at-least-once handler for "audit.log_created"
// events delivered by the kernel/events dispatcher. It persists the event
// payload to audit_logs using ON CONFLICT (outbox_event_id) DO NOTHING, so
// duplicate deliveries are silently discarded.
type ConsumeAuditLogUseCase struct {
	repo domain.AuditRepository
}

// NewConsumeAuditLogUseCase creates a consumer backed by the audit repository.
func NewConsumeAuditLogUseCase(repo domain.AuditRepository) *ConsumeAuditLogUseCase {
	return &ConsumeAuditLogUseCase{repo: repo}
}

// Handle deserialises the event payload and writes it to audit_logs.
// ev.ID (the outbox row UUID) is stored as outbox_event_id — the UNIQUE
// constraint ensures the insert is a no-op on redelivery.
func (uc *ConsumeAuditLogUseCase) Handle(ctx context.Context, ev events.Event) error {
	var p auditLogPayload
	b, err := json.Marshal(ev.Payload)
	if err != nil {
		return fmt.Errorf("audit consumer: marshal payload: %w", err)
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return fmt.Errorf("audit consumer: unmarshal payload: %w", err)
	}

	meta := p.Metadata
	if meta == nil {
		meta = map[string]any{}
	}

	log := &domain.AuditLog{
		ID:            uuid.NewString(),
		TenantID:      p.TenantID,
		ActorID:       p.ActorID,
		Action:        domain.AuditAction(p.Action),
		TargetType:    p.TargetType,
		TargetID:      p.TargetID,
		Metadata:      meta,
		OccurredAt:    p.OccurredAt,
		OutboxEventID: ev.ID,
	}
	return uc.repo.Save(ctx, log)
}
