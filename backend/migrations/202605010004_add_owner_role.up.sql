-- ============================================================
-- Migration  : 202605010004_add_owner_role (UP)
-- Description: Add ROLE_OWNER to the member_role ENUM.
--              ROLE_OWNER represents the founding member of a tenant
--              (the user who created it). Unlike ROLE_ADMIN, an owner
--              cannot be revoked by other admins and is the only role
--              allowed to delete the tenant.
--
-- NOTE: ALTER TYPE ... ADD VALUE is not transactional in PostgreSQL
-- (the value is committed immediately even inside a transaction).
-- The down migration cannot remove the value; see the .down.sql file.
-- ============================================================

ALTER TYPE member_role ADD VALUE IF NOT EXISTS 'ROLE_OWNER';
