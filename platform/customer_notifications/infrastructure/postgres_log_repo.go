package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresNotificationLogRepository implements domain.NotificationLogRepository.
type PostgresNotificationLogRepository struct {
	db *pgxpool.Pool
}

// NewPostgresNotificationLogRepository creates a new repository backed by pgxpool.
func NewPostgresNotificationLogRepository(db *pgxpool.Pool) *PostgresNotificationLogRepository {
	return &PostgresNotificationLogRepository{db: db}
}

// Create inserts a new notification log entry.
func (r *PostgresNotificationLogRepository) Create(ctx context.Context, log *domain.CustomerNotificationLog) error {
	row := r.db.QueryRow(ctx, `
		INSERT INTO customer_notification_logs
			(tenant_id, end_customer_id, channel, recipient, template_key, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`, log.TenantID, nullableStr(log.EndCustomerID), string(log.Channel), log.Recipient, log.TemplateKey, string(log.Status))
	return row.Scan(&log.ID, &log.CreatedAt)
}

// UpdateStatus updates the delivery outcome.
func (r *PostgresNotificationLogRepository) UpdateStatus(ctx context.Context, id string, status domain.DeliveryStatus, externalID, failureReason string, sentAt *time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE customer_notification_logs
		SET status = $2, external_id = $3, failure_reason = $4, sent_at = $5
		WHERE id = $1
	`, id, string(status), nullableStr(externalID), nullableStr(failureReason), sentAt)
	return err
}

// List returns a cursor-paginated list of notification logs.
// Cursor format: "{created_at RFC3339Nano}|{id}" — encodes the last row of the previous page.
// Ordering: created_at DESC, id DESC (consistent with cursor).
func (r *PostgresNotificationLogRepository) List(ctx context.Context, opts domain.ListLogsOptions) ([]*domain.CustomerNotificationLog, string, error) {
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var (
		clauses []string
		args    []any
		n       = 1
	)
	addArg := func(v any) string {
		args = append(args, v)
		s := fmt.Sprintf("$%d", n)
		n++
		return s
	}

	clauses = append(clauses, "tenant_id = "+addArg(opts.TenantID))
	if opts.EndCustomerID != "" {
		clauses = append(clauses, "end_customer_id = "+addArg(opts.EndCustomerID))
	}
	if opts.Channel != "" {
		clauses = append(clauses, "channel = "+addArg(string(opts.Channel)))
	}
	if opts.Status != "" {
		clauses = append(clauses, "status = "+addArg(string(opts.Status)))
	}
	if opts.TemplateKey != "" {
		clauses = append(clauses, "template_key = "+addArg(opts.TemplateKey))
	}
	if opts.Cursor != "" {
		// Cursor is "{created_at}|{id}" — decode into composite comparison.
		parts := strings.SplitN(opts.Cursor, "|", 2)
		if len(parts) == 2 {
			tsArg := addArg(parts[0])
			idArg := addArg(parts[1])
			clauses = append(clauses, "(created_at, id) < ("+tsArg+"::timestamptz, "+idArg+")")
		}
	}

	query := `
		SELECT id, tenant_id, coalesce(end_customer_id::text, ''), channel, recipient,
		       template_key, status, coalesce(external_id, ''), coalesce(failure_reason, ''),
		       sent_at, created_at
		FROM customer_notification_logs
		WHERE ` + strings.Join(clauses, " AND ") + `
		ORDER BY created_at DESC, id DESC
		LIMIT ` + addArg(limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var logs []*domain.CustomerNotificationLog
	for rows.Next() {
		l := &domain.CustomerNotificationLog{}
		var ch, status string
		var sentAt *time.Time
		if err := rows.Scan(
			&l.ID, &l.TenantID, &l.EndCustomerID, &ch, &l.Recipient,
			&l.TemplateKey, &status, &l.ExternalID, &l.FailureReason,
			&sentAt, &l.CreatedAt,
		); err != nil {
			return nil, "", err
		}
		l.Channel = domain.Channel(ch)
		l.Status = domain.DeliveryStatus(status)
		l.SentAt = sentAt
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(logs) > limit {
		last := logs[limit]
		nextCursor = last.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + last.ID
		logs = logs[:limit]
	}
	return logs, nextCursor, nil
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
