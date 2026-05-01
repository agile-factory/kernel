-- ============================================================
-- Migration  : 202605010005_add_audit_logs_cursor_indexes (DOWN)
-- Description: Drop the composite cursor indexes added in the
--              UP migration.
-- ============================================================

DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_cursor_tenant;
DROP INDEX CONCURRENTLY IF EXISTS idx_audit_logs_cursor_global;
