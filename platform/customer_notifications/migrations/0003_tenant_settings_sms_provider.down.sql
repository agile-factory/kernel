ALTER TABLE tenant_settings
    DROP COLUMN IF EXISTS email_from_addr,
    DROP COLUMN IF EXISTS email_from_name,
    DROP COLUMN IF EXISTS sms_sender_name,
    DROP COLUMN IF EXISTS sms_provider;
