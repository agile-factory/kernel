package usecase

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerResetPasswordUseCase consumes a single-use reset token and
// replaces the end-customer's password.
type CustomerResetPasswordUseCase struct {
	db        *pgxpool.Pool
	credRepo  domain.CredentialRepository
	resetRepo domain.CustomerResetTokenRepository
	hasher    Hasher
	pub       events.Publisher
}

// NewCustomerResetPasswordUseCase wires the use case.
func NewCustomerResetPasswordUseCase(
	db *pgxpool.Pool,
	credRepo domain.CredentialRepository,
	resetRepo domain.CustomerResetTokenRepository,
	hasher Hasher,
	pub events.Publisher,
) *CustomerResetPasswordUseCase {
	return &CustomerResetPasswordUseCase{db: db, credRepo: credRepo, resetRepo: resetRepo, hasher: hasher, pub: pub}
}

// CustomerResetPasswordInput holds the reset parameters.
type CustomerResetPasswordInput struct {
	ResetToken  string
	NewPassword string
}

// Execute validates the token, hashes the new password, and persists the change.
// Returns ErrPasswordTooShort, ErrTokenNotFound, ErrTokenConsumed, or ErrTokenExpired.
func (uc *CustomerResetPasswordUseCase) Execute(ctx context.Context, in CustomerResetPasswordInput) error {
	if len(in.NewPassword) < 12 {
		return domain.ErrPasswordTooShort
	}

	now := time.Now()
	stored, err := uc.resetRepo.FindByHash(ctx, hashToken(in.ResetToken))
	if err != nil {
		return domain.ErrTokenNotFound
	}

	if stored.ConsumedAt != nil {
		return domain.ErrTokenConsumed
	}
	if !stored.ExpiresAt.After(now) {
		return domain.ErrTokenExpired
	}

	return execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		hash, err := uc.hasher.Hash(in.NewPassword)
		if err != nil {
			return err
		}

		cred, err := uc.credRepo.FindByEndCustomerID(ctx, stored.EndCustomerID)
		if err != nil {
			return err
		}
		cred.PasswordHash = hash
		cred.FailedAttempts = 0
		cred.LockedUntil = nil
		cred.UpdatedAt = now

		if err := uc.credRepo.Update(ctx, cred); err != nil {
			return err
		}
		if err := uc.resetRepo.MarkConsumed(ctx, stored.ID, now); err != nil {
			return err
		}
		return uc.pub.Publish(ctx, events.Event{
			Type: "customer_auth.password_reset",
			Payload: map[string]any{
				"end_customer_id": stored.EndCustomerID,
			},
			Meta: events.EventMeta{OccurredAt: now},
		})
	})
}
