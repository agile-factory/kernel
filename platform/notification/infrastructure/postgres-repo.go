package infrastructure

import (
	"context"
	"encoding/json"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/notification/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresNotificationLogRepository persists NotificationLog rows.
type PostgresNotificationLogRepository struct {
	db *pgxpool.Pool
}

func NewPostgresNotificationLogRepository(db *pgxpool.Pool) *PostgresNotificationLogRepository {
	return &PostgresNotificationLogRepository{db: db}
}

func (r *PostgresNotificationLogRepository) Save(ctx context.Context, log *domain.NotificationLog) error {
	var tenantID, recipientID any
	if log.TenantID != "" {
		tenantID = log.TenantID
	}
	if log.RecipientID != "" {
		recipientID = log.RecipientID
	}
	payload, err := json.Marshal(log.Payload)
	if err != nil {
		return err
	}
	_, err = txctx.DB(ctx, r.db).Exec(ctx, queryInsertNotificationLog,
		log.ID,
		tenantID,
		recipientID,
		string(log.EventType),
		string(log.Channel),
		string(log.Status),
		payload,
		log.CreatedAt,
	)
	return err
}

func (r *PostgresNotificationLogRepository) Update(ctx context.Context, log *domain.NotificationLog) error {
	var externalID, errMsg any
	if log.ExternalID != "" {
		externalID = log.ExternalID
	}
	if log.Error != "" {
		errMsg = log.Error
	}
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryUpdateNotificationLog,
		log.ID,
		string(log.Status),
		externalID,
		log.SentAt,
		errMsg,
	)
	return err
}

func (r *PostgresNotificationLogRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.NotificationLog, error) {
	rows, err := txctx.DB(ctx, r.db).Query(ctx, queryListByTenant, tenantID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.NotificationLog
	for rows.Next() {
		var (
			log         domain.NotificationLog
			eventType   string
			channel     string
			status      string
			payloadRaw  []byte
		)
		if err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.RecipientID,
			&eventType,
			&channel,
			&status,
			&log.ExternalID,
			&payloadRaw,
			&log.SentAt,
			&log.Error,
			&log.CreatedAt,
		); err != nil {
			return nil, err
		}
		log.EventType = domain.EventType(eventType)
		log.Channel = domain.Channel(channel)
		log.Status = domain.Status(status)
		if len(payloadRaw) > 0 {
			payload := map[string]any{}
			_ = json.Unmarshal(payloadRaw, &payload)
			log.Payload = payload
		}
		out = append(out, &log)
	}
	return out, rows.Err()
}
