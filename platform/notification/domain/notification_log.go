package domain

import (
	"context"
	"time"
)

// Status models the row-level lifecycle of a notification.
//
//   PENDING → SENT  (success, provider accepted)
//   PENDING → FAILED (transient error, retries already exhausted)
//   SENT    → BOUNCED (provider webhook reports a hard bounce)
type Status string

const (
	STATUS_PENDING Status = "PENDING"
	STATUS_SENT    Status = "SENT"
	STATUS_FAILED  Status = "FAILED"
	STATUS_BOUNCED Status = "BOUNCED"
)

// NotificationLog is the persisted audit row for a notification attempt.
type NotificationLog struct {
	ID          string
	TenantID    string
	RecipientID string
	EventType   EventType
	Channel     Channel
	Status      Status
	ExternalID  string         // provider message id (Resend, Postmark, Twilio…)
	Payload     map[string]any // template data — kept for redelivery / debug
	SentAt      *time.Time
	Error       string
	CreatedAt   time.Time
}

// NewPending builds a NotificationLog in PENDING status. The dispatcher
// inserts this row first, then flips it to SENT/FAILED based on the
// provider's response.
func NewPending(id, tenantID, recipientID string, eventType EventType, channel Channel, payload map[string]any) *NotificationLog {
	return &NotificationLog{
		ID:          id,
		TenantID:    tenantID,
		RecipientID: recipientID,
		EventType:   eventType,
		Channel:     channel,
		Status:      STATUS_PENDING,
		Payload:     payload,
		CreatedAt:   time.Now().UTC(),
	}
}

// MarkSent flips the row to SENT and records the external ID.
func (n *NotificationLog) MarkSent(externalID string) {
	n.Status = STATUS_SENT
	n.ExternalID = externalID
	now := time.Now().UTC()
	n.SentAt = &now
}

// MarkFailed records a delivery error.
func (n *NotificationLog) MarkFailed(err string) {
	n.Status = STATUS_FAILED
	n.Error = err
}

// NotificationLogRepository is the persistence contract.
type NotificationLogRepository interface {
	// Save inserts a fresh PENDING row.
	Save(ctx context.Context, log *NotificationLog) error
	// Update persists status / external_id / sent_at / error mutations.
	Update(ctx context.Context, log *NotificationLog) error
	// ListByTenant returns rows for a tenant in created_at DESC order.
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*NotificationLog, error)
}
