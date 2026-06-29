package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxDispatcher is the single-writer poll loop: it acquires a Postgres
// advisory lock (so only one instance dispatches at a time), reads a batch
// of due events via SELECT FOR UPDATE SKIP LOCKED, invokes the registered
// handlers, then marks each event published (success) or schedules a retry
// with exponential backoff (failure). After MaxAttempts failures the event
// stays in the table without published_at — quarantine, surfaced via
// the admin query and an alert at the operator's discretion.
type OutboxDispatcher struct {
	pool   *pgxpool.Pool
	logger *slog.Logger

	pollInterval   time.Duration
	batchSize      int
	maxAttempts    int
	advisoryLockID int64

	handlersFor func(eventType string) []EventHandler
}

func newOutboxDispatcher(cfg Config, handlersFor func(string) []EventHandler) *OutboxDispatcher {
	return &OutboxDispatcher{
		pool:           cfg.Pool,
		logger:         cfg.Logger,
		pollInterval:   cfg.PollInterval,
		batchSize:      cfg.BatchSize,
		maxAttempts:    cfg.MaxAttempts,
		advisoryLockID: cfg.AdvisoryLockID,
		handlersFor:    handlersFor,
	}
}

// Run polls until ctx is cancelled. A polling tick that hits a transient
// database error logs and continues — the next tick will retry.
func (d *OutboxDispatcher) Run(ctx context.Context) error {
	d.logger.Info("outbox dispatcher starting",
		"poll_interval", d.pollInterval,
		"batch_size", d.batchSize,
		"max_attempts", d.maxAttempts,
	)
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("outbox dispatcher stopping", "reason", ctx.Err())
			return ctx.Err()
		case <-ticker.C:
			if err := d.dispatchBatch(ctx); err != nil && !errors.Is(err, context.Canceled) {
				d.logger.Error("outbox dispatcher: batch failed", "error", err)
			}
		}
	}
}

const selectDueEventsSQL = `
SELECT id, event_type, payload, meta, attempts
FROM event_outbox
WHERE published_at IS NULL
  AND attempts < $1
  AND next_attempt_at <= NOW()
ORDER BY next_attempt_at ASC, created_at ASC
LIMIT $2
FOR UPDATE SKIP LOCKED
`

func (d *OutboxDispatcher) dispatchBatch(ctx context.Context) error {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire conn: %w", err)
	}
	defer conn.Release()

	// Session-level advisory lock: only one dispatcher is allowed to scan
	// at a time. Released explicitly before conn is returned to the pool
	// (defer order: unlock fires before Release).
	var acquired bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, d.advisoryLockID).Scan(&acquired); err != nil {
		return fmt.Errorf("try advisory lock: %w", err)
	}
	if !acquired {
		// Another instance owns the lock — quiet skip, will retry next tick.
		return nil
	}
	defer func() {
		if _, err := conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, d.advisoryLockID); err != nil {
			d.logger.Warn("outbox dispatcher: advisory unlock failed", "error", err)
		}
	}()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	batch, err := d.fetchBatch(ctx, tx)
	if err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}

	for _, ev := range batch {
		if err := d.dispatchOne(ctx, tx, ev); err != nil {
			// dispatchOne already records the failure on the row; surfacing
			// the error here would break processing of the remaining batch.
			d.logger.Error("outbox dispatcher: dispatch error",
				"event_id", ev.id, "event_type", ev.event.Type, "error", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

type pendingEvent struct {
	id       string
	event    Event
	attempts int
}

func (d *OutboxDispatcher) fetchBatch(ctx context.Context, tx pgx.Tx) ([]pendingEvent, error) {
	rows, err := tx.Query(ctx, selectDueEventsSQL, d.maxAttempts, d.batchSize)
	if err != nil {
		return nil, fmt.Errorf("select due events: %w", err)
	}
	defer rows.Close()

	var batch []pendingEvent
	for rows.Next() {
		var (
			id          string
			eventType   string
			payloadRaw  []byte
			metaRaw     []byte
			attempts    int
			payloadMap  = map[string]any{}
			metaPayload EventMeta
		)
		if err := rows.Scan(&id, &eventType, &payloadRaw, &metaRaw, &attempts); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		if len(payloadRaw) > 0 {
			if err := json.Unmarshal(payloadRaw, &payloadMap); err != nil {
				d.logger.Warn("outbox: payload unmarshal failed", "event_id", id, "error", err)
			}
		}
		if len(metaRaw) > 0 {
			if err := json.Unmarshal(metaRaw, &metaPayload); err != nil {
				d.logger.Warn("outbox: meta unmarshal failed", "event_id", id, "error", err)
			}
		}
		batch = append(batch, pendingEvent{
			id: id,
			event: Event{
				ID:      id,
				Type:    eventType,
				Payload: payloadMap,
				Meta:    metaPayload,
			},
			attempts: attempts,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iter: %w", err)
	}
	return batch, nil
}

func (d *OutboxDispatcher) dispatchOne(ctx context.Context, tx pgx.Tx, pe pendingEvent) error {
	handlers := d.handlersFor(pe.event.Type)

	// No handlers? Mark published — the event was emitted but nobody cares.
	// Avoids accumulating events forever and lets operators audit volume.
	if len(handlers) == 0 {
		return d.markPublished(ctx, tx, pe.id)
	}

	for _, h := range handlers {
		if err := h(ctx, pe.event); err != nil {
			return d.recordFailure(ctx, tx, pe, err)
		}
	}
	return d.markPublished(ctx, tx, pe.id)
}

const markPublishedSQL = `
UPDATE event_outbox
SET published_at = NOW(),
    attempts = attempts + 1
WHERE id = $1
`

func (d *OutboxDispatcher) markPublished(ctx context.Context, tx pgx.Tx, id string) error {
	if _, err := tx.Exec(ctx, markPublishedSQL, id); err != nil {
		return fmt.Errorf("mark published %s: %w", id, err)
	}
	return nil
}

const recordFailureSQL = `
UPDATE event_outbox
SET attempts = attempts + 1,
    last_error = $1,
    next_attempt_at = $2
WHERE id = $3
`

// backoff returns the delay before the next retry, doubling each attempt
// up to a hard cap (5 minutes per RFC-0015 v1.1).
func backoff(attempts int) time.Duration {
	const cap = 5 * time.Minute
	// 1s · 2^attempts. Cap exponent at 8 (256s) before the explicit ceiling
	// to avoid overflow on absurd attempt counts.
	exp := attempts
	if exp > 8 {
		exp = 8
	}
	d := time.Duration(1<<exp) * time.Second
	if d > cap {
		return cap
	}
	return d
}

func (d *OutboxDispatcher) recordFailure(ctx context.Context, tx pgx.Tx, pe pendingEvent, cause error) error {
	delay := backoff(pe.attempts)
	nextAttempt := time.Now().Add(delay)

	d.logger.Warn("outbox event handler failed",
		"event_id", pe.id,
		"event_type", pe.event.Type,
		"attempts", pe.attempts+1,
		"next_retry_in", delay,
		"error", cause,
	)

	if _, err := tx.Exec(ctx, recordFailureSQL, cause.Error(), nextAttempt, pe.id); err != nil {
		return fmt.Errorf("record failure %s: %w", pe.id, err)
	}
	return nil
}
