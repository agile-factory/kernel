-- ============================================================
-- Migration  : 202604301918_create_users_table (DOWN)
-- Description: Drop the users table, its indexes, and the
--              user_role enum type.
-- ============================================================

DROP INDEX IF EXISTS idx_users_tenant_id;
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
DROP TYPE  IF EXISTS user_role;
