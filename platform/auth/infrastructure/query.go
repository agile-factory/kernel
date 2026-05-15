package infrastructure

const (
	queryInsertUser = `
		INSERT INTO users (id, email, password, role, email_verified, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	queryGetUserByEmail = `
		SELECT id, email, password, role, email_verified, status, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	queryGetUserByID = `
		SELECT id, email, password, role, email_verified, status, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	// queryMarkUserEmailVerified flips email_verified=TRUE and, when the user
	// is still in PENDING_EMAIL, transitions them to ACTIVE. Idempotent.
	queryMarkUserEmailVerified = `
		UPDATE users
		SET email_verified = TRUE,
		    status = CASE WHEN status = 'PENDING_EMAIL' THEN 'ACTIVE' ELSE status END,
		    updated_at = NOW()
		WHERE id = $1
	`

	queryInsertRefreshToken = `
		INSERT INTO refresh_tokens (id, user_id, tenant_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	queryGetRefreshTokenByHash = `
		SELECT id, user_id, tenant_id, token_hash, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	queryRevokeRefreshToken = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
	`

	queryInsertEmailVerificationToken = `
		INSERT INTO email_verification_tokens (id, user_id, token, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	queryGetEmailVerificationTokenByToken = `
		SELECT id, user_id, token, expires_at, consumed_at, created_at
		FROM email_verification_tokens
		WHERE token = $1
	`

	queryConsumeEmailVerificationToken = `
		UPDATE email_verification_tokens
		SET consumed_at = NOW()
		WHERE id = $1 AND consumed_at IS NULL
	`
)
