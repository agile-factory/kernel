ALTER TABLE tenant_settings
    DROP COLUMN IF EXISTS customer_otp_channel,
    DROP COLUMN IF EXISTS customer_refresh_ttl,
    DROP COLUMN IF EXISTS customer_token_ttl,
    DROP COLUMN IF EXISTS customer_open_registration;
