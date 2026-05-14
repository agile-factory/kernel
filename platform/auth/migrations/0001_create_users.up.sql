-- ============================================================
-- Migration  : 202604301918_create_users_table (UP)
-- Description: Create the users table with role-based access
--              control and a foreign key to tenants.
-- ============================================================

CREATE TYPE user_role AS ENUM (
    'ROLE_ADMIN',
    'ROLE_AGENT',
    'ROLE_VIEWER'
);

CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL REFERENCES tenants (id) ON DELETE CASCADE,
    email      VARCHAR(320) NOT NULL UNIQUE,
    password   TEXT         NOT NULL,
    role       user_role    NOT NULL DEFAULT 'ROLE_VIEWER',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email     ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);
