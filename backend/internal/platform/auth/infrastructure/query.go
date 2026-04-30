package infrastructure

const (
	queryInsertUser = `
		INSERT INTO users (id, tenant_id, email, password, role, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	queryGetUserByEmail = `
		SELECT id, email, password, role, created_at, updated_at 
		FROM users 
		WHERE email = $1
	`

	queryGetUserByID = `
		SELECT id, email, password, role, created_at, updated_at 
		FROM users 
		WHERE id = $1
	`
)
