-- ============================================================
-- Migration  : 0001_create_event_outbox (UP)
-- Description: Transactional outbox per RFC-0015 v1.1. Events
--              are inserted in the same tx as the business
--              mutation (at-least-once guarantee on commit);
--              the dispatcher polls + delivers + marks
--              published_at.
-- ============================================================

CREATE TABLE IF NOT EXISTS event_outbox (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      VARCHAR(100) NOT NULL,
    payload         JSONB        NOT NULL,
    meta            JSONB        NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at    TIMESTAMPTZ,
    attempts        INT          NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_error      TEXT
);

-- Hot path: dispatcher scan over events ready to be tried again.
-- The partial index keeps the index small (only unpublished rows).
CREATE INDEX IF NOT EXISTS idx_event_outbox_ready
    ON event_outbox (next_attempt_at)
    WHERE published_at IS NULL;

-- Retention sweep / quarantine listing scans over published rows.
CREATE INDEX IF NOT EXISTS idx_event_outbox_retention
    ON event_outbox (created_at)
    WHERE published_at IS NOT NULL;

-- Filter for quarantine listing (events that exhausted retries).
CREATE INDEX IF NOT EXISTS idx_event_outbox_attempts
    ON event_outbox (attempts)
    WHERE published_at IS NULL;
