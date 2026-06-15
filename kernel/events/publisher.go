package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxPublisher inserts events into event_outbox in the same transaction
// as the calling use case. If no transaction is installed on ctx via
// kernel/txctx, Publish returns txctx.ErrNoTx — by design, to make the
// at-least-once guarantee explicit (an event published outside a business
// tx could leak ahead of an eventual rollback).
type OutboxPublisher struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func newOutboxPublisher(pool *pgxpool.Pool, logger *slog.Logger) *OutboxPublisher {
	return &OutboxPublisher{pool: pool, logger: logger}
}

const insertEventOutboxSQL = `
INSERT INTO event_outbox (event_type, payload, meta)
VALUES ($1, $2, $3)
`

// Publish inserts the event into event_outbox inside the caller's
// transaction. Caller is expected to have wrapped its operation via
// txctx.ExecInTx (or to have called txctx.WithTx manually).
func (p *OutboxPublisher) Publish(ctx context.Context, event Event) error {
	tx, err := txctx.MustFromContext(ctx)
	if err != nil {
		return fmt.Errorf("outbox publish: %w", err)
	}

	if event.Meta.OccurredAt.IsZero() {
		event.Meta.OccurredAt = time.Now().UTC()
	}

	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("outbox publish: marshal payload: %w", err)
	}
	meta, err := json.Marshal(event.Meta)
	if err != nil {
		return fmt.Errorf("outbox publish: marshal meta: %w", err)
	}

	if _, err := tx.Exec(ctx, insertEventOutboxSQL, event.Type, payload, meta); err != nil {
		return fmt.Errorf("outbox publish: insert: %w", err)
	}
	return nil
}
