-- ============================================================
-- Migration  : 202605010001_create_memberships_table (UP)
-- Description: Create the memberships table to model the
--              many-to-many relationship between users and tenants.
--              Each membership carries its own role and lifecycle
--              status (PENDING → ACTIVE → REVOKED).
-- ============================================================

CREATE TYPE member_role AS ENUM (
    'ROLE_ADMIN',
    'ROLE_AGENT',
    'ROLE_VIEWER'
);

CREATE TYPE member_status AS ENUM (
    'PENDING',
    'ACTIVE',
    'REVOKED'
);

CREATE TABLE IF NOT EXISTS memberships (
    id          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id   UUID          NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    role        member_role   NOT NULL DEFAULT 'ROLE_VIEWER',
    status      member_status NOT NULL DEFAULT 'PENDING',
    invited_by  UUID          REFERENCES users (id) ON DELETE SET NULL,
    joined_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_memberships_user_tenant UNIQUE (user_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_user_id   ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_tenant_id ON memberships (tenant_id);
CREATE INDEX IF NOT EXISTS idx_memberships_status    ON memberships (status);
