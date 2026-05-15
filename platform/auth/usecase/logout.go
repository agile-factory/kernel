package usecase

import (
	"context"
	stderrors "errors"

	"github.com/AgileFactory/kernel/kernel/errors"
	authdomain "github.com/AgileFactory/kernel/platform/auth/domain"
)

// LogoutUseCase revokes the supplied refresh token. The matching access
// token remains valid until its short TTL expires (typically <1 hour),
// which is the accepted trade-off for stateless JWT access tokens.
type LogoutUseCase struct {
	refreshTokens authdomain.RefreshTokenRepository
}

func NewLogoutUseCase(refreshTokens authdomain.RefreshTokenRepository) *LogoutUseCase {
	return &LogoutUseCase{refreshTokens: refreshTokens}
}

// Execute revokes the refresh token. Idempotent: an already-revoked or
// unknown token returns nil (no information leak — the client cannot
// distinguish "we revoked it" from "it was never valid").
func (uc *LogoutUseCase) Execute(ctx context.Context, rawRefreshToken string) error {
	if rawRefreshToken == "" {
		return nil
	}
	hash := authdomain.HashRefreshToken(rawRefreshToken)
	tok, err := uc.refreshTokens.FindByHash(ctx, hash)
	if err != nil {
		if stderrors.Is(err, authdomain.ErrTokenNotFound) {
			return nil
		}
		return errors.WrapInternal(err)
	}
	if err := uc.refreshTokens.Revoke(ctx, tok.ID); err != nil {
		return errors.WrapInternal(err)
	}
	return nil
}
