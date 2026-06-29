-- platform/customer_notifications migration 0001 — dispatch log (RFC-0036).
-- body column intentionally omitted: OTP codes and payment amounts must not
-- be stored in plaintext (Lucas amendment, ADR-0019).

CREATE TABLE customer_notification_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    end_customer_id  UUID REFERENCES end_customers(id) ON DELETE SET NULL,
    channel          VARCHAR(10) NOT NULL CHECK (channel IN ('EMAIL','SMS')),
    recipient        VARCHAR(255) NOT NULL,
    template_key     VARCHAR(100) NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                         CHECK (status IN ('PENDING','SENT','FAILED','BOUNCED')),
    external_id      VARCHAR(255),
    failure_reason   TEXT,
    sent_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cn_logs_tenant_customer
    ON customer_notification_logs(tenant_id, end_customer_id, created_at DESC);

CREATE INDEX idx_cn_logs_status_pending
    ON customer_notification_logs(status, created_at)
    WHERE status = 'PENDING';
