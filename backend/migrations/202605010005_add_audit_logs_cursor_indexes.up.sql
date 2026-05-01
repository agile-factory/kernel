-- ============================================================
-- Migration  : 202605010005_add_audit_logs_cursor_indexes (UP)
-- Description: Add composite indexes to support cursor-based
--              (keyset) pagination on audit_logs.
--
-- The keyset query uses:
--   ORDER BY occurred_at DESC, id ASC
--   WHERE (occurred_at, id) < ($N::timestamptz, $M::uuid)
--
-- Two indexes are created:
--   1. Tenant-scoped  : filters on tenant_id first, then keyset
--   2. Global         : keyset only, used by the super-admin endpoint
--
-- CONCURRENTLY avoids a full table lock during index creation.
-- Safe to run on a live database (PostgreSQL 9.6+).
-- ============================================================

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_cursor_tenant
    ON audit_logs (tenant_id, occurred_at DESC, id ASC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_audit_logs_cursor_global
    ON audit_logs (occurred_at DESC, id ASC);
