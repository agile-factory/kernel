package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/agile-factory/kernel/platform/auth/usecase"
)

func storedRefresh(t *testing.T, userID string) (string, *authdomain.RefreshToken) {
	t.Helper()
	raw, tok, err := authdomain.NewRefreshToken("rt-1", userID, "", time.Hour)
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	return raw, tok
}

func newRefreshUC(users *mockUserRepo, refresh *mockRefreshTokenRepo) *usecase.RefreshTokenUseCase {
	return usecase.NewRefreshTokenUseCase(nil, users, refresh, &mockTokenGenerator{}, time.Hour).WithoutTx()
}

func TestRefresh_Rotates(t *testing.T) {
	user := activeUser(t, "user-1", "alice@agile.io")
	raw, tok := storedRefresh(t, user.ID)
	refresh := &mockRefreshTokenRepo{
		findFn: func(_ context.Context, hash string) (*authdomain.RefreshToken, error) {
			if hash != tok.TokenHash {
				return nil, authdomain.ErrTokenNotFound
			}
			return tok, nil
		},
	}
	users := &mockUserRepo{findByIDFn: func(context.Context, string) (*authdomain.User, error) { return user, nil }}

	out, err := newRefreshUC(users, refresh).Execute(context.Background(), raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.RefreshToken == "" || out.RefreshToken == raw {
		t.Fatal("expected a new refresh token")
	}
	if len(refresh.revokedIDs) != 1 || refresh.revokedIDs[0] != tok.ID {
		t.Fatalf("old token should be revoked, got %v", refresh.revokedIDs)
	}
	if len(refresh.revokedAllForUser) != 0 {
		t.Fatal("a normal rotation must not revoke the whole family")
	}
}

func TestRefresh_ReplayOfRevokedToken_RevokesAllUserTokens(t *testing.T) {
	raw, tok := storedRefresh(t, "user-1")
	revokedAt := time.Now().Add(-time.Minute)
	tok.RevokedAt = &revokedAt
	refresh := &mockRefreshTokenRepo{
		findFn: func(context.Context, string) (*authdomain.RefreshToken, error) { return tok, nil },
	}

	_, err := newRefreshUC(&mockUserRepo{}, refresh).Execute(context.Background(), raw)
	if !errors.Is(err, authdomain.ErrRefreshTokenRevoked) {
		t.Fatalf("want ErrRefreshTokenRevoked, got %v", err)
	}
	if len(refresh.revokedAllForUser) != 1 || refresh.revokedAllForUser[0] != "user-1" {
		t.Fatalf("expected all tokens of user-1 revoked, got %v", refresh.revokedAllForUser)
	}
	if len(refresh.saved) != 0 {
		t.Fatal("no new token must be issued on replay")
	}
}

func TestRefresh_UnknownToken(t *testing.T) {
	refresh := &mockRefreshTokenRepo{}
	_, err := newRefreshUC(&mockUserRepo{}, refresh).Execute(context.Background(), "nope")
	if !errors.Is(err, authdomain.ErrTokenNotFound) {
		t.Fatalf("want ErrTokenNotFound, got %v", err)
	}
	if len(refresh.revokedAllForUser) != 0 {
		t.Fatal("unknown token must not trigger family revocation")
	}
}
