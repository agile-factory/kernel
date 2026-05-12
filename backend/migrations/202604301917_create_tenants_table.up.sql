-- ============================================================
-- Migration  : 202604301917_create_tenants_table (UP)
-- Description: Create the tenants table.
--              Tenants represent isolated organizational units
--              in the multi-tenant architecture.
-- ============================================================

CREATE TABLE IF NOT EXISTS tenants (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL UNIQUE,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants (slug);
