package infrastructure

const (
	// ── v1 — offset-based ────────────────────────────────────────────────────

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

	queryListAll = `
		SELECT id, COALESCE(tenant_id::text,''), COALESCE(actor_id::text,''),
		       action, COALESCE(target_type,''), COALESCE(target_id::text,''),
		       metadata, occurred_at
		FROM   audit_logs
		ORDER  BY occurred_at DESC
		LIMIT  $1 OFFSET $2
	`

	// ── v2 — cursor-based (keyset on occurred_at DESC, id ASC) ───────────────
	//
	// Two variants per scope: without cursor (first page) and with cursor
	// (subsequent pages). The row-value comparison
	//   (occurred_at, id) < ($3::timestamptz, $4::uuid)
	// is efficiently served by the composite index:
	//   idx_audit_logs_cursor_tenant  ON audit_logs (tenant_id, occurred_at DESC, id ASC)
	//   idx_audit_logs_cursor_global  ON audit_logs (occurred_at DESC, id ASC)

	// queryListByTenantCursorFirst — first page for a specific tenant
	queryListByTenantCursorFirst = `
		SELECT id, COALESCE(tenant_id::text,''), COALESCE(actor_id::text,''),
		       action, COALESCE(target_type,''), COALESCE(target_id::text,''),
		       metadata, occurred_at
		FROM   audit_logs
		WHERE  tenant_id = $1
		ORDER  BY occurred_at DESC, id ASC
		LIMIT  $2
	`

	// queryListByTenantCursorAfter — next page for a specific tenant
	// $3 = cursor.OccurredAt (timestamptz), $4 = cursor.ID (uuid)
	queryListByTenantCursorAfter = `
		SELECT id, COALESCE(tenant_id::text,''), COALESCE(actor_id::text,''),
		       action, COALESCE(target_type,''), COALESCE(target_id::text,''),
		       metadata, occurred_at
		FROM   audit_logs
		WHERE  tenant_id = $1
		  AND  (occurred_at, id) < ($3::timestamptz, $4::uuid)
		ORDER  BY occurred_at DESC, id ASC
		LIMIT  $2
	`

	// queryListAllCursorFirst — first page across all tenants
	queryListAllCursorFirst = `
		SELECT id, COALESCE(tenant_id::text,''), COALESCE(actor_id::text,''),
		       action, COALESCE(target_type,''), COALESCE(target_id::text,''),
		       metadata, occurred_at
		FROM   audit_logs
		ORDER  BY occurred_at DESC, id ASC
		LIMIT  $1
	`

	// queryListAllCursorAfter — next page across all tenants
	// $2 = cursor.OccurredAt (timestamptz), $3 = cursor.ID (uuid)
	queryListAllCursorAfter = `
		SELECT id, COALESCE(tenant_id::text,''), COALESCE(actor_id::text,''),
		       action, COALESCE(target_type,''), COALESCE(target_id::text,''),
		       metadata, occurred_at
		FROM   audit_logs
		WHERE  (occurred_at, id) < ($2::timestamptz, $3::uuid)
		ORDER  BY occurred_at DESC, id ASC
		LIMIT  $1
	`
)
