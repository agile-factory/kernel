package usecase

import (
	"context"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// execWithOptionalTx runs fn inside a transaction when db is non-nil.
// When db is nil (unit-test mode with in-memory stubs) fn is called directly
// so that tests never need a live Postgres connection.
func execWithOptionalTx(ctx context.Context, db *pgxpool.Pool, fn func(context.Context) error) error {
	if db == nil {
		return fn(ctx)
	}
	return txctx.ExecInTx(ctx, db, fn)
}
