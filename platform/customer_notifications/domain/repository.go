package domain

import (
	"context"
	"time"
)

// NotificationLogRepository persists dispatch attempts.
type NotificationLogRepository interface {
	// Create inserts a new log entry.
	Create(ctx context.Context, log *CustomerNotificationLog) error

	// UpdateStatus updates the status, external ID, and failure reason.
	UpdateStatus(ctx context.Context, id string, status DeliveryStatus, externalID, failureReason string, sentAt *time.Time) error

	// List returns a cursor-paginated list filtered by the given options.
	List(ctx context.Context, opts ListLogsOptions) ([]*CustomerNotificationLog, string, error)
}

// ListLogsOptions controls pagination and filtering for ListCustomerNotificationLogs.
type ListLogsOptions struct {
	TenantID      string
	EndCustomerID string // optional filter
	Channel       Channel
	Status        DeliveryStatus
	TemplateKey   string
	Cursor        string
	Limit         int // 0 → default 50
}

// CustomerTemplateRepository persists per-tenant template overrides.
type CustomerTemplateRepository interface {
	// Upsert inserts or replaces a template for (tenant_id, key, channel, locale).
	Upsert(ctx context.Context, t *CustomerTemplate) error

	// Find looks up the tenant's override for the given key, channel, locale.
	// Returns ErrTemplateNotFound when absent.
	Find(ctx context.Context, tenantID, key string, ch Channel, locale string) (*CustomerTemplate, error)

	// Delete removes a tenant override (reverts to default).
	Delete(ctx context.Context, tenantID, key string, ch Channel, locale string) error

	// ListByTenant returns all templates overrides for the tenant.
	ListByTenant(ctx context.Context, tenantID string) ([]*CustomerTemplate, error)
}
