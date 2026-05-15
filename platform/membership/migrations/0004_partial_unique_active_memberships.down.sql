-- ============================================================
-- Migration  : 0004_partial_unique_active_memberships (DOWN)
-- ============================================================

DROP INDEX IF EXISTS uq_memberships_user_tenant_active;

-- Restore the original plain UNIQUE constraint. Note: this rollback
-- will fail if duplicate (user_id, tenant_id) rows exist across
-- REVOKED + ACTIVE states (which is exactly the scenario this migration
-- enables). Operators should reconcile data before rolling back.
ALTER TABLE memberships
    ADD CONSTRAINT uq_memberships_user_tenant UNIQUE (user_id, tenant_id);
