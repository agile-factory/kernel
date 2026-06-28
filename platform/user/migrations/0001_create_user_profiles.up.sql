-- ============================================================
-- Migration  : 0001_create_user_profiles (UP)
-- Description: Create user_profiles table — 1:1 optional with users.
--              Profile is created lazily on first GET request.
-- ============================================================

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id      UUID         PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100),
    avatar_url   TEXT,
    bio          TEXT,
    locale       VARCHAR(10)  NOT NULL DEFAULT 'fr-SN',
    timezone     VARCHAR(50)  NOT NULL DEFAULT 'Africa/Dakar',
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
