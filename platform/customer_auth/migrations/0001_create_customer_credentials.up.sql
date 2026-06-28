-- platform/customer_auth migration 0001 — core auth tables (RFC-0034).

CREATE TABLE customer_credentials (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    end_customer_id  UUID NOT NULL UNIQUE REFERENCES end_customers(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    password_hash    VARCHAR(72) NOT NULL,
    last_login_at    TIMESTAMPTZ,
    failed_attempts  INTEGER NOT NULL DEFAULT 0,
    locked_until     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE customer_refresh_tokens (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    end_customer_id  UUID NOT NULL REFERENCES end_customers(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash       VARCHAR(64) NOT NULL UNIQUE,
    expires_at       TIMESTAMPTZ NOT NULL,
    revoked_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customer_refresh_token_lookup
    ON customer_refresh_tokens(token_hash)
    WHERE revoked_at IS NULL;

CREATE TABLE customer_reset_tokens (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    end_customer_id  UUID NOT NULL REFERENCES end_customers(id) ON DELETE CASCADE,
    token_hash       VARCHAR(64) NOT NULL UNIQUE,
    expires_at       TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    consumed_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customer_reset_token_lookup
    ON customer_reset_tokens(token_hash)
    WHERE consumed_at IS NULL;

-- OTP rate-limit lookup: find recent unconsumed tokens per end-customer.
CREATE INDEX idx_customer_reset_token_recent
    ON customer_reset_tokens(end_customer_id, created_at DESC)
    WHERE consumed_at IS NULL;
