-- ============================================================
-- Migration  : 0004_partial_unique_active_memberships (UP)
-- Description: RFC-0018 §4.3 — allow re-invitation of a member
--              whose previous membership was REVOKED.
--              The plain UNIQUE(user_id, tenant_id) constraint
--              made this impossible (the revoked row blocked a
--              new ACTIVE one). We drop it and replace with a
--              partial index that excludes REVOKED, so a single
--              (user_id, tenant_id) can carry one non-revoked
--              row at any time while keeping the historical
--              REVOKED rows for audit.
-- ============================================================

ALTER TABLE memberships
    DROP CONSTRAINT IF EXISTS uq_memberships_user_tenant;

CREATE UNIQUE INDEX IF NOT EXISTS uq_memberships_user_tenant_active
    ON memberships (user_id, tenant_id)
    WHERE status <> 'REVOKED';
