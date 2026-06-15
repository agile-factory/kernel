package http

import (
	"time"

	"github.com/agile-factory/kernel/platform/audit/domain"
)

// AuditLogResponse is the public JSON representation of an audit log entry.
type AuditLogResponse struct {
	ID         string         `json:"id"`
	TenantID   string         `json:"tenantId,omitempty"`
	ActorID    string         `json:"actorId,omitempty"`
	Action     string         `json:"action"`
	TargetType string         `json:"targetType,omitempty"`
	TargetID   string         `json:"targetId,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	OccurredAt string         `json:"occurredAt"`
}

// toAuditLogResponse maps a domain.AuditLog to its HTTP response DTO.
func toAuditLogResponse(l *domain.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:         l.ID,
		TenantID:   l.TenantID,
		ActorID:    l.ActorID,
		Action:     string(l.Action),
		TargetType: l.TargetType,
		TargetID:   l.TargetID,
		Metadata:   l.Metadata,
		OccurredAt: l.OccurredAt.Format(time.RFC3339),
	}
}

// AuditLogListResponse is a paginated wrapper returned by the list endpoint.
type AuditLogListResponse struct {
	Data   []AuditLogResponse `json:"data"`
	Total  int                `json:"total"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
}
