package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agile-factory/kernel/platform/audit"
	authdomain "github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/agile-factory/kernel/platform/auth/usecase"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

// mockUserRepo simulates authdomain.UserRepository for the in-memory tests
// below. Test-specific behaviour is plugged in via the *Fn fields.
type mockUserRepo struct {
	findByEmailFn        func(ctx context.Context, email authdomain.Email) (*authdomain.User, error)
	findByIDFn           func(ctx context.Context, id string) (*authdomain.User, error)
	saveFn               func(ctx context.Context, user *authdomain.User) error
	updateEmailVerified  func(ctx context.Context, userID string) error
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email authdomain.Email) (*authdomain.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, authdomain.ErrUserNotFound
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*authdomain.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, authdomain.ErrUserNotFound
}

func (m *mockUserRepo) Save(ctx context.Context, user *authdomain.User) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, user)
	}
	return nil
}

func (m *mockUserRepo) UpdateEmailVerified(ctx context.Context, userID string) error {
	if m.updateEmailVerified != nil {
		return m.updateEmailVerified(ctx, userID)
	}
	return nil
}

// mockRefreshTokenRepo simulates authdomain.RefreshTokenRepository in memory.
type mockRefreshTokenRepo struct {
	saved             []*authdomain.RefreshToken
	revokedIDs        []string
	revokedAllForUser []string
	findFn            func(ctx context.Context, hash string) (*authdomain.RefreshToken, error)
}

func (m *mockRefreshTokenRepo) Save(ctx context.Context, t *authdomain.RefreshToken) error {
	m.saved = append(m.saved, t)
	return nil
}

func (m *mockRefreshTokenRepo) FindByHash(ctx context.Context, hash string) (*authdomain.RefreshToken, error) {
	if m.findFn != nil {
		return m.findFn(ctx, hash)
	}
	return nil, authdomain.ErrTokenNotFound
}

func (m *mockRefreshTokenRepo) FindByHashForUpdate(ctx context.Context, hash string) (*authdomain.RefreshToken, error) {
	return m.FindByHash(ctx, hash)
}

func (m *mockRefreshTokenRepo) RevokeAllForUser(ctx context.Context, userID string) error {
	m.revokedAllForUser = append(m.revokedAllForUser, userID)
	return nil
}

func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, id string) error {
	m.revokedIDs = append(m.revokedIDs, id)
	return nil
}

type mockVerifier struct {
	verifyFn func(hashed, plain string) error
}

func (m *mockVerifier) Verify(hashed, plain string) error {
	if m.verifyFn != nil {
		return m.verifyFn(hashed, plain)
	}
	if hashed == "hashed_"+plain {
		return nil
	}
	return errors.New("invalid password")
}

type mockTokenGenerator struct {
	generateFn func(user *authdomain.User) (string, error)
}

func (m *mockTokenGenerator) GenerateToken(user *authdomain.User) (string, error) {
	if m.generateFn != nil {
		return m.generateFn(user)
	}
	return "jwt_token_for_" + user.ID, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// noopAuditLogger is a test double for audit.EventLogger that discards all events.
type noopAuditLogger struct{}

func (n *noopAuditLogger) Execute(_ context.Context, _ audit.LogEventParams) {}

func newAuditUseCase() audit.EventLogger {
	return &noopAuditLogger{}
}

func newLoginUC(repo *mockUserRepo, refresh *mockRefreshTokenRepo, verifier *mockVerifier, gen *mockTokenGenerator) *usecase.LoginUseCase {
	return usecase.NewLoginUseCase(repo, refresh, verifier, gen, newAuditUseCase(), 30*24*time.Hour)
}

func activeUser(t *testing.T, id, emailStr string) *authdomain.User {
	t.Helper()
	email, err := authdomain.NewEmail(emailStr)
	if err != nil {
		t.Fatalf("activeUser: invalid email: %v", err)
	}
	u := authdomain.NewUser(id, email, "hashed_Password123!", authdomain.ROLE_VIEWER)
	u.MarkEmailVerified()
	return u
}

// ─── LoginUseCase ────────────────────────────────────────────────────────────

func TestLogin_Success_IssuesAccessAndRefresh(t *testing.T) {
	user := activeUser(t, "user-001", "alice@agile.io")
	users := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) { return user, nil },
	}
	refresh := &mockRefreshTokenRepo{}

	uc := newLoginUC(users, refresh, &mockVerifier{}, &mockTokenGenerator{})

	out, err := uc.Execute(context.Background(), "alice@agile.io", "Password123!")
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if out.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if out.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if len(refresh.saved) != 1 {
		t.Errorf("expected exactly one refresh token persisted, got %d", len(refresh.saved))
	}
}

func TestLogin_InvalidEmailFormat_HidesAsInvalidCredentials(t *testing.T) {
	uc := newLoginUC(&mockUserRepo{}, &mockRefreshTokenRepo{}, &mockVerifier{}, &mockTokenGenerator{})
	_, err := uc.Execute(context.Background(), "not-an-email", "Password123!")
	if err != authdomain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for bad email, got %v", err)
	}
}

func TestLogin_UserNotFound_HidesAsInvalidCredentials(t *testing.T) {
	users := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) {
			return nil, authdomain.ErrUserNotFound
		},
	}
	uc := newLoginUC(users, &mockRefreshTokenRepo{}, &mockVerifier{}, &mockTokenGenerator{})
	_, err := uc.Execute(context.Background(), "ghost@agile.io", "Password123!")
	if err != authdomain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials (anti-enumeration), got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	user := activeUser(t, "user-001", "alice@agile.io")
	users := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) { return user, nil },
	}
	verifier := &mockVerifier{
		verifyFn: func(hashed, plain string) error { return errors.New("password mismatch") },
	}
	uc := newLoginUC(users, &mockRefreshTokenRepo{}, verifier, &mockTokenGenerator{})

	_, err := uc.Execute(context.Background(), "alice@agile.io", "wrong_password")
	if err != authdomain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_PendingEmail_BlocksWithUserNotActive(t *testing.T) {
	email, _ := authdomain.NewEmail("alice@agile.io")
	// Fresh user — still in PENDING_EMAIL.
	user := authdomain.NewUser("user-001", email, "hashed_Password123!", authdomain.ROLE_VIEWER)
	users := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) { return user, nil },
	}
	uc := newLoginUC(users, &mockRefreshTokenRepo{}, &mockVerifier{}, &mockTokenGenerator{})

	_, err := uc.Execute(context.Background(), "alice@agile.io", "Password123!")
	if err != authdomain.ErrUserNotActive {
		t.Errorf("expected ErrUserNotActive, got %v", err)
	}
}

func TestLogin_TokenGenerationError_WrapsInternal(t *testing.T) {
	user := activeUser(t, "user-001", "alice@agile.io")
	users := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) { return user, nil },
	}
	gen := &mockTokenGenerator{
		generateFn: func(user *authdomain.User) (string, error) {
			return "", errors.New("jwt: signing key unavailable")
		},
	}
	uc := newLoginUC(users, &mockRefreshTokenRepo{}, &mockVerifier{}, gen)

	_, err := uc.Execute(context.Background(), "alice@agile.io", "Password123!")
	if err == nil {
		t.Fatal("Execute() should propagate token generation error")
	}
}

// ─── LogoutUseCase ───────────────────────────────────────────────────────────

func TestLogout_RevokesPresentedToken(t *testing.T) {
	stored := &authdomain.RefreshToken{
		ID:        "rt-001",
		UserID:    "user-001",
		TokenHash: authdomain.HashRefreshToken("raw-token-abc"),
	}
	refresh := &mockRefreshTokenRepo{
		findFn: func(ctx context.Context, hash string) (*authdomain.RefreshToken, error) {
			if hash == stored.TokenHash {
				return stored, nil
			}
			return nil, authdomain.ErrTokenNotFound
		},
	}
	uc := usecase.NewLogoutUseCase(refresh)

	if err := uc.Execute(context.Background(), "raw-token-abc"); err != nil {
		t.Fatalf("Logout: unexpected error: %v", err)
	}
	if len(refresh.revokedIDs) != 1 || refresh.revokedIDs[0] != "rt-001" {
		t.Errorf("expected exactly one revocation of rt-001, got %v", refresh.revokedIDs)
	}
}

func TestLogout_UnknownToken_IsIdempotent(t *testing.T) {
	refresh := &mockRefreshTokenRepo{} // findFn returns ErrTokenNotFound by default
	uc := usecase.NewLogoutUseCase(refresh)

	if err := uc.Execute(context.Background(), "unknown-token"); err != nil {
		t.Errorf("Logout on unknown token should be a no-op, got %v", err)
	}
	if len(refresh.revokedIDs) != 0 {
		t.Errorf("expected no revocation, got %v", refresh.revokedIDs)
	}
}

func TestLogout_EmptyToken_IsNoOp(t *testing.T) {
	refresh := &mockRefreshTokenRepo{}
	uc := usecase.NewLogoutUseCase(refresh)
	if err := uc.Execute(context.Background(), ""); err != nil {
		t.Errorf("Logout on empty token should be a no-op, got %v", err)
	}
}
