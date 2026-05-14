-- ============================================================
-- Migration  : 0004_create_email_verification_tokens (UP)
-- Description: Per RFC-0018 §4. Tokens issued at signup-owner /
--              re-send. The opaque token is what the user clicks;
--              we store only it (and its expiration) — there is no
--              hash because the token IS the secret (UUID v7-like,
--              128 chars URL-safe).
-- ============================================================

CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token       VARCHAR(128) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Hot path: lookup by token for the still-valid, non-consumed rows.
CREATE INDEX IF NOT EXISTS idx_email_verification_token_pending
    ON email_verification_tokens (token)
    WHERE consumed_at IS NULL;

-- Used to find an unconsumed token for a given user (resend flow).
CREATE INDEX IF NOT EXISTS idx_email_verification_user_pending
    ON email_verification_tokens (user_id)
    WHERE consumed_at IS NULL;
