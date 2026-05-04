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
-- NOTE on CONCURRENTLY:
--   golang-migrate wraps each migration in a transaction by default,
--   which is incompatible with CREATE INDEX CONCURRENTLY. As this is
--   the initial schema bootstrap (audit_logs is empty when the migration
--   runs), a regular CREATE INDEX is safe and acquires only a brief lock.
--   For backfilling indexes on a live table, run an out-of-band script
--   with CONCURRENTLY instead of through golang-migrate.
-- ============================================================

CREATE INDEX IF NOT EXISTS idx_audit_logs_cursor_tenant
    ON audit_logs (tenant_id, occurred_at DESC, id ASC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_cursor_global
    ON audit_logs (occurred_at DESC, id ASC);
