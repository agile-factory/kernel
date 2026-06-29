package infrastructure

const queryFindByUserID = `
SELECT user_id, display_name, avatar_url, bio, locale, timezone, updated_at
FROM user_profiles
WHERE user_id = $1
`

const queryUpsert = `
INSERT INTO user_profiles (user_id, display_name, avatar_url, bio, locale, timezone, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (user_id) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    avatar_url   = EXCLUDED.avatar_url,
    bio          = EXCLUDED.bio,
    locale       = EXCLUDED.locale,
    timezone     = EXCLUDED.timezone,
    updated_at   = EXCLUDED.updated_at
`
