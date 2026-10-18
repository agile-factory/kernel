-- ============================================================
-- Migration  : 0006_hash_email_verification_tokens (UP)
-- Description: Store email verification tokens as their hex SHA-256
--              instead of in clear (same scheme as refresh_tokens).
--              Existing rows are hashed in place, so pending links
--              sent before the upgrade keep working.
-- ============================================================

UPDATE email_verification_tokens
SET token = encode(sha256(convert_to(token, 'UTF8')), 'hex');
