-- ============================================================
-- Migration  : 0005_create_refresh_tokens (UP)
-- Description: Per RFC-0018 §5. The client receives a random
--              opaque token; we only store SHA-256(token), so a
--              DB dump can't be used to mint new sessions. Rolling
--              refresh: each /refresh call revokes the current
--              token and issues a new one — replaying an old token
--              is detected at the unique constraint.
-- ============================================================

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users   (id) ON DELETE CASCADE,
    tenant_id  UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    token_hash CHAR(64)     NOT NULL UNIQUE,                 -- hex SHA-256
    expires_at TIMESTAMPTZ  NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Hot path: validate a refresh attempt — still active.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_active
    ON refresh_tokens (token_hash)
    WHERE revoked_at IS NULL;

-- Admin/scrubbing scan: revoked tokens by user, useful for forced logout.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user
    ON refresh_tokens (user_id, created_at DESC);
