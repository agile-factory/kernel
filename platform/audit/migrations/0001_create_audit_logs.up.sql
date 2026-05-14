-- ============================================================
-- Migration  : 202605010003_create_audit_logs_table (UP)
-- Description: Create the audit_logs table to store an
--              immutable trail of significant actions.
-- ============================================================

CREATE TYPE audit_action AS ENUM (
    'MEMBER_INVITED',
    'MEMBER_ACCEPTED',
    'MEMBER_REVOKED',
    'USER_REGISTERED',
    'USER_LOGGED_IN',
    'TENANT_CREATED'
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        REFERENCES tenants(id) ON DELETE SET NULL,
    actor_id    UUID        REFERENCES users(id)   ON DELETE SET NULL,
    action      audit_action NOT NULL,
    target_type VARCHAR(64),
    target_id   UUID,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_id   ON audit_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id    ON audit_logs (actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action      ON audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_occurred_at ON audit_logs (occurred_at DESC);
