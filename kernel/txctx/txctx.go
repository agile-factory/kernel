// Package txctx propagates a database transaction through context.Context so
// that an inner component (e.g. an outbox publisher) can write inside the
// caller's transaction without taking it as an explicit parameter.
//
// Usage:
//
//	err := txctx.ExecInTx(ctx, pool, func(ctx context.Context) error {
//	    // ctx now carries the pgx.Tx — repos can pick it up via DB(ctx)
//	    if err := repo.Save(ctx, entity); err != nil { return err }
//	    return publisher.Publish(ctx, event) // writes to event_outbox in the same tx
//	})
//
// Repos that want to participate transparently call DB(ctx) to obtain a
// DBTX that is either the in-context transaction or the fallback pool.
package txctx

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX is the minimal contract shared by *pgxpool.Pool and pgx.Tx. Repos
// should accept DBTX (or use DB(ctx) below) rather than the concrete pool so
// they can be enlisted into a caller-started transaction.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type txKey struct{}

// ErrNoTx is returned by MustFromContext when no transaction has been
// installed on the context.
var ErrNoTx = errors.New("txctx: no transaction in context")

// WithTx returns a copy of ctx that carries tx.
func WithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// FromContext returns the transaction stored on ctx, or (nil, false) when
// no transaction is present.
func FromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

// MustFromContext returns the transaction stored on ctx. Callers that must
// write inside an existing transaction (e.g. the outbox publisher) use this
// to fail fast when no tx is present rather than silently writing outside.
func MustFromContext(ctx context.Context) (pgx.Tx, error) {
	tx, ok := FromContext(ctx)
	if !ok {
		return nil, ErrNoTx
	}
	return tx, nil
}

// DB returns the in-context transaction if present, otherwise the supplied
// fallback (typically the *pgxpool.Pool). Repos that should work both
// inside and outside a caller-started tx call this with the pool they own.
func DB(ctx context.Context, fallback DBTX) DBTX {
	if tx, ok := FromContext(ctx); ok {
		return tx
	}
	return fallback
}

// ExecInTx begins a transaction on pool, installs it on ctx, runs fn, and
// commits — or rolls back if fn returns an error or panics. Panics are
// re-thrown after rollback.
func ExecInTx(ctx context.Context, pool *pgxpool.Pool, fn func(ctx context.Context) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("txctx: begin: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = fmt.Errorf("%w (rollback failed: %v)", err, rbErr)
			}
		}
	}()

	if err = fn(WithTx(ctx, tx)); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("txctx: commit: %w", err)
	}
	return nil
}
