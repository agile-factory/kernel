-- ============================================================
-- Migration  : 202604301917_create_tenants_table (DOWN)
-- Description: Drop the tenants table and its indexes.
-- ============================================================

DROP INDEX IF EXISTS idx_tenants_slug;
DROP TABLE IF EXISTS tenants;
