package usecase

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CustomerRefreshUseCase issues a new token pair from a valid refresh token.
// Implements a rolling-refresh pattern: the old refresh token is revoked
// and a new one is issued atomically.
type CustomerRefreshUseCase struct {
	db          *pgxpool.Pool
	refreshRepo domain.CustomerRefreshTokenRepository
	jwtSvc      CustomerJWTIssuer
	refreshTTL  time.Duration
}

// NewCustomerRefreshUseCase wires the use case.
func NewCustomerRefreshUseCase(
	db *pgxpool.Pool,
	refreshRepo domain.CustomerRefreshTokenRepository,
	jwtSvc CustomerJWTIssuer,
	refreshTTL time.Duration,
) *CustomerRefreshUseCase {
	return &CustomerRefreshUseCase{db: db, refreshRepo: refreshRepo, jwtSvc: jwtSvc, refreshTTL: refreshTTL}
}

// CustomerRefreshOutput holds the newly issued token pair.
type CustomerRefreshOutput struct {
	AccessToken  string
	RefreshToken string
}

// Execute validates the given raw refresh token, revokes it, and returns a new pair.
// Returns ErrTokenNotFound or ErrTokenExpired on invalid input.
func (uc *CustomerRefreshUseCase) Execute(ctx context.Context, rawToken string) (CustomerRefreshOutput, error) {
	now := time.Now()
	stored, err := uc.refreshRepo.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		return CustomerRefreshOutput{}, domain.ErrTokenNotFound
	}
	if !stored.IsValid(now) {
		return CustomerRefreshOutput{}, domain.ErrTokenExpired
	}

	var out CustomerRefreshOutput
	err = execWithOptionalTx(ctx, uc.db, func(ctx context.Context) error {
		if err := uc.refreshRepo.Revoke(ctx, stored.ID); err != nil {
			return err
		}

		accessToken, err := uc.jwtSvc.GenerateToken(stored.EndCustomerID, stored.TenantID)
		if err != nil {
			return err
		}

		newRaw := generateOpaqueToken()
		newToken := &domain.CustomerRefreshToken{
			EndCustomerID: stored.EndCustomerID,
			TenantID:      stored.TenantID,
			TokenHash:     hashToken(newRaw),
			ExpiresAt:     now.Add(uc.refreshTTL),
		}
		if err := uc.refreshRepo.Create(ctx, newToken); err != nil {
			return err
		}

		out = CustomerRefreshOutput{AccessToken: accessToken, RefreshToken: newRaw}
		return nil
	})
	return out, err
}
