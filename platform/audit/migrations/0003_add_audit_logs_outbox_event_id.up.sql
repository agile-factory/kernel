-- ============================================================
-- Migration  : 0003_add_audit_logs_outbox_event_id (UP)
-- Description: Add outbox_event_id for at-least-once idempotency.
--              The UNIQUE constraint ensures duplicate deliveries from the
--              outbox dispatcher are silently discarded via ON CONFLICT DO NOTHING.
--              NULL is allowed for rows written before this migration (pre-outbox).
-- ============================================================

ALTER TABLE audit_logs
    ADD COLUMN outbox_event_id UUID UNIQUE;
