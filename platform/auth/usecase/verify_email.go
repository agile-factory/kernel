package usecase

import (
	"context"
	"time"

	"github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/kernel/txctx"
	authdomain "github.com/AgileFactory/kernel/platform/auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// VerifyEmailUseCase consumes an email-verification token and marks the
// associated user's email as verified. Atomic (txctx) so the token consumption
// and user update commit together or roll back together.
type VerifyEmailUseCase struct {
	pool        *pgxpool.Pool
	users       authdomain.UserRepository
	verifTokens authdomain.EmailVerificationTokenRepository
}

func NewVerifyEmailUseCase(
	pool *pgxpool.Pool,
	users authdomain.UserRepository,
	verifTokens authdomain.EmailVerificationTokenRepository,
) *VerifyEmailUseCase {
	return &VerifyEmailUseCase{
		pool:        pool,
		users:       users,
		verifTokens: verifTokens,
	}
}

// Execute redeems the token. Returns ErrTokenNotFound / ErrTokenExpired /
// ErrTokenAlreadyConsumed for the corresponding failure modes.
func (uc *VerifyEmailUseCase) Execute(ctx context.Context, rawToken string) error {
	return txctx.ExecInTx(ctx, uc.pool, func(ctx context.Context) error {
		evt, err := uc.verifTokens.FindByToken(ctx, rawToken)
		if err != nil {
			return err // ErrTokenNotFound bubbles up unwrapped
		}
		if evt.ConsumedAt != nil {
			return authdomain.ErrTokenAlreadyConsumed
		}
		if time.Now().After(evt.ExpiresAt) {
			return authdomain.ErrTokenExpired
		}

		if err := uc.verifTokens.Consume(ctx, evt.ID); err != nil {
			return errors.WrapInternal(err)
		}
		if err := uc.users.UpdateEmailVerified(ctx, evt.UserID); err != nil {
			return errors.WrapInternal(err)
		}
		return nil
	})
}
