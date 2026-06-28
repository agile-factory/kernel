-- ============================================================
-- Migration  : 0001_create_end_customers (UP)
-- Description: Create the end_customers table for B2B2C — RFC-0033
-- ============================================================

CREATE TABLE end_customers (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id  VARCHAR(255),
    email        VARCHAR(255),
    phone        VARCHAR(20),
    first_name   VARCHAR(100) NOT NULL,
    last_name    VARCHAR(100) NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                     CHECK (status IN ('ACTIVE','SUSPENDED','DELETED')),
    metadata     JSONB        NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,

    CONSTRAINT uq_end_customer_email_tenant
        UNIQUE NULLS NOT DISTINCT (tenant_id, email),
    CONSTRAINT uq_end_customer_external_id_tenant
        UNIQUE NULLS NOT DISTINCT (tenant_id, external_id)
);

CREATE INDEX idx_end_customers_tenant_status
    ON end_customers(tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_end_customers_cursor
    ON end_customers(tenant_id, created_at DESC, id)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_end_customers_search
    ON end_customers
    USING gin(to_tsvector('simple',
        coalesce(first_name,'') || ' ' ||
        coalesce(last_name,'')  || ' ' ||
        coalesce(email,'')))
    WHERE deleted_at IS NULL;
