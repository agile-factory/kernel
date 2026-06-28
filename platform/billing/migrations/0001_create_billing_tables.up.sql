-- Platform Billing tables — RFC-0020 Platform Mode
-- Migration: 0001_create_billing_tables

CREATE TABLE IF NOT EXISTS plans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(50)  NOT NULL UNIQUE,
    price_cents     BIGINT       NOT NULL DEFAULT 0,
    currency        VARCHAR(3)   NOT NULL DEFAULT 'USD',
    interval        VARCHAR(10)  NOT NULL DEFAULT 'MONTHLY'
                        CHECK (interval IN ('MONTHLY','YEARLY')),
    max_users       BIGINT,
    max_storage     BIGINT,
    max_api_calls   BIGINT,
    can_export      BOOLEAN      NOT NULL DEFAULT FALSE,
    can_white_label BOOLEAN      NOT NULL DEFAULT FALSE,
    can_merchant    BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS subscriptions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID UNIQUE NOT NULL REFERENCES tenants(id),
    plan_id                 UUID NOT NULL REFERENCES plans(id),
    status                  VARCHAR(20) NOT NULL DEFAULT 'TRIALING'
                                CHECK (status IN ('TRIALING','ACTIVE','PAST_DUE','CANCELED')),
    current_period_start    TIMESTAMPTZ NOT NULL,
    current_period_end      TIMESTAMPTZ NOT NULL,
    canceled_at             TIMESTAMPTZ,
    payment_method          VARCHAR(20) NOT NULL DEFAULT 'STRIPE',
    external_id             VARCHAR(255),
    mobile_money_ref        VARCHAR(255),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tenant_settings (
    tenant_id           UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    logo_url            VARCHAR(500),
    primary_color       VARCHAR(7),
    locale              VARCHAR(10)  NOT NULL DEFAULT 'en',
    timezone            VARCHAR(50)  NOT NULL DEFAULT 'UTC',
    currency            VARCHAR(3)   NOT NULL DEFAULT 'USD',
    data_region         VARCHAR(10)  NOT NULL DEFAULT 'us'
                            CHECK (data_region IN ('us','eu','apac')),
    payment_region      VARCHAR(20)  NOT NULL DEFAULT 'INTERNATIONAL'
                            CHECK (payment_region IN ('INTERNATIONAL','CEDEAO')),
    merchant_mode       BOOLEAN      NOT NULL DEFAULT FALSE,
    merchant_processor  VARCHAR(50),
    merchant_account_id VARCHAR(255),
    merchant_kyc_status VARCHAR(20)  NOT NULL DEFAULT 'NOT_STARTED',
    custom_domain       VARCHAR(255),
    b2b2c_enabled       BOOLEAN      NOT NULL DEFAULT FALSE,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Idempotency table for inbound webhooks (Stripe, Wave, Orange Money, Free Money)
CREATE TABLE IF NOT EXISTS processed_events (
    event_id     VARCHAR(255) NOT NULL,
    source       VARCHAR(50)  NOT NULL,
    processed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, source)
);

-- Seed free plan (always present)
INSERT INTO plans (name, price_cents, currency, interval)
VALUES ('free', 0, 'USD', 'MONTHLY')
ON CONFLICT (name) DO NOTHING;
