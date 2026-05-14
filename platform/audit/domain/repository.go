package domain

import "context"

// AuditRepository defines the persistence contract for audit logs.
type AuditRepository interface {
	// Save persists an AuditLog (best-effort, errors are logged but not propagated).
	Save(ctx context.Context, log *AuditLog) error

	// ── v1 — offset-based (kept for backward compatibility) ──────────────────

	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*AuditLog, error)
	ListAll(ctx context.Context, limit, offset int) ([]*AuditLog, error)

	// ── v2 — cursor-based (keyset on occurred_at DESC, id ASC) ───────────────

	// ListByTenantCursor returns at most limit rows for the given tenant,
	// starting after the keyset position encoded in after.
	// When after is nil the first page is returned.
	ListByTenantCursor(ctx context.Context, tenantID string, limit int, after *CursorToken) ([]*AuditLog, error)

	// ListAllCursor returns at most limit rows across all tenants,
	// starting after the keyset position encoded in after.
	// When after is nil the first page is returned.
	ListAllCursor(ctx context.Context, limit int, after *CursorToken) ([]*AuditLog, error)
}
