package domain

import "context"

// AuditRepository defines the write-only persistence contract.
// Queries (list, search) are reserved for a future read-model.
type AuditRepository interface {
	Save(ctx context.Context, log *AuditLog) error
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error)
	ListAll(ctx context.Context, limit, offset int) ([]*AuditLog, error)
}
