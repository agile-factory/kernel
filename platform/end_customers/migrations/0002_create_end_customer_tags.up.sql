-- ============================================================
-- Migration  : 0002_create_end_customer_tags (UP)
-- Description: Create the end_customer_tags table for segmentation — RFC-0033
-- ============================================================

CREATE TABLE end_customer_tags (
    end_customer_id UUID        NOT NULL REFERENCES end_customers(id) ON DELETE CASCADE,
    tag             VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (end_customer_id, tag)
);

CREATE INDEX idx_end_customer_tags_by_tag
    ON end_customer_tags(tag)
    INCLUDE (end_customer_id);
