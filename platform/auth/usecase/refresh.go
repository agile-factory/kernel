package usecase

import (
	"context"
	"log/slog"
	"time"

	"github.com/agile-factory/kernel/kernel/errors"
	authdomain "github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RefreshTokenUseCase implements rolling refresh (RFC-0018 §1):
//   - Validate the presented refresh token (lookup by SHA-256 hash, not
//     revoked, not expired).
//   - Revoke the current refresh token and mint a brand-new one.
//   - Issue a fresh access token.
//
// Atomicity (revoke-old + insert-new) is guaranteed via txctx so a crash in
// between cannot leave both the old and the new tokens valid simultaneously.
// The presented token's row is locked (FOR UPDATE) so two concurrent
// rotations of the same token cannot both succeed.
//
// Reuse detection: presenting a token that was already revoked means either
// a replay by an attacker or by the legitimate client after the attacker
// rotated it. Both cases revoke every refresh token of the user, forcing a
// fresh login.
type RefreshTokenUseCase struct {
	runInTx         txRunner
	users           authdomain.UserRepository
	refreshTokens   authdomain.RefreshTokenRepository
	tokenIssuer     TokenGenerator
	refreshTokenTTL time.Duration
}

func NewRefreshTokenUseCase(
	pool *pgxpool.Pool,
	users authdomain.UserRepository,
	refreshTokens authdomain.RefreshTokenRepository,
	tokenIssuer TokenGenerator,
	refreshTokenTTL time.Duration,
) *RefreshTokenUseCase {
	return &RefreshTokenUseCase{
		runInTx:         poolTxRunner(pool),
		users:           users,
		refreshTokens:   refreshTokens,
		tokenIssuer:     tokenIssuer,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// RefreshTokenOutput is what /api/v1/auth/refresh returns to the client.
type RefreshTokenOutput struct {
	AccessToken  string
	RefreshToken string // the NEW raw refresh token; the old one is revoked
	User         *authdomain.User
}

// Execute rotates the refresh token presented in rawRefreshToken.
func (uc *RefreshTokenUseCase) Execute(ctx context.Context, rawRefreshToken string) (*RefreshTokenOutput, error) {
	hash := authdomain.HashRefreshToken(rawRefreshToken)

	out := &RefreshTokenOutput{}
	var reusedByUserID string

	txErr := uc.runInTx(ctx, func(ctx context.Context) error {
		stored, err := uc.refreshTokens.FindByHashForUpdate(ctx, hash)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if stored.RevokedAt != nil {
			reusedByUserID = stored.UserID
			return authdomain.ErrRefreshTokenRevoked
		}
		if !stored.IsActive(now) {
			return authdomain.ErrTokenExpired
		}

		user, err := uc.users.FindByID(ctx, stored.UserID)
		if err != nil {
			return err
		}
		if !user.IsActive() {
			return authdomain.ErrUserNotActive
		}

		// Rolling rotation: revoke current, mint new.
		if err := uc.refreshTokens.Revoke(ctx, stored.ID); err != nil {
			return errors.WrapInternal(err)
		}
		rawNew, newTok, err := authdomain.NewRefreshToken(uuid.NewString(), user.ID, stored.TenantID, uc.refreshTokenTTL)
		if err != nil {
			return errors.WrapInternal(err)
		}
		if err := uc.refreshTokens.Save(ctx, newTok); err != nil {
			return errors.WrapInternal(err)
		}

		out.User = user
		out.RefreshToken = rawNew
		return nil
	})
	if txErr != nil {
		if reusedByUserID != "" {
			// Outside the rolled-back tx so the revocation sticks.
			if err := uc.refreshTokens.RevokeAllForUser(ctx, reusedByUserID); err != nil {
				slog.ErrorContext(ctx, "refresh token reuse: revoke all failed", "user_id", reusedByUserID, "error", err)
			} else {
				slog.WarnContext(ctx, "refresh token reuse detected; all sessions revoked", "user_id", reusedByUserID)
			}
		}
		return nil, txErr
	}

	accessToken, err := uc.tokenIssuer.GenerateToken(out.User)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}
	out.AccessToken = accessToken
	return out, nil
}
