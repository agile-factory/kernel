package infrastructure

const (
	queryInsertNotificationLog = `
		INSERT INTO notification_logs
		    (id, tenant_id, recipient_id, event_type, channel, status, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	queryUpdateNotificationLog = `
		UPDATE notification_logs
		SET status      = $2,
		    external_id = $3,
		    sent_at     = $4,
		    error       = $5
		WHERE id = $1
	`

	queryListByTenant = `
		SELECT id,
		       COALESCE(tenant_id::text, '') AS tenant_id,
		       COALESCE(recipient_id::text, '') AS recipient_id,
		       event_type, channel, status,
		       COALESCE(external_id, '') AS external_id,
		       payload, sent_at,
		       COALESCE(error, '') AS error,
		       created_at
		FROM notification_logs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
)
