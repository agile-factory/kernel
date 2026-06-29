-- ============================================================
-- Migration  : 0003_add_audit_logs_outbox_event_id (DOWN)
-- Description: Remove the outbox deduplication column.
-- ============================================================

ALTER TABLE audit_logs
    DROP COLUMN outbox_event_id;
