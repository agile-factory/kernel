package infrastructure

const (
	queryInsertAuditLog = `
		INSERT INTO audit_logs
		      (id, tenant_id, actor_id, action, target_type, target_id, metadata, occurred_at)
		VALUES ($1, NULLIF($2,'')::uuid, NULLIF($3,'')::uuid, $4, $5, NULLIF($6,'')::uuid, $7, $8)
	`

	queryListByTenant = `
		SELECT id, COALESCE(tenant_id::text,''), COALESCE(actor_id::text,''),
		       action, COALESCE(target_type,''), COALESCE(target_id::text,''),
		       metadata, occurred_at
		FROM   audit_logs
		WHERE  tenant_id = $1
		ORDER  BY occurred_at DESC
		LIMIT  $2 OFFSET $3
	`
)
