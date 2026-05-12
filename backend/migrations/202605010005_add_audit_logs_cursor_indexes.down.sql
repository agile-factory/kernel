-- ============================================================
-- Migration  : 202605010005_add_audit_logs_cursor_indexes (DOWN)
-- Description: Drop the composite cursor indexes added in the
--              UP migration.
--
-- DROP INDEX (without CONCURRENTLY) keeps this compatible with
-- golang-migrate's per-migration transaction wrapping.
-- ============================================================

DROP INDEX IF EXISTS idx_audit_logs_cursor_tenant;
DROP INDEX IF EXISTS idx_audit_logs_cursor_global;
