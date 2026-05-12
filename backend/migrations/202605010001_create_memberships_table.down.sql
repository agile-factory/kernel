-- ============================================================
-- Migration  : 202605010001_create_memberships_table (DOWN)
-- Description: Drop the memberships table and its enum types.
-- ============================================================

DROP INDEX  IF EXISTS idx_memberships_status;
DROP INDEX  IF EXISTS idx_memberships_tenant_id;
DROP INDEX  IF EXISTS idx_memberships_user_id;
DROP TABLE  IF EXISTS memberships;
DROP TYPE   IF EXISTS member_status;
DROP TYPE   IF EXISTS member_role;
