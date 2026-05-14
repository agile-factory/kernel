-- ============================================================
-- Migration  : 0003_add_email_verification_to_users (UP)
-- Description: Add email verification + user lifecycle status
--              per RFC-0018 §7. Existing users are retroactively
--              flagged as ACTIVE + verified (no behaviour change).
-- ============================================================

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS status         VARCHAR(20) NOT NULL DEFAULT 'PENDING_EMAIL';

ALTER TABLE users
    ADD CONSTRAINT users_status_check
    CHECK (status IN ('PENDING_EMAIL', 'ACTIVE', 'SUSPENDED', 'DELETED'));

-- Backfill: existing rows are trusted (they predate the verification flow).
UPDATE users
SET email_verified = TRUE,
    status         = 'ACTIVE'
WHERE status = 'PENDING_EMAIL' AND email_verified = FALSE;

CREATE INDEX IF NOT EXISTS idx_users_status ON users (status);
