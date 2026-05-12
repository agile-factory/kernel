-- ============================================================
-- Migration  : 202605010003_create_audit_logs_table (DOWN)
-- Description: Rollback the audit_logs table creation.
-- ============================================================

DROP TABLE IF EXISTS audit_logs;
DROP TYPE IF EXISTS audit_action;
