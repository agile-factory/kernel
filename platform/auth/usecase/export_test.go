package usecase

import "context"

// passThroughTx runs fn directly, without a database transaction.
func passThroughTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

// WithoutTx makes uc run its transactional section without Postgres.
func (uc *AcceptInviteUseCase) WithoutTx() *AcceptInviteUseCase {
	uc.runInTx = passThroughTx
	return uc
}

// WithoutTx makes uc run its transactional section without Postgres.
func (uc *RefreshTokenUseCase) WithoutTx() *RefreshTokenUseCase {
	uc.runInTx = passThroughTx
	return uc
}
