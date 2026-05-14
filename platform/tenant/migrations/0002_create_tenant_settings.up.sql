-- ============================================================
-- Migration  : 0002_create_tenant_settings (UP)
-- Description: Per RFC-0018 + plan §4.3. One row per tenant.
--              Holds branding, locale, currency, residency,
--              custom-domain and profile flags consumed by
--              white-label / billing / B2B2C surfaces.
--
-- Notes
--   * 1:1 with tenants — enforced by PRIMARY KEY on tenant_id.
--   * payment_region: 'INTERNATIONAL' (Stripe-only) or 'CEDEAO'
--     (Stripe + Wave + Orange Money + Free Money). RFC-0020.
--   * data_region: 'eu' / 'us' / 'apac' for GDPR pinning (RFC-0027).
--   * merchant_mode / b2b2c_enabled gate the B2B2C surfaces.
-- ============================================================

CREATE TABLE IF NOT EXISTS tenant_settings (
    tenant_id         UUID         PRIMARY KEY REFERENCES tenants (id) ON DELETE CASCADE,
    logo_url          TEXT,
    primary_color     VARCHAR(7),                                       -- '#RRGGBB'
    locale            VARCHAR(10)  NOT NULL DEFAULT 'en-US',
    timezone          VARCHAR(64)  NOT NULL DEFAULT 'UTC',
    currency          VARCHAR(3)   NOT NULL DEFAULT 'USD',              -- ISO 4217
    data_region       VARCHAR(8)   NOT NULL DEFAULT 'eu',
    custom_domain     VARCHAR(253),                                     -- RFC 1035 max
    payment_region    VARCHAR(20)  NOT NULL DEFAULT 'INTERNATIONAL',
    merchant_mode     BOOLEAN      NOT NULL DEFAULT FALSE,              -- B2B2C opt-in
    b2b2c_enabled     BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT tenant_settings_data_region_check
        CHECK (data_region IN ('eu', 'us', 'apac')),
    CONSTRAINT tenant_settings_payment_region_check
        CHECK (payment_region IN ('INTERNATIONAL', 'CEDEAO'))
);

-- Custom domain → tenant_id lookup (Host-header routing for B2B2C).
CREATE UNIQUE INDEX IF NOT EXISTS idx_tenant_settings_custom_domain
    ON tenant_settings (custom_domain)
    WHERE custom_domain IS NOT NULL;
