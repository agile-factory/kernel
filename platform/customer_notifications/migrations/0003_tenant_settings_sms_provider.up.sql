-- platform/customer_notifications migration 0003 — SMS provider + email sender settings (RFC-0036).

ALTER TABLE tenant_settings
    ADD COLUMN IF NOT EXISTS sms_provider    VARCHAR(30) NOT NULL DEFAULT 'NOOP'
                                 CHECK (sms_provider IN ('ORANGE','TWILIO','AFRICAS_TALKING','NOOP')),
    ADD COLUMN IF NOT EXISTS sms_sender_name VARCHAR(11),
    ADD COLUMN IF NOT EXISTS email_from_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS email_from_addr VARCHAR(255);
