-- ============================================================
-- Migration  : 0001_create_notification_logs (UP)
-- Description: Append-only log of outbound notifications per
--              RFC-0019 §7. The dispatcher writes one row per
--              attempt; status transitions from PENDING → SENT
--              or FAILED. Bounce processing (provider webhooks)
--              flips SENT → BOUNCED later.
-- ============================================================

CREATE TABLE IF NOT EXISTS notification_logs (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         REFERENCES tenants (id),
    recipient_id UUID         REFERENCES users   (id),
    event_type   VARCHAR(100) NOT NULL,
    channel      VARCHAR(20)  NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'PENDING',
    external_id  VARCHAR(255),                                 -- provider message id (Resend, Postmark…)
    payload      JSONB,
    sent_at      TIMESTAMPTZ,
    error        TEXT,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT notification_logs_channel_check
        CHECK (channel IN ('email','sms','whatsapp','in_app','push','webhook')),
    CONSTRAINT notification_logs_status_check
        CHECK (status IN ('PENDING','SENT','FAILED','BOUNCED'))
);

-- Hot path: list a tenant's recent notifications (admin views).
CREATE INDEX IF NOT EXISTS idx_notification_logs_tenant_created
    ON notification_logs (tenant_id, created_at DESC);

-- Operator scan for stuck PENDING rows (alerting / replay).
CREATE INDEX IF NOT EXISTS idx_notification_logs_pending
    ON notification_logs (created_at)
    WHERE status = 'PENDING';
