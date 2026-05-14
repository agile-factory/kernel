package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/AgileFactory/kernel/platform/auth/domain"
	"github.com/AgileFactory/kernel/platform/auth/usecase"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
	tenantdomain "github.com/AgileFactory/kernel/platform/tenant/domain"
	auditusecase "github.com/AgileFactory/kernel/platform/audit/usecase"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

// mockUserRepo simule authdomain.UserRepository
type mockUserRepo struct {
	findByEmailFn func(ctx context.Context, email authdomain.Email) (*authdomain.User, error)
	findByIDFn    func(ctx context.Context, id string) (*authdomain.User, error)
	saveFn        func(ctx context.Context, user *authdomain.User) error
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

// mockHasher simule PasswordHasher
type mockHasher struct {
	hashFn func(password string) (string, error)
}

func (m *mockHasher) Hash(password string) (string, error) {
	if m.hashFn != nil {
		return m.hashFn(password)
	}
	return "hashed_" + password, nil
}

// mockVerifier simule PasswordVerifier
type mockVerifier struct {
	verifyFn func(hashed, plain string) error
}

func (m *mockVerifier) Verify(hashed, plain string) error {
	if m.verifyFn != nil {
		return m.verifyFn(hashed, plain)
	}
	// Par défaut : OK si hashed == "hashed_"+plain
	if hashed == "hashed_"+plain {
		return nil
	}
	return errors.New("invalid password")
}

// mockTokenGenerator simule TokenGenerator
type mockTokenGenerator struct {
	generateFn func(user *authdomain.User) (string, error)
}

func (m *mockTokenGenerator) GenerateToken(user *authdomain.User) (string, error) {
	if m.generateFn != nil {
		return m.generateFn(user)
	}
	return "jwt_token_for_" + user.ID, nil
}

// mockTokenValidator simule TokenValidator
type mockTokenValidator struct {
	validateFn func(token string) (string, error)
}

func (m *mockTokenValidator) ValidateToken(token string) (string, error) {
	if m.validateFn != nil {
		return m.validateFn(token)
	}
	return "user-001", nil
}

// mockTenantRepo simule tenantdomain.TenantRepository
type mockTenantRepo struct {
	findByIDFn func(ctx context.Context, id string) (*tenantdomain.Tenant, error)
}

func (m *mockTenantRepo) FindByID(ctx context.Context, id string) (*tenantdomain.Tenant, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return &tenantdomain.Tenant{ID: id, IsActive: true}, nil
}

func (m *mockTenantRepo) Save(ctx context.Context, t *tenantdomain.Tenant) error { return nil }
func (m *mockTenantRepo) FindBySlug(ctx context.Context, slug string) (*tenantdomain.Tenant, error) {
	return nil, tenantdomain.ErrTenantNotFound
}
func (m *mockTenantRepo) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	return false, nil
}

// mockMembershipRepo simule memberdomain.MembershipRepository
type mockMembershipRepo struct {
	saveFn func(ctx context.Context, m *memberdomain.Membership) error
}

func (m *mockMembershipRepo) Save(ctx context.Context, mem *memberdomain.Membership) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, mem)
	}
	return nil
}
func (m *mockMembershipRepo) FindByID(ctx context.Context, id string) (*memberdomain.Membership, error) {
	return nil, memberdomain.ErrMembershipNotFound
}
func (m *mockMembershipRepo) FindByUserAndTenant(ctx context.Context, userID, tenantID string) (*memberdomain.Membership, error) {
	return nil, memberdomain.ErrMembershipNotFound
}
func (m *mockMembershipRepo) ListByTenant(ctx context.Context, tenantID string) ([]*memberdomain.Membership, error) {
	return nil, nil
}
func (m *mockMembershipRepo) ListByUser(ctx context.Context, userID string) ([]*memberdomain.Membership, error) {
	return nil, nil
}
func (m *mockMembershipRepo) UpdateStatus(ctx context.Context, id string, status memberdomain.MemberStatus, joinedAt *time.Time) error {
	return nil
}

// mockAuditRepo simule auditdomain.AuditRepository (no-op pour les tests unitaires)
type mockAuditRepo struct{}

func (m *mockAuditRepo) Save(ctx context.Context, log *auditdomain.AuditLog) error { return nil }
func (m *mockAuditRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*auditdomain.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditRepo) ListAll(ctx context.Context, limit, offset int) ([]*auditdomain.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditRepo) ListByTenantCursor(ctx context.Context, tenantID string, limit int, after *auditdomain.CursorToken) ([]*auditdomain.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditRepo) ListAllCursor(ctx context.Context, limit int, after *auditdomain.CursorToken) ([]*auditdomain.AuditLog, error) {
	return nil, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newAuditUseCase() *auditusecase.LogEventUseCase {
	return auditusecase.NewLogEventUseCase(&mockAuditRepo{})
}

func newRegisterUC(
	repo *mockUserRepo,
	hasher *mockHasher,
	tenantRepo *mockTenantRepo,
	memberRepo *mockMembershipRepo,
) *usecase.RegisterUseCase {
	return usecase.NewRegisterUseCase(repo, hasher, tenantRepo, memberRepo, newAuditUseCase())
}

func newLoginUC(repo *mockUserRepo, verifier *mockVerifier, gen *mockTokenGenerator) *usecase.LoginUseCase {
	return usecase.NewLoginUseCase(repo, verifier, gen, newAuditUseCase())
}

// ─── RegisterUseCase ─────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	repo := &mockUserRepo{}
	uc := newRegisterUC(repo, &mockHasher{}, &mockTenantRepo{}, &mockMembershipRepo{})

	user, err := uc.Execute(context.Background(),
		"user-001", "tenant-001", "alice@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("Execute() returned nil user on success")
	}
	if user.ID != "user-001" {
		t.Errorf("user.ID: got %q, want %q", user.ID, "user-001")
	}
}

func TestRegister_InvalidEmail(t *testing.T) {
	uc := newRegisterUC(&mockUserRepo{}, &mockHasher{}, &mockTenantRepo{}, &mockMembershipRepo{})

	_, err := uc.Execute(context.Background(),
		"user-002", "tenant-001", "not-an-email", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err == nil {
		t.Fatal("Execute() should return error for invalid email")
	}
	if err != authdomain.ErrInvalidEmail {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestRegister_TenantNotFound(t *testing.T) {
	tenantRepo := &mockTenantRepo{
		findByIDFn: func(ctx context.Context, id string) (*tenantdomain.Tenant, error) {
			return nil, tenantdomain.ErrTenantNotFound
		},
	}
	uc := newRegisterUC(&mockUserRepo{}, &mockHasher{}, tenantRepo, &mockMembershipRepo{})

	_, err := uc.Execute(context.Background(),
		"user-003", "ghost-tenant", "bob@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err == nil {
		t.Fatal("Execute() should fail when tenant is not found")
	}
}

func TestRegister_TenantInactive(t *testing.T) {
	tenantRepo := &mockTenantRepo{
		findByIDFn: func(ctx context.Context, id string) (*tenantdomain.Tenant, error) {
			return &tenantdomain.Tenant{ID: id, IsActive: false}, nil
		},
	}
	uc := newRegisterUC(&mockUserRepo{}, &mockHasher{}, tenantRepo, &mockMembershipRepo{})

	_, err := uc.Execute(context.Background(),
		"user-004", "inactive-tenant", "carol@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err == nil {
		t.Fatal("Execute() should fail for inactive tenant")
	}
}

func TestRegister_EmailAlreadyTaken(t *testing.T) {
	existingUser := authdomain.NewUser("existing-001", authdomain.Email{Value: "taken@agile.io"}, "hash", authdomain.ROLE_VIEWER)
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email authdomain.Email) (*authdomain.User, error) {
			return existingUser, nil
		},
	}
	uc := newRegisterUC(repo, &mockHasher{}, &mockTenantRepo{}, &mockMembershipRepo{})

	_, err := uc.Execute(context.Background(),
		"user-005", "tenant-001", "taken@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err != authdomain.ErrUserAlreadyExists {
		t.Errorf("expected ErrUserAlreadyExists, got %v", err)
	}
}

func TestRegister_HasherError(t *testing.T) {
	hasher := &mockHasher{
		hashFn: func(password string) (string, error) {
			return "", errors.New("bcrypt: internal error")
		},
	}
	uc := newRegisterUC(&mockUserRepo{}, hasher, &mockTenantRepo{}, &mockMembershipRepo{})

	_, err := uc.Execute(context.Background(),
		"user-006", "tenant-001", "dave@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err == nil {
		t.Fatal("Execute() should propagate hasher error")
	}
}

func TestRegister_RepoSaveError(t *testing.T) {
	repo := &mockUserRepo{
		saveFn: func(ctx context.Context, user *authdomain.User) error {
			return errors.New("db: connection lost")
		},
	}
	uc := newRegisterUC(repo, &mockHasher{}, &mockTenantRepo{}, &mockMembershipRepo{})

	_, err := uc.Execute(context.Background(),
		"user-007", "tenant-001", "eve@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err == nil {
		t.Fatal("Execute() should propagate repo save error")
	}
}

func TestRegister_MembershipSaveError(t *testing.T) {
	memberRepo := &mockMembershipRepo{
		saveFn: func(ctx context.Context, m *memberdomain.Membership) error {
			return errors.New("db: membership save failed")
		},
	}
	uc := newRegisterUC(&mockUserRepo{}, &mockHasher{}, &mockTenantRepo{}, memberRepo)

	_, err := uc.Execute(context.Background(),
		"user-008", "tenant-001", "frank@agile.io", "Password123!", memberdomain.MEMBER_ROLE_OWNER,
	)
	if err == nil {
		t.Fatal("Execute() should propagate membership save error")
	}
}

// ─── LoginUseCase ────────────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	email, _ := authdomain.NewEmail("alice@agile.io")
	storedUser := authdomain.NewUser("user-001", email, "hashed_Password123!", authdomain.ROLE_VIEWER)

	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) {
			return storedUser, nil
		},
	}
	uc := newLoginUC(repo, &mockVerifier{}, &mockTokenGenerator{})

	token, user, err := uc.Execute(context.Background(), "alice@agile.io", "Password123!")
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if token == "" {
		t.Error("Execute() should return a non-empty token")
	}
	if user == nil || user.ID != "user-001" {
		t.Errorf("Execute() returned unexpected user: %+v", user)
	}
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	uc := newLoginUC(&mockUserRepo{}, &mockVerifier{}, &mockTokenGenerator{})

	_, _, err := uc.Execute(context.Background(), "not-an-email", "Password123!")
	if err != authdomain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials for bad email, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, email authdomain.Email) (*authdomain.User, error) {
			return nil, authdomain.ErrUserNotFound
		},
	}
	uc := newLoginUC(repo, &mockVerifier{}, &mockTokenGenerator{})

	_, _, err := uc.Execute(context.Background(), "ghost@agile.io", "Password123!")
	// Doit retourner ErrInvalidCredentials (pas ErrUserNotFound — anti-enumeration)
	if err != authdomain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials (anti-enumeration), got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	email, _ := authdomain.NewEmail("alice@agile.io")
	storedUser := authdomain.NewUser("user-001", email, "hashed_correct_pass", authdomain.ROLE_VIEWER)

	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) {
			return storedUser, nil
		},
	}
	verifier := &mockVerifier{
		verifyFn: func(hashed, plain string) error {
			return errors.New("password mismatch")
		},
	}
	uc := newLoginUC(repo, verifier, &mockTokenGenerator{})

	_, _, err := uc.Execute(context.Background(), "alice@agile.io", "wrong_password")
	if err != authdomain.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials on wrong password, got %v", err)
	}
}

func TestLogin_TokenGenerationError(t *testing.T) {
	email, _ := authdomain.NewEmail("alice@agile.io")
	storedUser := authdomain.NewUser("user-001", email, "hashed_Password123!", authdomain.ROLE_VIEWER)

	repo := &mockUserRepo{
		findByEmailFn: func(ctx context.Context, e authdomain.Email) (*authdomain.User, error) {
			return storedUser, nil
		},
	}
	gen := &mockTokenGenerator{
		generateFn: func(user *authdomain.User) (string, error) {
			return "", errors.New("jwt: signing key unavailable")
		},
	}
	uc := newLoginUC(repo, &mockVerifier{}, gen)

	_, _, err := uc.Execute(context.Background(), "alice@agile.io", "Password123!")
	if err == nil {
		t.Fatal("Execute() should propagate token generation error")
	}
}

// ─── RefreshUseCase ──────────────────────────────────────────────────────────

func TestRefresh_Success(t *testing.T) {
	email, _ := authdomain.NewEmail("alice@agile.io")
	storedUser := authdomain.NewUser("user-001", email, "hashed_pass", authdomain.ROLE_VIEWER)

	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*authdomain.User, error) {
			return storedUser, nil
		},
	}
	uc := usecase.NewRefreshUseCase(repo, &mockTokenValidator{}, &mockTokenGenerator{})

	newToken, err := uc.Execute(context.Background(), "old_valid_token")
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if newToken == "" {
		t.Error("Execute() should return a non-empty new token")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	validator := &mockTokenValidator{
		validateFn: func(token string) (string, error) {
			return "", errors.New("jwt: token expired")
		},
	}
	uc := usecase.NewRefreshUseCase(&mockUserRepo{}, validator, &mockTokenGenerator{})

	_, err := uc.Execute(context.Background(), "expired_token")
	if err == nil {
		t.Fatal("Execute() should fail on invalid token")
	}
}

func TestRefresh_UserDisappeared(t *testing.T) {
	repo := &mockUserRepo{
		findByIDFn: func(ctx context.Context, id string) (*authdomain.User, error) {
			return nil, authdomain.ErrUserNotFound
		},
	}
	uc := usecase.NewRefreshUseCase(repo, &mockTokenValidator{}, &mockTokenGenerator{})

	_, err := uc.Execute(context.Background(), "valid_token_for_deleted_user")
	if err != authdomain.ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}
