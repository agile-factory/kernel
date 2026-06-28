package usecase

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerForgotPasswordUseCase generates a single-use reset token and
// publishes an event (handled by customer_notifications to send OTP/email).
//
// Email enumeration resistance: when the email does not exist the use case
// returns nil silently — the HTTP handler returns 204 either way.
//
// OTP rate-limit: a second request within 60 seconds returns ErrOTPRateLimit.
type CustomerForgotPasswordUseCase struct {
	db        *pgxpool.Pool
	credRepo  domain.CredentialRepository
	resetRepo domain.CustomerResetTokenRepository
	pub       events.Publisher
}

// NewCustomerForgotPasswordUseCase wires the use case.
func NewCustomerForgotPasswordUseCase(
	db *pgxpool.Pool,
	credRepo domain.CredentialRepository,
	resetRepo domain.CustomerResetTokenRepository,
	pub events.Publisher,
) *CustomerForgotPasswordUseCase {
	return &CustomerForgotPasswordUseCase{db: db, credRepo: credRepo, resetRepo: resetRepo, pub: pub}
}

// CustomerForgotPasswordInput holds the request parameters.
type CustomerForgotPasswordInput struct {
	Email    string
	TenantID string
}

// Execute generates a reset token for the given email.
// Returns nil silently when the email is unknown (enumeration resistance).
// Returns ErrOTPRateLimit when a token was already issued within 60 seconds.
func (uc *CustomerForgotPasswordUseCase) Execute(ctx context.Context, in CustomerForgotPasswordInput) error {
	now := time.Now()

	cred, err := uc.credRepo.FindByEmail(ctx, in.Email, in.TenantID)
	if err != nil {
		// Email not found — return nil to avoid enumeration.
		return nil
	}

	// OTP 60-second rate-limit (Lucas amendment).
	since := now.Add(-60 * time.Second)
	has, err := uc.resetRepo.HasRecentToken(ctx, cred.EndCustomerID, since)
	if err != nil {
		return err
	}
	if has {
		return domain.ErrOTPRateLimit
	}

	return execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		rawToken := generateOpaqueToken()
		reset := &domain.CustomerResetToken{
			EndCustomerID: cred.EndCustomerID,
			TokenHash:     hashToken(rawToken),
			ExpiresAt:     now.Add(1 * time.Hour),
		}
		if err := uc.resetRepo.Create(ctx, reset); err != nil {
			return err
		}
		return uc.pub.Publish(ctx, events.Event{
			Type: "customer_auth.password_reset_requested",
			Payload: map[string]any{
				"end_customer_id": cred.EndCustomerID,
				"tenant_id":       cred.TenantID,
				"reset_token":     rawToken,
			},
			Meta: events.EventMeta{TenantID: cred.TenantID, OccurredAt: now},
		})
	})
}
