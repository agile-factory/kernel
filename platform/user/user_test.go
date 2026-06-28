package user_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/user"
	"github.com/agile-factory/kernel/platform/user/domain"
	"github.com/agile-factory/kernel/platform/user/usecase"
	"log/slog"
	"os"
)

// ─── Stubs ───────────────────────────────────────────────────────────────────

type stubRepo struct {
	profiles map[string]*domain.UserProfile
}

func newStubRepo(profiles ...*domain.UserProfile) *stubRepo {
	r := &stubRepo{profiles: map[string]*domain.UserProfile{}}
	for _, p := range profiles {
		r.profiles[p.UserID] = p
	}
	return r
}

func (r *stubRepo) FindByUserID(_ context.Context, userID string) (*domain.UserProfile, error) {
	p, ok := r.profiles[userID]
	if !ok {
		return nil, apperrors.NotFound("user profile not found")
	}
	return p, nil
}

func (r *stubRepo) Upsert(_ context.Context, p *domain.UserProfile) error {
	cp := *p
	r.profiles[p.UserID] = &cp
	return nil
}

// stubChecker allows or denies all permissions based on the allow flag.
type stubChecker struct {
	allow bool
}

func (c *stubChecker) Require(_ context.Context, _ interface{ String() string }) error {
	if c.allow {
		return nil
	}
	return apperrors.PermissionDenied("no permission")
}

// fakeSubscriber records Subscribe calls.
type fakeSubscriber struct {
	handlers map[string]events.EventHandler
}

func (f *fakeSubscriber) Subscribe(eventType string, handler events.EventHandler) {
	f.handlers[eventType] = handler
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func ptr(s string) *string { return &s }

// ─── UseCase tests ───────────────────────────────────────────────────────────

func TestGetProfile_ExistingProfile(t *testing.T) {
	profile := &domain.UserProfile{
		UserID:      "u1",
		DisplayName: "Alice",
		Locale:      "fr-SN",
		Timezone:    "Africa/Dakar",
		UpdatedAt:   time.Now(),
	}
	repo := newStubRepo(profile)
	uc := usecase.NewGetProfileUseCase(repo)

	out, err := uc.Execute(context.Background(), usecase.GetProfileInput{UserID: "u1"})
	if err != nil {
		t.Fatalf("GetProfile existing: %v", err)
	}
	if out.Profile.DisplayName != "Alice" {
		t.Errorf("expected DisplayName=Alice, got %s", out.Profile.DisplayName)
	}
}

func TestGetProfile_AbsentProfile_LazyCreate(t *testing.T) {
	repo := newStubRepo()
	uc := usecase.NewGetProfileUseCase(repo)

	out, err := uc.Execute(context.Background(), usecase.GetProfileInput{
		UserID: "u2",
		Email:  "u2@example.com",
	})
	if err != nil {
		t.Fatalf("GetProfile lazy-create: %v", err)
	}
	if out.Profile.DisplayName != "u2" {
		t.Errorf("expected DisplayName=u2 (email prefix), got %s", out.Profile.DisplayName)
	}
	// Verify it was persisted
	if _, err := repo.FindByUserID(context.Background(), "u2"); err != nil {
		t.Error("profile should have been persisted by lazy-create")
	}
}

func TestGetProfile_AbsentProfile_NoEmail_ReturnsNotFound(t *testing.T) {
	repo := newStubRepo()
	uc := usecase.NewGetProfileUseCase(repo)

	_, err := uc.Execute(context.Background(), usecase.GetProfileInput{UserID: "u3"})
	if err == nil {
		t.Error("expected 404 error when profile absent and no email provided")
	}
	var appErr *apperrors.AppError
	if e, ok := err.(*apperrors.AppError); !ok || e.HTTPCode != 404 {
		t.Errorf("expected HTTP 404, got %v", appErr)
	}
}

func TestUpdateProfile_UpdatesFields(t *testing.T) {
	profile := &domain.UserProfile{
		UserID:      "u1",
		DisplayName: "Alice",
		Locale:      "fr-SN",
		Timezone:    "Africa/Dakar",
		UpdatedAt:   time.Now(),
	}
	repo := newStubRepo(profile)
	uc := usecase.NewUpdateProfileUseCase(repo)

	out, err := uc.Execute(context.Background(), usecase.UpdateProfileInput{
		UserID:      "u1",
		DisplayName: ptr("Alice Smith"),
		Bio:         ptr("Software engineer"),
	})
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if out.Profile.DisplayName != "Alice Smith" {
		t.Errorf("expected DisplayName=Alice Smith, got %s", out.Profile.DisplayName)
	}
	if out.Profile.Bio != "Software engineer" {
		t.Errorf("expected Bio updated, got %s", out.Profile.Bio)
	}
	// Untouched fields should be preserved
	if out.Profile.Locale != "fr-SN" {
		t.Errorf("Locale should be unchanged, got %s", out.Profile.Locale)
	}
}

func TestUpdateProfile_NilFields_IsNoop(t *testing.T) {
	profile := &domain.UserProfile{
		UserID:      "u1",
		DisplayName: "Alice",
		Bio:         "Original bio",
		Locale:      "en-US",
		Timezone:    "UTC",
		UpdatedAt:   time.Now(),
	}
	repo := newStubRepo(profile)
	uc := usecase.NewUpdateProfileUseCase(repo)

	out, err := uc.Execute(context.Background(), usecase.UpdateProfileInput{
		UserID: "u1",
		// all pointer fields nil
	})
	if err != nil {
		t.Fatalf("UpdateProfile noop: %v", err)
	}
	if out.Profile.DisplayName != "Alice" {
		t.Errorf("DisplayName should be unchanged, got %s", out.Profile.DisplayName)
	}
	if out.Profile.Bio != "Original bio" {
		t.Errorf("Bio should be unchanged, got %s", out.Profile.Bio)
	}
}

// ─── Module tests ─────────────────────────────────────────────────────────────

func testModule(repo *stubRepo, allow bool) *user.Module {
	// We need a real permission.Checker — use the permission package
	// with a stub membership repo that allows everything when allow=true.
	// Instead, wire the module with a real checker backed by our stub.
	// Since permission.New needs a membership.Repository, we use a helper.
	checker := &permCheckerStub{allow: allow}
	mod, err := user.New(user.Config{
		Repo:    repo,
		Checker: checker,
		Logger:  testLogger(),
	})
	if err != nil {
		panic("testModule: " + err.Error())
	}
	return mod
}

// permCheckerStub satisfies permission.Checker for tests.
type permCheckerStub struct{ allow bool }

func (c *permCheckerStub) Require(_ context.Context, _ permission.Permission) error {
	if c.allow {
		return nil
	}
	return apperrors.PermissionDenied("no permission in test")
}

func ctxWithUser(userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = kernelauth.WithUserID(ctx, userID)
	ctx = kernelauth.WithTenantID(ctx, tenantID)
	return ctx
}

func TestHTTP_GetProfile_Existing(t *testing.T) {
	profile := &domain.UserProfile{
		UserID:      "u1",
		DisplayName: "Bob",
		Locale:      "en-US",
		Timezone:    "UTC",
		UpdatedAt:   time.Now(),
	}
	repo := newStubRepo(profile)
	mod := testModule(repo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/users/u1/profile", nil)
	req = req.WithContext(ctxWithUser("u1", "t1"))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["displayName"] != "Bob" {
		t.Errorf("expected displayName=Bob, got %v", resp["displayName"])
	}
}

func TestHTTP_GetProfile_NotFound(t *testing.T) {
	repo := newStubRepo()
	mod := testModule(repo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/users/unknown/profile", nil)
	req = req.WithContext(ctxWithUser("u1", "t1"))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHTTP_GetProfile_PermissionDenied(t *testing.T) {
	repo := newStubRepo()
	mod := testModule(repo, false) // checker denies

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/users/u1/profile", nil)
	req = req.WithContext(ctxWithUser("u1", "t1"))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestHTTP_PatchProfile_OwnProfile(t *testing.T) {
	profile := &domain.UserProfile{
		UserID:      "u1",
		DisplayName: "Old Name",
		Locale:      "fr-SN",
		Timezone:    "Africa/Dakar",
		UpdatedAt:   time.Now(),
	}
	repo := newStubRepo(profile)
	mod := testModule(repo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	body := `{"displayName": "New Name", "bio": "Hello world"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/u1/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(ctxWithUser("u1", "t1"))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["displayName"] != "New Name" {
		t.Errorf("expected displayName=New Name, got %v", resp["displayName"])
	}
}

func TestHTTP_PatchProfile_OtherUser_Forbidden(t *testing.T) {
	repo := newStubRepo()
	mod := testModule(repo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	body := `{"displayName": "Hacked"}`
	req := httptest.NewRequest("PATCH", "/api/v1/users/other-user/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// callerID = "u1" but path userId = "other-user"
	req = req.WithContext(ctxWithUser("u1", "t1"))
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 when patching another user's profile, got %d", w.Code)
	}
}

func TestHTTP_GetProfile_Unauthenticated(t *testing.T) {
	repo := newStubRepo()
	mod := testModule(repo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/users/u1/profile", nil)
	// No auth context set
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestSubscribe_SignupOwner_CreatesProfile(t *testing.T) {
	repo := newStubRepo()
	mod := testModule(repo, true)

	bus := &fakeSubscriber{handlers: map[string]events.EventHandler{}}
	mod.Subscribe(bus)

	handler, ok := bus.handlers["auth.signup_owner"]
	if !ok {
		t.Fatal("auth.signup_owner handler not registered")
	}

	err := handler(context.Background(), events.Event{
		Type: "auth.signup_owner",
		Payload: map[string]any{
			"user_id": "new-user-id",
			"email":   "owner@example.com",
		},
	})
	if err != nil {
		t.Fatalf("handleSignupOwner: %v", err)
	}

	// Profile should have been created
	p, err := repo.FindByUserID(context.Background(), "new-user-id")
	if err != nil {
		t.Fatalf("profile not created: %v", err)
	}
	if p.DisplayName != "owner" {
		t.Errorf("expected DisplayName=owner (email prefix), got %s", p.DisplayName)
	}
}

func TestNew_MissingConfig_ReturnsError(t *testing.T) {
	repo := newStubRepo()
	checker := &permCheckerStub{allow: true}
	logger := testLogger()

	cases := []struct {
		name string
		cfg  user.Config
	}{
		{"no Repo", user.Config{Checker: checker, Logger: logger}},
		{"no Checker", user.Config{Repo: repo, Logger: logger}},
		{"no Logger", user.Config{Repo: repo, Checker: checker}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := user.New(tc.cfg)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
