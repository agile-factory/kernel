package domain

import "time"

// AuditAction represents the type of audited action.
type AuditAction string

const (
	ACTION_MEMBER_INVITED   AuditAction = "MEMBER_INVITED"
	ACTION_MEMBER_ACCEPTED  AuditAction = "MEMBER_ACCEPTED"
	ACTION_MEMBER_REVOKED   AuditAction = "MEMBER_REVOKED"
	ACTION_USER_REGISTERED  AuditAction = "USER_REGISTERED"
	ACTION_USER_LOGGED_IN   AuditAction = "USER_LOGGED_IN"
	ACTION_TENANT_CREATED   AuditAction = "TENANT_CREATED"
	ACTION_TENANT_UPDATED   AuditAction = "TENANT_UPDATED"
	ACTION_TENANT_DELETED   AuditAction = "TENANT_DELETED"
	ACTION_SETTINGS_UPDATED AuditAction = "SETTINGS_UPDATED"
)

// AuditLog is an immutable record of a single audited event.
type AuditLog struct {
	ID            string
	TenantID      string         // empty when not tenant-scoped
	ActorID       string         // empty for system events
	Action        AuditAction
	TargetType    string         // e.g. "membership", "user", "tenant"
	TargetID      string         // UUID of the affected resource
	Metadata      map[string]any // extra context (role, email, …)
	OccurredAt    time.Time
	OutboxEventID string // outbox row UUID; empty for pre-outbox rows; used for deduplication
}

// NewAuditLog constructs an AuditLog with OccurredAt set to now.
func NewAuditLog(id, tenantID, actorID string, action AuditAction, targetType, targetID string, meta map[string]any) *AuditLog {
	if meta == nil {
		meta = make(map[string]any)
	}
	return &AuditLog{
		ID:         id,
		TenantID:   tenantID,
		ActorID:    actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Metadata:   meta,
		OccurredAt: time.Now().UTC(),
	}
}
