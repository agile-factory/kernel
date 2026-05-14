-- ============================================================
-- Migration  : 202605010002_drop_users_tenant_id (UP)
-- Description: Remove the tenant_id column from users now that
--              the memberships table handles the user↔tenant
--              relationship. This is a breaking schema change.
-- ============================================================

ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;
