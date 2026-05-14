-- ============================================================
-- Migration  : 0003_create_invite_tokens (UP)
-- Description: Per RFC-0018 §6. Issued when an admin invites a
--              new member; the invitee redeems via
--              POST /api/v1/auth/accept-invite. UNIQUE(token)
--              prevents double-redemption; partial indexes keep
--              scans cheap on the still-pending set.
-- ============================================================

CREATE TABLE IF NOT EXISTS invite_tokens (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    invited_by  UUID         NOT NULL REFERENCES users   (id),
    email       VARCHAR(320) NOT NULL,
    role        member_role  NOT NULL,
    token       VARCHAR(128) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ  NOT NULL DEFAULT (NOW() + INTERVAL '7 days'),
    consumed_at TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Hot path: redeem by token, only pending rows.
CREATE INDEX IF NOT EXISTS idx_invite_tokens_pending
    ON invite_tokens (token)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

-- Admin: find pending invite for (tenant, email) — used to detect
-- duplicate invites and to support resend.
CREATE INDEX IF NOT EXISTS idx_invite_tokens_tenant_email
    ON invite_tokens (tenant_id, email)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;
