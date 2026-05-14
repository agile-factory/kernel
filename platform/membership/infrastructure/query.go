package infrastructure

const (
	// queryInsertMembership inserts a new membership row.
	queryInsertMembership = `
		INSERT INTO memberships (id, user_id, tenant_id, role, status, invited_by, joined_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	// queryGetMembershipByID retrieves a membership by primary key.
	queryGetMembershipByID = `
		SELECT id, user_id, tenant_id, role, status,
		       COALESCE(invited_by::text, '') AS invited_by,
		       joined_at, created_at, updated_at
		FROM memberships
		WHERE id = $1
	`

	// queryGetMembershipByUserAndTenant retrieves the unique membership for a user/tenant pair.
	queryGetMembershipByUserAndTenant = `
		SELECT id, user_id, tenant_id, role, status,
		       COALESCE(invited_by::text, '') AS invited_by,
		       joined_at, created_at, updated_at
		FROM memberships
		WHERE user_id = $1 AND tenant_id = $2
	`

	// queryListMembershipsByTenant returns all memberships for a tenant.
	queryListMembershipsByTenant = `
		SELECT id, user_id, tenant_id, role, status,
		       COALESCE(invited_by::text, '') AS invited_by,
		       joined_at, created_at, updated_at
		FROM memberships
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	// queryListMembershipsByUser returns all memberships a user holds.
	queryListMembershipsByUser = `
		SELECT id, user_id, tenant_id, role, status,
		       COALESCE(invited_by::text, '') AS invited_by,
		       joined_at, created_at, updated_at
		FROM memberships
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	// queryUpdateMembershipStatus updates status and optionally joined_at.
	// joined_at is only set when status = 'ACTIVE' (Accept flow); COALESCE keeps existing value on NULL.
	queryUpdateMembershipStatus = `
		UPDATE memberships
		SET    status     = $2,
		       joined_at  = COALESCE($3, joined_at),
		       updated_at = NOW()
		WHERE  id = $1
	`
)
