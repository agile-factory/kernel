-- platform/customer_notifications migration 0002 — per-tenant template overrides (RFC-0036).

CREATE TABLE customer_templates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key          VARCHAR(100) NOT NULL,
    channel      VARCHAR(10) NOT NULL CHECK (channel IN ('EMAIL','SMS')),
    locale       VARCHAR(10) NOT NULL DEFAULT 'fr',
    subject_tmpl VARCHAR(500),
    body_tmpl    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, key, channel, locale)
);
