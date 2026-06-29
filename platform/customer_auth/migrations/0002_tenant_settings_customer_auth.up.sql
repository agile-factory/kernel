-- platform/customer_auth migration 0002 — customer auth settings in tenant_settings (RFC-0034).
-- Added via customer_auth/migrations (not auth/migrations — per ADR-0006).

ALTER TABLE tenant_settings
    ADD COLUMN IF NOT EXISTS customer_open_registration BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS customer_token_ttl         INTERVAL NOT NULL DEFAULT '1 hour',
    ADD COLUMN IF NOT EXISTS customer_refresh_ttl       INTERVAL NOT NULL DEFAULT '30 days',
    ADD COLUMN IF NOT EXISTS customer_otp_channel       VARCHAR(10) NOT NULL DEFAULT 'EMAIL'
                                 CHECK (customer_otp_channel IN ('EMAIL', 'SMS'));
