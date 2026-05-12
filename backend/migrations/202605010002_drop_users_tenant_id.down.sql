-- ============================================================
-- Migration  : 202605010002_drop_users_tenant_id (DOWN)
-- Description: Restore the tenant_id column on users.
--              Note: data will NOT be backfilled automatically;
--              this is a best-effort structural rollback only.
-- ============================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS tenant_id UUID REFERENCES tenants (id) ON DELETE SET NULL;
