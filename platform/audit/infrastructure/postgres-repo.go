package infrastructure

import (
	"context"
	"fmt"

	"github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAuditRepository implements the domain.AuditRepository using pgx.
type PostgresAuditRepository struct {
	db *pgxpool.Pool
}

// NewPostgresAuditRepository creates a new PostgresAuditRepository.
func NewPostgresAuditRepository(db *pgxpool.Pool) *PostgresAuditRepository {
	return &PostgresAuditRepository{db: db}
}

// Save persists an AuditLog. This is designed to be best-effort;
// any database error is wrapped but should typically be logged, not propagated
// to fail a broader business transaction unless specifically required.
func (r *PostgresAuditRepository) Save(ctx context.Context, log *domain.AuditLog) error {
	_, err := r.db.Exec(ctx, queryInsertAuditLog,
		log.ID,
		log.TenantID,
		log.ActorID,
		log.Action,
		log.TargetType,
		log.TargetID,
		log.Metadata,
		log.OccurredAt,
	)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrAuditSaveFailed, err)
	}
	return nil
}

// ListByTenant retrieves audit logs for a specific tenant with pagination.
func (r *PostgresAuditRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, queryListByTenant, tenantID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(
			&l.ID,
			&l.TenantID,
			&l.ActorID,
			&l.Action,
			&l.TargetType,
			&l.TargetID,
			&l.Metadata,
			&l.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, &l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit log iteration error: %w", err)
	}

	return logs, nil
}

// ListAll retrieves audit logs across all tenants with pagination.
// Intended for platform super-admins only.
func (r *PostgresAuditRepository) ListAll(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	rows, err := r.db.Query(ctx, queryListAll, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query all audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(
			&l.ID,
			&l.TenantID,
			&l.ActorID,
			&l.Action,
			&l.TargetType,
			&l.TargetID,
			&l.Metadata,
			&l.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, &l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit log iteration error: %w", err)
	}

	return logs, nil
}

// ── v2 cursor-based ──────────────────────────────────────────────────────────

// ListByTenantCursor retrieves at most limit rows for the given tenant using
// keyset pagination on (occurred_at DESC, id ASC).
// When after is nil the first page is returned (no keyset condition).
func (r *PostgresAuditRepository) ListByTenantCursor(
	ctx context.Context, tenantID string, limit int, after *domain.CursorToken,
) ([]*domain.AuditLog, error) {
	var (
		rows pgx_rows
		err  error
	)
	if after == nil {
		rows, err = r.db.Query(ctx, queryListByTenantCursorFirst, tenantID, limit)
	} else {
		rows, err = r.db.Query(ctx, queryListByTenantCursorAfter,
			tenantID, limit, after.OccurredAt, after.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs (cursor, tenant): %w", err)
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

// ListAllCursor retrieves at most limit rows across all tenants using
// keyset pagination on (occurred_at DESC, id ASC).
// When after is nil the first page is returned (no keyset condition).
func (r *PostgresAuditRepository) ListAllCursor(
	ctx context.Context, limit int, after *domain.CursorToken,
) ([]*domain.AuditLog, error) {
	var (
		rows pgx_rows
		err  error
	)
	if after == nil {
		rows, err = r.db.Query(ctx, queryListAllCursorFirst, limit)
	} else {
		rows, err = r.db.Query(ctx, queryListAllCursorAfter, limit, after.OccurredAt, after.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs (cursor, global): %w", err)
	}
	defer rows.Close()
	return scanAuditLogs(rows)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// pgx_rows is a local alias to allow the shared scanner to accept both query
// results from pgxpool.Pool.Query without an extra import cycle.
type pgx_rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

// scanAuditLogs iterates the rows returned by any SELECT query and maps each
// row into a domain.AuditLog pointer.
func scanAuditLogs(rows pgx_rows) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		if err := rows.Scan(
			&l.ID,
			&l.TenantID,
			&l.ActorID,
			&l.Action,
			&l.TargetType,
			&l.TargetID,
			&l.Metadata,
			&l.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("audit log iteration error: %w", err)
	}
	return logs, nil
}
