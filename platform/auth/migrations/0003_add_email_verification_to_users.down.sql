-- ============================================================
-- Migration  : 0003_add_email_verification_to_users (DOWN)
-- ============================================================

DROP INDEX IF EXISTS idx_users_status;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_status_check;
ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
