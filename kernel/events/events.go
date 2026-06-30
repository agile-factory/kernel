// Package events provides the transactional outbox bus described in
// RFC-0015 v1.1. Use cases publish events inside their business transaction;
// the dispatcher polls event_outbox in a single-writer loop and invokes
// in-process handlers with at-least-once semantics.
//
// Publishing (inside a tx propagated via kernel/txctx):
//
//	err := txctx.ExecInTx(ctx, pool, func(ctx context.Context) error {
//	    if err := repo.Save(ctx, entity); err != nil { return err }
//	    return bus.Publisher().Publish(ctx, events.Event{
//	        Type:    "tenant.created",
//	        Payload: map[string]any{"tenant_id": entity.ID},
//	        Meta:    events.EventMeta{TenantID: entity.ID, OccurredAt: time.Now()},
//	    })
//	})
//
// Subscribing (typically wired in main.go before bus.Run):
//
//	bus.Subscribe("tenant.created", func(ctx context.Context, ev events.Event) error {
//	    return auditConsumer.Handle(ctx, ev)
//	})
//
// Running the dispatcher (a goroutine, until ctx cancellation):
//
//	go func() { _ = bus.Run(ctx) }()
package events

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the SQL files this package owns (event_outbox +
// processed_events). Composed via kernel/migrator alongside platform FSs.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Event is the unit published to the outbox.
// ID is populated by the dispatcher (outbox row UUID) before handlers are
// invoked; it is empty when the event is first constructed by a publisher.
// Handlers may use it as a deduplication key.
type Event struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
	Meta    EventMeta      `json:"meta"`
}

// EventMeta captures tenant scope + tracing metadata for an Event.
type EventMeta struct {
	TenantID   string    `json:"tenant_id,omitempty"`
	ActorID    string    `json:"actor_id,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
	TraceID    string    `json:"trace_id,omitempty"`
}

// Publisher is implemented by OutboxPublisher and is the surface other
// platforms depend on. Publishing requires a transaction installed on ctx
// via kernel/txctx — Publish returns txctx.ErrNoTx otherwise.
type Publisher interface {
	Publish(ctx context.Context, event Event) error
}

// EventHandler is the in-process callback signature. Handlers must be
// idempotent: a handler returning an error will be retried with exponential
// backoff up to MaxAttempts.
type EventHandler func(ctx context.Context, event Event) error

// Subscriber registers handlers by event type. Implemented by Bus.
type Subscriber interface {
	Subscribe(eventType string, handler EventHandler)
}

// Config captures the dependencies + tunables of the events Bus.
type Config struct {
	Pool   *pgxpool.Pool
	Logger *slog.Logger

	// Tunables (zero = applyDefaults)
	PollInterval   time.Duration // default 1s
	BatchSize      int           // default 100
	MaxAttempts    int           // default 10 (per RFC-0015 v1.1)
	AdvisoryLockID int64         // default = lockKeyOutboxDispatcher
	WorkerCount    int           // parallel dispatch workers; default 1 (serial)
}

// default advisory lock key — ASCII "OUTBOXKL" packed as an int64. Stable so
// every kernel binary uses the same lock across deploys; consumers that need
// multiple independent dispatchers can override via Config.
const lockKeyOutboxDispatcher int64 = 0x4F5554424F584B4C

func (c *Config) applyDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = time.Second
	}
	if c.BatchSize == 0 {
		c.BatchSize = 100
	}
	if c.MaxAttempts == 0 {
		c.MaxAttempts = 10
	}
	if c.AdvisoryLockID == 0 {
		c.AdvisoryLockID = lockKeyOutboxDispatcher
	}
	if c.WorkerCount <= 0 {
		c.WorkerCount = 1
	}
}

func (c *Config) validate() error {
	if c.Pool == nil {
		return errors.New("events.Config: Pool is required")
	}
	if c.Logger == nil {
		return errors.New("events.Config: Logger is required")
	}
	return nil
}

// Bus is the public façade: it bundles a Publisher (write side) and a
// dispatcher (read side) into a single, life-cycle-managed unit. Call Run
// in a goroutine and Subscribe before Run for handlers to receive events
// from the first batch.
type Bus struct {
	cfg        Config
	publisher  *OutboxPublisher
	dispatcher *OutboxDispatcher

	mu       sync.RWMutex
	handlers map[string][]EventHandler
}

// New builds the Bus. It does not open any background work — call Run to
// start the dispatcher loop.
func New(cfg Config) (*Bus, error) {
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	b := &Bus{
		cfg:      cfg,
		handlers: make(map[string][]EventHandler),
	}
	b.publisher = newOutboxPublisher(cfg.Pool, cfg.Logger)
	b.dispatcher = newOutboxDispatcher(cfg, b.handlersFor)
	return b, nil
}

// Publisher returns the Publisher contract for injection into other modules'
// Config (Module pattern, RFC-0038).
func (b *Bus) Publisher() Publisher { return b.publisher }

// Subscribe registers an in-process handler. Multiple handlers may be
// registered for the same eventType; they are invoked in registration order
// and the event is only marked published when all of them succeed.
func (b *Bus) Subscribe(eventType string, handler EventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Run blocks until ctx is cancelled, polling the outbox and invoking
// handlers. Returns ctx.Err() on cancellation.
func (b *Bus) Run(ctx context.Context) error {
	return b.dispatcher.Run(ctx)
}

// handlersFor returns a snapshot of the handlers registered for eventType.
// The snapshot avoids holding the read lock while running handlers (which
// may take an arbitrary amount of time).
func (b *Bus) handlersFor(eventType string) []EventHandler {
	b.mu.RLock()
	defer b.mu.RUnlock()
	src := b.handlers[eventType]
	if len(src) == 0 {
		return nil
	}
	out := make([]EventHandler, len(src))
	copy(out, src)
	return out
}
