package usecase

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerLoginUseCase authenticates an end-customer with email + password.
// Email enumeration resistance: any failure before the password check returns
// ErrInvalidCredentials — the same error as a bad password.
type CustomerLoginUseCase struct {
	db          *pgxpool.Pool
	credRepo    domain.CredentialRepository
	refreshRepo domain.CustomerRefreshTokenRepository
	verifier    PasswordVerifier
	jwtSvc      CustomerJWTIssuer
	pub         events.Publisher
	refreshTTL  time.Duration
}

// NewCustomerLoginUseCase wires the use case.
func NewCustomerLoginUseCase(
	db *pgxpool.Pool,
	credRepo domain.CredentialRepository,
	refreshRepo domain.CustomerRefreshTokenRepository,
	verifier PasswordVerifier,
	jwtSvc CustomerJWTIssuer,
	pub events.Publisher,
	refreshTTL time.Duration,
) *CustomerLoginUseCase {
	return &CustomerLoginUseCase{
		db:          db,
		credRepo:    credRepo,
		refreshRepo: refreshRepo,
		verifier:    verifier,
		jwtSvc:      jwtSvc,
		pub:         pub,
		refreshTTL:  refreshTTL,
	}
}

// CustomerLoginInput holds the login parameters.
type CustomerLoginInput struct {
	Email    string
	Password string
	TenantID string
}

// CustomerLoginOutput holds the issued tokens.
type CustomerLoginOutput struct {
	AccessToken  string
	RefreshToken string // opaque bearer value; hash is stored in DB
}

// Execute authenticates the end-customer. On any failure it returns
// ErrInvalidCredentials (email enumeration resistance).
func (uc *CustomerLoginUseCase) Execute(ctx context.Context, in CustomerLoginInput) (CustomerLoginOutput, error) {
	now := time.Now()

	// Email lookup — on not-found return ErrInvalidCredentials, not ErrCredentialNotFound.
	cred, err := uc.credRepo.FindByEmail(ctx, in.Email, in.TenantID)
	if err != nil {
		return CustomerLoginOutput{}, domain.ErrInvalidCredentials
	}

	if cred.IsLocked(now) {
		return CustomerLoginOutput{}, domain.ErrAccountLocked
	}

	// Password check — on mismatch increment failure counter and return ErrInvalidCredentials.
	if verifyErr := uc.verifier.Verify(in.Password, cred.PasswordHash); verifyErr != nil {
		cred.RecordLoginFailure(now)
		_ = execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
			return uc.credRepo.Update(ctx, cred)
		})
		return CustomerLoginOutput{}, domain.ErrInvalidCredentials
	}

	cred.RecordLoginSuccess(now)

	var out CustomerLoginOutput
	err = execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.credRepo.Update(ctx, cred); err != nil {
			return err
		}

		accessToken, err := uc.jwtSvc.GenerateToken(cred.EndCustomerID, cred.TenantID)
		if err != nil {
			return err
		}

		rawRefresh := generateOpaqueToken()
		refresh := &domain.CustomerRefreshToken{
			EndCustomerID: cred.EndCustomerID,
			TenantID:      cred.TenantID,
			TokenHash:     hashToken(rawRefresh),
			ExpiresAt:     now.Add(uc.refreshTTL),
		}
		if err := uc.refreshRepo.Create(ctx, refresh); err != nil {
			return err
		}

		if err := uc.pub.Publish(ctx, events.Event{
			Type: "customer_auth.logged_in",
			Payload: map[string]any{
				"end_customer_id": cred.EndCustomerID,
				"tenant_id":       cred.TenantID,
			},
			Meta: events.EventMeta{TenantID: cred.TenantID, OccurredAt: now},
		}); err != nil {
			return err
		}

		out = CustomerLoginOutput{AccessToken: accessToken, RefreshToken: rawRefresh}
		return nil
	})
	return out, err
}
