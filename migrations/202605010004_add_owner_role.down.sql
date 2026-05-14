-- ============================================================
-- Migration  : 202605010004_add_owner_role (DOWN)
-- Description: PostgreSQL does not support removing values from
--              an ENUM type. Rolling back this migration requires
--              a manual procedure:
--
--              1. Create a new ENUM without ROLE_OWNER
--              2. ALTER each column using the old ENUM
--              3. DROP the old ENUM
--              4. RENAME the new ENUM
--
--              This is intentionally left as a no-op to prevent
--              accidental data corruption. Perform the rollback
--              manually if needed after verifying no rows use ROLE_OWNER.
-- ============================================================

-- No-op: removing an ENUM value in PostgreSQL requires manual intervention.
SELECT 1;
