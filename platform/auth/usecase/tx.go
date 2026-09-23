package usecase

import (
	"context"

	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PasswordHasherVerifier is implemented by hashers that can both hash a new
// password and check a candidate against a stored hash (BcryptHasher does).
type PasswordHasherVerifier interface {
	PasswordHasher
	PasswordVerifier
}

// txRunner runs fn inside a transaction installed on ctx. It is a field on
// the use cases (rather than a direct txctx.ExecInTx call) so unit tests can
// swap in a pass-through runner and exercise the logic without Postgres.
type txRunner func(ctx context.Context, fn func(ctx context.Context) error) error

func poolTxRunner(pool *pgxpool.Pool) txRunner {
	return func(ctx context.Context, fn func(ctx context.Context) error) error {
		return txctx.ExecInTx(ctx, pool, fn)
	}
}
