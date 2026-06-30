package events

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxDispatcher polls the outbox and dispatches events. It acquires a
// Postgres advisory lock so only one instance dispatches at a time across
// replicas. When WorkerCount > 1, events within a batch are dispatched
// concurrently: each worker owns its own connection + transaction, so
// row-level conflicts are impossible given the advisory lock guard.
//
// Dispatch flow:
//  1. Acquire advisory lock on a dedicated connection (held for full batch).
//  2. Fetch due events in a short tx with SELECT FOR UPDATE SKIP LOCKED.
//  3. Commit the fetch tx — releases row locks, advisory lock stays held.
//  4. Fan out to WorkerCount goroutines; each processes its slice with its
//     own connection + tx (mark published or record failure).
//  5. Release advisory lock + return connection to pool.
type OutboxDispatcher struct {
	pool   *pgxpool.Pool
	logger *slog.Logger

	pollInterval   time.Duration
	batchSize      int
	maxAttempts    int
	advisoryLockID int64
	workerCount    int

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
		workerCount:    cfg.WorkerCount,
		handlersFor:    handlersFor,
	}
}

// Run polls until ctx is cancelled. Transient DB errors are logged and the
// loop continues — the next tick will retry.
func (d *OutboxDispatcher) Run(ctx context.Context) error {
	d.logger.Info("outbox dispatcher starting",
		"poll_interval", d.pollInterval,
		"batch_size", d.batchSize,
		"max_attempts", d.maxAttempts,
		"worker_count", d.workerCount,
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
	// Step 1 — acquire a dedicated connection that holds the advisory lock
	// for the full duration of this batch (fetch + dispatch).
	lockConn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire lock conn: %w", err)
	}
	defer lockConn.Release()

	var acquired bool
	if err := lockConn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, d.advisoryLockID).Scan(&acquired); err != nil {
		return fmt.Errorf("try advisory lock: %w", err)
	}
	if !acquired {
		return nil // another instance owns the lock — skip quietly
	}
	defer func() {
		if _, err := lockConn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, d.advisoryLockID); err != nil {
			d.logger.Warn("outbox dispatcher: advisory unlock failed", "error", err)
		}
	}()

	// Step 2 — fetch the batch (short tx, commits immediately after scan).
	batch, err := d.fetchBatch(ctx, lockConn)
	if err != nil {
		return err
	}
	if len(batch) == 0 {
		return nil
	}

	// Step 3 — dispatch in parallel via worker pool.
	if d.workerCount <= 1 {
		// Serial path — avoids goroutine overhead for WorkerCount=1.
		for _, ev := range batch {
			if err := d.processOne(ctx, ev); err != nil {
				d.logger.Error("outbox dispatcher: process error",
					"event_id", ev.id, "event_type", ev.event.Type, "error", err)
			}
		}
		return nil
	}

	// Parallel path — split batch into workerCount slices.
	work := make(chan pendingEvent, len(batch))
	for _, ev := range batch {
		work <- ev
	}
	close(work)

	var wg sync.WaitGroup
	wg.Add(d.workerCount)
	for range d.workerCount {
		go func() {
			defer wg.Done()
			for ev := range work {
				if err := d.processOne(ctx, ev); err != nil {
					d.logger.Error("outbox dispatcher: process error",
						"event_id", ev.id, "event_type", ev.event.Type, "error", err)
				}
			}
		}()
	}
	wg.Wait()
	return nil
}

// fetchBatch runs a SELECT FOR UPDATE SKIP LOCKED in its own short transaction
// on lockConn (which holds the advisory lock). It commits immediately so row
// locks are released before the (potentially slow) handler execution begins.
func (d *OutboxDispatcher) fetchBatch(ctx context.Context, lockConn *pgxpool.Conn) ([]pendingEvent, error) {
	tx, err := lockConn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin fetch tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, selectDueEventsSQL, d.maxAttempts, d.batchSize)
	if err != nil {
		return nil, fmt.Errorf("select due events: %w", err)
	}

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
			rows.Close()
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
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iter: %w", err)
	}

	// Commit releases the row locks. Advisory lock stays on lockConn.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit fetch tx: %w", err)
	}
	return batch, nil
}

type pendingEvent struct {
	id       string
	event    Event
	attempts int
}

// processOne acquires a fresh connection from the pool, runs the handlers,
// and commits the status update in its own transaction. Each worker calls
// this independently so failures are isolated per-event.
func (d *OutboxDispatcher) processOne(ctx context.Context, pe pendingEvent) error {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire worker conn: %w", err)
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin worker tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := d.dispatchOne(ctx, tx, pe); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (d *OutboxDispatcher) dispatchOne(ctx context.Context, tx pgx.Tx, pe pendingEvent) error {
	handlers := d.handlersFor(pe.event.Type)
	if len(handlers) == 0 {
		// No handlers registered — mark published to avoid accumulation.
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
// up to a hard cap of 5 minutes (RFC-0015 v1.1).
// Exponent is capped at 9 so 1<<9 = 512s → clamped to 300s = 5min.
func backoff(attempts int) time.Duration {
	const maxBackoff = 5 * time.Minute
	exp := attempts
	if exp > 9 {
		exp = 9
	}
	d := time.Duration(1<<exp) * time.Second
	if d > maxBackoff {
		return maxBackoff
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
