package customer_auth_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	kernevents "github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/customer_auth"
	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/agile-factory/kernel/platform/customer_auth/usecase"
)

// ─── Stubs ────────────────────────────────────────────────────────────────────

// credRepoWithEmail is an in-memory CredentialRepository with email indexing.
type credRepoWithEmail struct {
	mu      sync.Mutex
	byID    map[string]*domain.CustomerCredential
	byEmail map[string]*domain.CustomerCredential // keyed by "email:tenantID"
}

func newCredRepoWithEmail() *credRepoWithEmail {
	return &credRepoWithEmail{
		byID:    map[string]*domain.CustomerCredential{},
		byEmail: map[string]*domain.CustomerCredential{},
	}
}

func (r *credRepoWithEmail) seedWithEmail(cred *domain.CustomerCredential, email string) {
	cp := *cred
	if cp.ID == "" {
		cp.ID = "cred-" + cred.EndCustomerID
	}
	r.byID[cp.EndCustomerID] = &cp
	r.byEmail[email+":"+cp.TenantID] = &cp
}

func (r *credRepoWithEmail) Create(_ context.Context, cred *domain.CustomerCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[cred.EndCustomerID]; ok {
		return domain.ErrCredentialAlreadyExists
	}
	cp := *cred
	cp.ID = "cred-" + cred.EndCustomerID
	r.byID[cp.EndCustomerID] = &cp
	return nil
}

func (r *credRepoWithEmail) FindByEndCustomerID(_ context.Context, id string) (*domain.CustomerCredential, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *credRepoWithEmail) FindByEmail(_ context.Context, email, tenantID string) (*domain.CustomerCredential, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byEmail[email+":"+tenantID]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *credRepoWithEmail) Update(_ context.Context, cred *domain.CustomerCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[cred.EndCustomerID]; !ok {
		return domain.ErrCredentialNotFound
	}
	cp := *cred
	r.byID[cp.EndCustomerID] = &cp
	return nil
}

// simpleCredRepo is a minimal in-memory CredentialRepository without email.
type simpleCredRepo struct {
	mu   sync.Mutex
	byID map[string]*domain.CustomerCredential
}

func newSimpleCredRepo(creds ...*domain.CustomerCredential) *simpleCredRepo {
	r := &simpleCredRepo{byID: map[string]*domain.CustomerCredential{}}
	for _, c := range creds {
		cp := *c
		if cp.ID == "" {
			cp.ID = "cred-" + c.EndCustomerID
		}
		r.byID[c.EndCustomerID] = &cp
	}
	return r
}

func (r *simpleCredRepo) Create(_ context.Context, cred *domain.CustomerCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[cred.EndCustomerID]; ok {
		return domain.ErrCredentialAlreadyExists
	}
	cp := *cred
	cp.ID = "cred-" + cred.EndCustomerID
	r.byID[cp.EndCustomerID] = &cp
	return nil
}

func (r *simpleCredRepo) FindByEndCustomerID(_ context.Context, id string) (*domain.CustomerCredential, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrCredentialNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *simpleCredRepo) FindByEmail(_ context.Context, _, _ string) (*domain.CustomerCredential, error) {
	return nil, domain.ErrCredentialNotFound
}

func (r *simpleCredRepo) Update(_ context.Context, cred *domain.CustomerCredential) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.byID[cred.EndCustomerID]; !ok {
		return domain.ErrCredentialNotFound
	}
	cp := *cred
	r.byID[cp.EndCustomerID] = &cp
	return nil
}

// stubRefreshRepo is an in-memory CustomerRefreshTokenRepository.
type stubRefreshRepo struct {
	mu     sync.Mutex
	byHash map[string]*domain.CustomerRefreshToken
}

func newStubRefreshRepo() *stubRefreshRepo {
	return &stubRefreshRepo{byHash: map[string]*domain.CustomerRefreshToken{}}
}

func (r *stubRefreshRepo) Create(_ context.Context, t *domain.CustomerRefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	if cp.ID == "" {
		cp.ID = "rt-" + t.TokenHash[:8]
	}
	r.byHash[t.TokenHash] = &cp
	return nil
}

func (r *stubRefreshRepo) FindByHash(_ context.Context, hash string) (*domain.CustomerRefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byHash[hash]
	if !ok || t.RevokedAt != nil {
		return nil, domain.ErrTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *stubRefreshRepo) Revoke(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, t := range r.byHash {
		if t.ID == id {
			t.RevokedAt = &now
			return nil
		}
	}
	return nil
}

func (r *stubRefreshRepo) RevokeAllForEndCustomer(_ context.Context, endCustomerID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, t := range r.byHash {
		if t.EndCustomerID == endCustomerID {
			t.RevokedAt = &now
		}
	}
	return nil
}

// stubResetRepo is an in-memory CustomerResetTokenRepository.
type stubResetRepo struct {
	mu     sync.Mutex
	byHash map[string]*domain.CustomerResetToken
}

func newStubResetRepo() *stubResetRepo {
	return &stubResetRepo{byHash: map[string]*domain.CustomerResetToken{}}
}

func (r *stubResetRepo) Create(_ context.Context, t *domain.CustomerResetToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	if cp.ID == "" {
		cp.ID = "reset-" + t.TokenHash[:8]
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	r.byHash[t.TokenHash] = &cp
	return nil
}

func (r *stubResetRepo) FindByHash(_ context.Context, hash string) (*domain.CustomerResetToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byHash[hash]
	if !ok {
		return nil, domain.ErrTokenNotFound
	}
	cp := *t
	return &cp, nil
}

func (r *stubResetRepo) MarkConsumed(_ context.Context, id string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byHash {
		if t.ID == id {
			t.ConsumedAt = &now
			return nil
		}
	}
	return nil
}

func (r *stubResetRepo) HasRecentToken(_ context.Context, endCustomerID string, since time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byHash {
		if t.EndCustomerID == endCustomerID && t.ConsumedAt == nil && t.CreatedAt.After(since) {
			return true, nil
		}
	}
	return false, nil
}

// stubHasher echoes password as "hash:<password>" — NOT secure, for tests only.
type stubHasher struct{}

func (h *stubHasher) Hash(password string) (string, error) { return "hash:" + password, nil }
func (h *stubHasher) Verify(password, hash string) error {
	if "hash:"+password == hash {
		return nil
	}
	return errors.New("wrong password")
}

// stubJWTSvc issues tokens as "jwt:<endCustomerID>:<tenantID>".
type stubJWTSvc struct{}

func (s *stubJWTSvc) GenerateToken(endCustomerID, tenantID string) (string, error) {
	return "jwt:" + endCustomerID + ":" + tenantID, nil
}

func (s *stubJWTSvc) ValidateToken(token string) (string, string, error) {
	if len(token) < 5 || token[:4] != "jwt:" {
		return "", "", errors.New("invalid token")
	}
	rest := token[4:]
	for i, ch := range rest {
		if ch == ':' {
			return rest[:i], rest[i+1:], nil
		}
	}
	return "", "", errors.New("invalid token format")
}

// stubPublisher records published events.
type stubPublisher struct {
	mu     sync.Mutex
	events []kernevents.Event
}

func (p *stubPublisher) Publish(_ context.Context, ev kernevents.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, ev)
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// testHash computes SHA-256 hex of s — mirrors usecase.hashToken for test setup.
func testHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// buildModule creates a wired customer_auth.Module with simple stub dependencies.
func buildModule(t *testing.T) (*customer_auth.Module, *http.ServeMux, *stubPublisher) {
	t.Helper()
	pub := &stubPublisher{}
	mod, err := customer_auth.New(customer_auth.Config{
		DB:          nil,
		CredRepo:    newSimpleCredRepo(),
		RefreshRepo: newStubRefreshRepo(),
		ResetRepo:   newStubResetRepo(),
		Hasher:      &stubHasher{},
		JWTSvc:      &stubJWTSvc{},
		Publisher:   pub,
		Logger:      testLogger(),
	})
	if err != nil {
		t.Fatalf("customer_auth.New: %v", err)
	}
	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)
	return mod, mux, pub
}

// ─── Use case tests ──────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	credRepo := newSimpleCredRepo()
	pub := &stubPublisher{}
	uc := usecase.NewCustomerRegisterUseCase(nil, credRepo, &stubHasher{}, pub)

	err := uc.Execute(context.Background(), usecase.CustomerRegisterInput{
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		Password:      "correct-horse-battery",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	pub.mu.Lock()
	evType := ""
	if len(pub.events) > 0 {
		evType = pub.events[0].Type
	}
	pub.mu.Unlock()
	if evType != "customer_auth.registered" {
		t.Errorf("expected customer_auth.registered event, got %q", evType)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	uc := usecase.NewCustomerRegisterUseCase(nil, newSimpleCredRepo(), &stubHasher{}, &stubPublisher{})
	err := uc.Execute(context.Background(), usecase.CustomerRegisterInput{
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		Password:      "short",
	})
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegister_Duplicate(t *testing.T) {
	existing := &domain.CustomerCredential{ID: "cred-ec-1", EndCustomerID: "ec-1", TenantID: "t-1", PasswordHash: "hash:x"}
	uc := usecase.NewCustomerRegisterUseCase(nil, newSimpleCredRepo(existing), &stubHasher{}, &stubPublisher{})
	err := uc.Execute(context.Background(), usecase.CustomerRegisterInput{
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		Password:      "correct-horse-battery",
	})
	if !errors.Is(err, domain.ErrCredentialAlreadyExists) {
		t.Errorf("expected ErrCredentialAlreadyExists, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	credRepo := newCredRepoWithEmail()
	credRepo.seedWithEmail(&domain.CustomerCredential{
		ID:            "cred-ec-1",
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		PasswordHash:  "hash:strongpassword123",
	}, "user@example.com")

	refreshRepo := newStubRefreshRepo()
	pub := &stubPublisher{}
	uc := usecase.NewCustomerLoginUseCase(nil, credRepo, refreshRepo, &stubHasher{}, &stubJWTSvc{}, pub, 30*24*time.Hour)

	out, err := uc.Execute(context.Background(), usecase.CustomerLoginInput{
		Email:    "user@example.com",
		Password: "strongpassword123",
		TenantID: "t-1",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Error("expected non-empty token pair")
	}
	pub.mu.Lock()
	evType := ""
	if len(pub.events) > 0 {
		evType = pub.events[0].Type
	}
	pub.mu.Unlock()
	if evType != "customer_auth.logged_in" {
		t.Errorf("expected customer_auth.logged_in event, got %q", evType)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	credRepo := newCredRepoWithEmail()
	credRepo.seedWithEmail(&domain.CustomerCredential{
		ID:            "cred-ec-1",
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		PasswordHash:  "hash:correctpassword123",
	}, "user@example.com")

	uc := usecase.NewCustomerLoginUseCase(nil, credRepo, newStubRefreshRepo(), &stubHasher{}, &stubJWTSvc{}, &stubPublisher{}, 30*24*time.Hour)
	_, err := uc.Execute(context.Background(), usecase.CustomerLoginInput{
		Email:    "user@example.com",
		Password: "wrongpassword",
		TenantID: "t-1",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	uc := usecase.NewCustomerLoginUseCase(nil, newCredRepoWithEmail(), newStubRefreshRepo(), &stubHasher{}, &stubJWTSvc{}, &stubPublisher{}, 30*24*time.Hour)
	_, err := uc.Execute(context.Background(), usecase.CustomerLoginInput{
		Email:    "ghost@example.com",
		Password: "anypassword123",
		TenantID: "t-1",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials (enumeration resistance), got %v", err)
	}
}

func TestLogin_AccountLocked(t *testing.T) {
	credRepo := newCredRepoWithEmail()
	lockUntil := time.Now().Add(15 * time.Minute)
	credRepo.seedWithEmail(&domain.CustomerCredential{
		ID:             "cred-ec-1",
		EndCustomerID:  "ec-1",
		TenantID:       "t-1",
		PasswordHash:   "hash:correctpassword123",
		FailedAttempts: 5,
		LockedUntil:    &lockUntil,
	}, "user@example.com")

	uc := usecase.NewCustomerLoginUseCase(nil, credRepo, newStubRefreshRepo(), &stubHasher{}, &stubJWTSvc{}, &stubPublisher{}, 30*24*time.Hour)
	_, err := uc.Execute(context.Background(), usecase.CustomerLoginInput{
		Email:    "user@example.com",
		Password: "correctpassword123",
		TenantID: "t-1",
	})
	if !errors.Is(err, domain.ErrAccountLocked) {
		t.Errorf("expected ErrAccountLocked, got %v", err)
	}
}

func TestRefresh_Success(t *testing.T) {
	refreshRepo := newStubRefreshRepo()
	rawToken := "opaque-test-token"
	hash := testHash(rawToken)
	_ = refreshRepo.Create(context.Background(), &domain.CustomerRefreshToken{
		ID:            "rt-1",
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		TokenHash:     hash,
		ExpiresAt:     time.Now().Add(30 * 24 * time.Hour),
	})

	uc := usecase.NewCustomerRefreshUseCase(nil, refreshRepo, &stubJWTSvc{}, 30*24*time.Hour)
	out, err := uc.Execute(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Error("expected non-empty token pair")
	}
	// Old token must be revoked.
	if _, lookupErr := refreshRepo.FindByHash(context.Background(), hash); lookupErr == nil {
		t.Error("expected old refresh token to be revoked after rolling refresh")
	}
}

func TestLogout_Idempotent(t *testing.T) {
	uc := usecase.NewCustomerLogoutUseCase(newStubRefreshRepo())
	if err := uc.Execute(context.Background(), "nonexistent"); err != nil {
		t.Errorf("Logout idempotent failed: %v", err)
	}
}

func TestForgotPassword_OTPRateLimit(t *testing.T) {
	credRepo := newCredRepoWithEmail()
	credRepo.seedWithEmail(&domain.CustomerCredential{
		ID:            "cred-ec-1",
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		PasswordHash:  "hash:password123",
	}, "user@example.com")

	resetRepo := newStubResetRepo()
	pub := &stubPublisher{}
	uc := usecase.NewCustomerForgotPasswordUseCase(nil, credRepo, resetRepo, pub)
	in := usecase.CustomerForgotPasswordInput{Email: "user@example.com", TenantID: "t-1"}

	if err := uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("ForgotPassword first: %v", err)
	}
	err := uc.Execute(context.Background(), in)
	if !errors.Is(err, domain.ErrOTPRateLimit) {
		t.Errorf("expected ErrOTPRateLimit on second request within 60s, got %v", err)
	}
}

func TestForgotPassword_UnknownEmail_Silent(t *testing.T) {
	uc := usecase.NewCustomerForgotPasswordUseCase(nil, newCredRepoWithEmail(), newStubResetRepo(), &stubPublisher{})
	if err := uc.Execute(context.Background(), usecase.CustomerForgotPasswordInput{
		Email:    "ghost@example.com",
		TenantID: "t-1",
	}); err != nil {
		t.Errorf("expected nil for unknown email (enumeration resistance), got %v", err)
	}
}

func TestResetPassword_Success(t *testing.T) {
	credRepo := newSimpleCredRepo(&domain.CustomerCredential{
		ID:            "cred-ec-1",
		EndCustomerID: "ec-1",
		TenantID:      "t-1",
		PasswordHash:  "hash:oldpassword123",
	})
	resetRepo := newStubResetRepo()
	rawToken := "reset-token-abc"
	_ = resetRepo.Create(context.Background(), &domain.CustomerResetToken{
		ID:            "reset-1",
		EndCustomerID: "ec-1",
		TokenHash:     testHash(rawToken),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
	})

	pub := &stubPublisher{}
	uc := usecase.NewCustomerResetPasswordUseCase(nil, credRepo, resetRepo, &stubHasher{}, pub)
	if err := uc.Execute(context.Background(), usecase.CustomerResetPasswordInput{
		ResetToken:  rawToken,
		NewPassword: "newstrongpassword123",
	}); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	pub.mu.Lock()
	evType := ""
	if len(pub.events) > 0 {
		evType = pub.events[0].Type
	}
	pub.mu.Unlock()
	if evType != "customer_auth.password_reset" {
		t.Errorf("expected customer_auth.password_reset event, got %q", evType)
	}
}

func TestResetPassword_ShortPassword(t *testing.T) {
	uc := usecase.NewCustomerResetPasswordUseCase(nil, newSimpleCredRepo(), newStubResetRepo(), &stubHasher{}, &stubPublisher{})
	err := uc.Execute(context.Background(), usecase.CustomerResetPasswordInput{
		ResetToken:  "any",
		NewPassword: "short",
	})
	if !errors.Is(err, domain.ErrPasswordTooShort) {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

// ─── Module wiring tests ──────────────────────────────────────────────────────

func TestNew_MissingDeps_Errors(t *testing.T) {
	pub := &stubPublisher{}
	log := testLogger()
	hasher := &stubHasher{}
	jwt := &stubJWTSvc{}
	credRepo := newSimpleCredRepo()
	refreshRepo := newStubRefreshRepo()
	resetRepo := newStubResetRepo()

	cases := []struct {
		name string
		cfg  customer_auth.Config
	}{
		{"no CredRepo", customer_auth.Config{CredRepo: nil, RefreshRepo: refreshRepo, ResetRepo: resetRepo, Hasher: hasher, JWTSvc: jwt, Publisher: pub, Logger: log}},
		{"no RefreshRepo", customer_auth.Config{CredRepo: credRepo, RefreshRepo: nil, ResetRepo: resetRepo, Hasher: hasher, JWTSvc: jwt, Publisher: pub, Logger: log}},
		{"no ResetRepo", customer_auth.Config{CredRepo: credRepo, RefreshRepo: refreshRepo, ResetRepo: nil, Hasher: hasher, JWTSvc: jwt, Publisher: pub, Logger: log}},
		{"no Hasher", customer_auth.Config{CredRepo: credRepo, RefreshRepo: refreshRepo, ResetRepo: resetRepo, Hasher: nil, JWTSvc: jwt, Publisher: pub, Logger: log}},
		{"no JWTSvc", customer_auth.Config{CredRepo: credRepo, RefreshRepo: refreshRepo, ResetRepo: resetRepo, Hasher: hasher, JWTSvc: nil, Publisher: pub, Logger: log}},
		{"no Publisher", customer_auth.Config{CredRepo: credRepo, RefreshRepo: refreshRepo, ResetRepo: resetRepo, Hasher: hasher, JWTSvc: jwt, Publisher: nil, Logger: log}},
		{"no Logger", customer_auth.Config{CredRepo: credRepo, RefreshRepo: refreshRepo, ResetRepo: resetRepo, Hasher: hasher, JWTSvc: jwt, Publisher: pub, Logger: nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := customer_auth.New(tc.cfg)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

// ─── HTTP integration tests ──────────────────────────────────────────────────

func TestHTTP_Register_201(t *testing.T) {
	_, mux, _ := buildModule(t)
	body := `{"end_customer_id":"ec-1","password":"correct-horse-battery"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/t1/customers/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Register_ShortPassword_400(t *testing.T) {
	_, mux, _ := buildModule(t)
	body := `{"end_customer_id":"ec-1","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/t1/customers/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — %s", w.Code, w.Body.String())
	}
}

func TestHTTP_Login_200(t *testing.T) {
	credRepo := newCredRepoWithEmail()
	credRepo.seedWithEmail(&domain.CustomerCredential{
		ID:            "cred-ec-1",
		EndCustomerID: "ec-1",
		TenantID:      "t1",
		PasswordHash:  "hash:strongpassword123",
	}, "user@example.com")

	pub := &stubPublisher{}
	mod, err := customer_auth.New(customer_auth.Config{
		DB:          nil,
		CredRepo:    credRepo,
		RefreshRepo: newStubRefreshRepo(),
		ResetRepo:   newStubResetRepo(),
		Hasher:      &stubHasher{},
		JWTSvc:      &stubJWTSvc{},
		Publisher:   pub,
		Logger:      testLogger(),
	})
	if err != nil {
		t.Fatalf("customer_auth.New: %v", err)
	}
	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	body := `{"email":"user@example.com","password":"strongpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/t1/customers/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["access_token"] == nil || resp["refresh_token"] == nil {
		t.Error("expected access_token and refresh_token in response")
	}
}

func TestHTTP_ForgotPassword_204(t *testing.T) {
	_, mux, _ := buildModule(t)
	body := `{"email":"ghost@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/t1/customers/auth/forgot-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	// Always 204, even for unknown email (enumeration resistance).
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — %s", w.Code, w.Body.String())
	}
}
