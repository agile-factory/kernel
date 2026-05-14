-- ============================================================
-- Migration  : 0002_create_processed_events (UP)
-- Description: Per-consumer dedup table required by RFC-0015 v1.1.
--              Each consumer inserts (consumer, event_id) before
--              applying side effects; UNIQUE conflict means
--              "already processed, skip".
-- ============================================================

CREATE TABLE IF NOT EXISTS processed_events (
    consumer     VARCHAR(100) NOT NULL,
    event_id     UUID         NOT NULL,
    processed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer, event_id)
);
