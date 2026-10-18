package usecase

import (
	"context"
	"errors"

	"github.com/agile-factory/kernel/platform/customer_auth/domain"
)

// CustomerLogoutUseCase revokes the given refresh token.
// Idempotent: if the token is already gone the call succeeds silently.
type CustomerLogoutUseCase struct {
	refreshRepo domain.CustomerRefreshTokenRepository
}

// NewCustomerLogoutUseCase wires the use case.
func NewCustomerLogoutUseCase(refreshRepo domain.CustomerRefreshTokenRepository) *CustomerLogoutUseCase {
	return &CustomerLogoutUseCase{refreshRepo: refreshRepo}
}

// Execute revokes the refresh token identified by rawToken.
func (uc *CustomerLogoutUseCase) Execute(ctx context.Context, rawToken string) error {
	stored, err := uc.refreshRepo.FindByHash(ctx, hashToken(rawToken))
	if err != nil {
		// Token already gone — idempotent logout.
		return nil
	}
	if err := uc.refreshRepo.Revoke(ctx, stored.ID); err != nil && !errors.Is(err, domain.ErrTokenConsumed) {
		return err
	}
	return nil
}
