-- ============================================================
-- Migration  : 0006_hash_email_verification_tokens (DOWN)
-- Description: Hashes cannot be reversed. Pending tokens are
--              dropped (users can request a new verification
--              email); consumed rows are kept for history.
-- ============================================================

DELETE FROM email_verification_tokens WHERE consumed_at IS NULL;
