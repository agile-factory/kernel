-- platform/white_label/migrations/0001_create_brand_settings.up.sql
-- Per-tenant branding configuration (RFC-0035).

CREATE TABLE brand_settings (
    tenant_id                  UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    app_name                   VARCHAR(50)  NOT NULL DEFAULT '',
    logo_url                   VARCHAR(500),
    favicon_url                VARCHAR(500),
    primary_color              VARCHAR(7)   NOT NULL DEFAULT '#000000',
    secondary_color            VARCHAR(7)   NOT NULL DEFAULT '#FFFFFF',
    accent_color               VARCHAR(7)   NOT NULL DEFAULT '#0066FF',
    font_family                VARCHAR(20)  NOT NULL DEFAULT 'inter'
                                   CHECK (font_family IN ('inter','roboto','lato','custom')),
    custom_font_url            VARCHAR(500),
    custom_domain              VARCHAR(255),
    custom_domain_status       VARCHAR(20)  NOT NULL DEFAULT 'NOT_SET'
                                   CHECK (custom_domain_status IN ('NOT_SET','PENDING','TIMEOUT','VERIFIED','FAILED')),
    custom_domain_token        VARCHAR(100),
    custom_domain_verified_at  TIMESTAMPTZ,
    email_footer_html          TEXT,
    email_logo_url             VARCHAR(500),
    terms_url                  VARCHAR(500),
    privacy_url                VARCHAR(500),
    support_email              VARCHAR(255),
    support_phone              VARCHAR(20),
    updated_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Fast lookup by verified custom domain for portal routing.
CREATE UNIQUE INDEX idx_brand_custom_domain
    ON brand_settings(custom_domain)
    WHERE custom_domain IS NOT NULL AND custom_domain_status = 'VERIFIED';
