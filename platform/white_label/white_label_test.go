package white_label_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/white_label"
	"github.com/agile-factory/kernel/platform/white_label/domain"
	"github.com/agile-factory/kernel/platform/white_label/usecase"
)

// ─── Stubs ────────────────────────────────────────────────────────────────────

type stubBrandRepo struct {
	mu      sync.Mutex
	records map[string]*domain.BrandSettings // keyed by tenantID
}

func newStubBrandRepo(brands ...*domain.BrandSettings) *stubBrandRepo {
	r := &stubBrandRepo{records: map[string]*domain.BrandSettings{}}
	for _, b := range brands {
		cp := *b
		r.records[b.TenantID] = &cp
	}
	return r
}

func (r *stubBrandRepo) Upsert(_ context.Context, b *domain.BrandSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *b
	r.records[b.TenantID] = &cp
	return nil
}

func (r *stubBrandRepo) FindByTenantID(_ context.Context, tenantID string) (*domain.BrandSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.records[tenantID]
	if !ok {
		return nil, domain.ErrBrandNotFound
	}
	cp := *b
	return &cp, nil
}

func (r *stubBrandRepo) FindByCustomDomain(_ context.Context, d string) (*domain.BrandSettings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range r.records {
		if b.CustomDomain == d && b.CustomDomainStatus == domain.DomainVerified {
			cp := *b
			return &cp, nil
		}
	}
	return nil, domain.ErrBrandNotFound
}

func (r *stubBrandRepo) UpdateDomainStatus(_ context.Context, tenantID string, status domain.DomainVerificationStatus, verifiedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.records[tenantID]
	if !ok {
		return domain.ErrBrandNotFound
	}
	b.CustomDomainStatus = status
	b.CustomDomainVerifiedAt = verifiedAt
	return nil
}

// stubPublisher records events.
type stubPublisher struct {
	mu     sync.Mutex
	events []events.Event
}

func (p *stubPublisher) Publish(_ context.Context, ev events.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, ev)
	return nil
}

// permCheckerStub allows or denies all permissions.
type permCheckerStub struct {
	allow bool
}

func (c *permCheckerStub) Require(_ context.Context, _ permission.Permission) error {
	if c.allow {
		return nil
	}
	return apperrors.PermissionDenied("no permission")
}

// stubDNSResolver returns configured TXT records.
type stubDNSResolver struct {
	records []string
	err     error
}

func (r *stubDNSResolver) LookupTXT(_ context.Context, _ string) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.records, nil
}

func ctxWithIdentity(tenantID, userID string) context.Context {
	ctx := kernelauth.WithTenantID(context.Background(), tenantID)
	return kernelauth.WithUserID(ctx, userID)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func buildTestModule(t *testing.T, repo domain.BrandRepository, pub events.Publisher, dns usecase.DNSResolver) (*white_label.Module, *http.ServeMux) {
	t.Helper()
	mod, err := white_label.New(white_label.Config{
		DB:          nil,
		Repo:        repo,
		Checker:     &permCheckerStub{allow: true},
		Publisher:   pub,
		Logger:      testLogger(),
		DNSResolver: dns,
	})
	if err != nil {
		t.Fatalf("white_label.New: %v", err)
	}
	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)
	return mod, mux
}

// ─── Use case tests ──────────────────────────────────────────────────────────

func TestGetBrandSettings_DefaultsWhenAbsent(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewGetBrandSettingsUseCase(repo, checker)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	out, err := uc.Execute(ctx)
	if err != nil {
		t.Fatalf("GetBrandSettings: %v", err)
	}
	if out.Brand == nil {
		t.Fatal("expected non-nil brand")
	}
	if out.Brand.TenantID != "tenant-1" {
		t.Errorf("expected tenantID tenant-1, got %s", out.Brand.TenantID)
	}
	if out.Brand.PrimaryColor != "#000000" {
		t.Errorf("expected default primary color, got %s", out.Brand.PrimaryColor)
	}
	_ = pub // suppress unused warning
}

func TestUpdateBrandSettings_Success(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewUpdateBrandSettingsUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	brand, err := uc.Execute(ctx, usecase.UpdateBrandSettingsInput{
		AppName:      "MonApp",
		PrimaryColor: "#FF5733",
	})
	if err != nil {
		t.Fatalf("UpdateBrandSettings: %v", err)
	}
	if brand.AppName != "MonApp" {
		t.Errorf("expected AppName MonApp, got %s", brand.AppName)
	}
	if brand.PrimaryColor != "#FF5733" {
		t.Errorf("expected PrimaryColor #FF5733, got %s", brand.PrimaryColor)
	}
	pub.mu.Lock()
	evCount := len(pub.events)
	pub.mu.Unlock()
	if evCount != 1 {
		t.Errorf("expected 1 event, got %d", evCount)
	}
	if pub.events[0].Type != "white_label.brand_updated" {
		t.Errorf("expected white_label.brand_updated event, got %s", pub.events[0].Type)
	}
}

func TestUpdateBrandSettings_InvalidColor(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewUpdateBrandSettingsUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	_, err := uc.Execute(ctx, usecase.UpdateBrandSettingsInput{
		PrimaryColor: "notacolor",
	})
	if err == nil {
		t.Fatal("expected error for invalid color")
	}
}

func TestUpdateBrandSettings_CustomDomainGeneratesToken(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewUpdateBrandSettingsUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	brand, err := uc.Execute(ctx, usecase.UpdateBrandSettingsInput{
		CustomDomain: "app.mycompany.sn",
	})
	if err != nil {
		t.Fatalf("UpdateBrandSettings: %v", err)
	}
	if brand.CustomDomainStatus != domain.DomainPending {
		t.Errorf("expected PENDING status, got %s", brand.CustomDomainStatus)
	}
	if brand.CustomDomainToken == "" {
		t.Error("expected non-empty token")
	}
}

func TestVerifyCustomDomain_Success(t *testing.T) {
	token := "_kernel-verify=test-token-123"
	b := &domain.BrandSettings{
		TenantID:           "tenant-1",
		CustomDomain:       "app.mycompany.sn",
		CustomDomainStatus: domain.DomainPending,
		CustomDomainToken:  token,
		PrimaryColor:       "#000000",
		SecondaryColor:     "#FFFFFF",
		AccentColor:        "#0066FF",
		FontFamily:         "inter",
	}
	repo := newStubBrandRepo(b)
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	dns := &stubDNSResolver{records: []string{token}}
	uc := usecase.NewVerifyCustomDomainUseCase(nil, repo, checker, pub, dns)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	if err := uc.Execute(ctx); err != nil {
		t.Fatalf("VerifyCustomDomain: %v", err)
	}

	stored, _ := repo.FindByTenantID(ctx, "tenant-1")
	if stored.CustomDomainStatus != domain.DomainVerified {
		t.Errorf("expected VERIFIED, got %s", stored.CustomDomainStatus)
	}
	if stored.CustomDomainVerifiedAt == nil {
		t.Error("expected non-nil verified_at")
	}
	pub.mu.Lock()
	evType := pub.events[0].Type
	pub.mu.Unlock()
	if evType != "white_label.domain_verified" {
		t.Errorf("expected white_label.domain_verified event, got %s", evType)
	}
}

func TestVerifyCustomDomain_Timeout(t *testing.T) {
	b := &domain.BrandSettings{
		TenantID:           "tenant-1",
		CustomDomain:       "app.mycompany.sn",
		CustomDomainStatus: domain.DomainPending,
		CustomDomainToken:  "_kernel-verify=token",
		PrimaryColor:       "#000000",
		SecondaryColor:     "#FFFFFF",
		AccentColor:        "#0066FF",
		FontFamily:         "inter",
	}
	repo := newStubBrandRepo(b)
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	dns := &stubDNSResolver{err: context.DeadlineExceeded}
	uc := usecase.NewVerifyCustomDomainUseCase(nil, repo, checker, pub, dns)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	err := uc.Execute(ctx)
	if err == nil {
		t.Fatal("expected error on DNS timeout")
	}

	stored, _ := repo.FindByTenantID(ctx, "tenant-1")
	if stored.CustomDomainStatus != domain.DomainTimeout {
		t.Errorf("expected TIMEOUT, got %s", stored.CustomDomainStatus)
	}
}

func TestVerifyCustomDomain_Mismatch(t *testing.T) {
	b := &domain.BrandSettings{
		TenantID:           "tenant-1",
		CustomDomain:       "app.mycompany.sn",
		CustomDomainStatus: domain.DomainPending,
		CustomDomainToken:  "_kernel-verify=correct-token",
		PrimaryColor:       "#000000",
		SecondaryColor:     "#FFFFFF",
		AccentColor:        "#0066FF",
		FontFamily:         "inter",
	}
	repo := newStubBrandRepo(b)
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	dns := &stubDNSResolver{records: []string{"_kernel-verify=wrong-token"}}
	uc := usecase.NewVerifyCustomDomainUseCase(nil, repo, checker, pub, dns)

	ctx := ctxWithIdentity("tenant-1", "user-1")
	err := uc.Execute(ctx)
	if err == nil {
		t.Fatal("expected error on token mismatch")
	}

	stored, _ := repo.FindByTenantID(ctx, "tenant-1")
	if stored.CustomDomainStatus != domain.DomainFailed {
		t.Errorf("expected FAILED, got %s", stored.CustomDomainStatus)
	}
}

// ─── Module wiring tests ──────────────────────────────────────────────────────

func TestNew_MissingDeps_Errors(t *testing.T) {
	cases := []struct {
		name string
		cfg  white_label.Config
	}{
		{"no Repo", white_label.Config{Repo: nil, Checker: &permCheckerStub{}, Publisher: &stubPublisher{}, Logger: testLogger()}},
		{"no Checker", white_label.Config{Repo: newStubBrandRepo(), Checker: nil, Publisher: &stubPublisher{}, Logger: testLogger()}},
		{"no Publisher", white_label.Config{Repo: newStubBrandRepo(), Checker: &permCheckerStub{}, Publisher: nil, Logger: testLogger()}},
		{"no Logger", white_label.Config{Repo: newStubBrandRepo(), Checker: &permCheckerStub{}, Publisher: &stubPublisher{}, Logger: nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := white_label.New(tc.cfg)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

// ─── HTTP handler integration tests ──────────────────────────────────────────

func TestHTTP_GetBrand_200(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	dns := &stubDNSResolver{}
	_, mux := buildTestModule(t, repo, pub, dns)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t1/brand", nil)
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["primaryColor"] == nil {
		t.Error("expected primaryColor in response")
	}
}

func TestHTTP_UpdateBrand_200(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	dns := &stubDNSResolver{}
	_, mux := buildTestModule(t, repo, pub, dns)

	body := `{"appName":"TestApp","primaryColor":"#123456"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/t1/brand", bytes.NewBufferString(body))
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_UpdateBrand_BadJSON_400(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	dns := &stubDNSResolver{}
	_, mux := buildTestModule(t, repo, pub, dns)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/t1/brand", bytes.NewBufferString("not-json"))
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHTTP_UpdateBrand_InvalidColor_400(t *testing.T) {
	repo := newStubBrandRepo()
	pub := &stubPublisher{}
	dns := &stubDNSResolver{}
	_, mux := buildTestModule(t, repo, pub, dns)

	body := `{"primaryColor":"notacolor"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/t1/brand", bytes.NewBufferString(body))
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_VerifyDomain_204(t *testing.T) {
	token := "_kernel-verify=valid-token"
	b := &domain.BrandSettings{
		TenantID:           "t1",
		CustomDomain:       "app.test.sn",
		CustomDomainStatus: domain.DomainPending,
		CustomDomainToken:  token,
		PrimaryColor:       "#000000",
		SecondaryColor:     "#FFFFFF",
		AccentColor:        "#0066FF",
		FontFamily:         "inter",
	}
	repo := newStubBrandRepo(b)
	pub := &stubPublisher{}
	dns := &stubDNSResolver{records: []string{token}}
	_, mux := buildTestModule(t, repo, pub, dns)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/t1/brand/verify-domain", nil)
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_GetPublicBrand_200_ByTenantID(t *testing.T) {
	b := &domain.BrandSettings{
		TenantID:           "t1",
		AppName:            "PublicApp",
		PrimaryColor:       "#ABCDEF",
		SecondaryColor:     "#FFFFFF",
		AccentColor:        "#0066FF",
		FontFamily:         "inter",
		CustomDomainStatus: domain.DomainNotSet,
	}
	repo := newStubBrandRepo(b)
	pub := &stubPublisher{}
	dns := &stubDNSResolver{}
	_, mux := buildTestModule(t, repo, pub, dns)

	req := httptest.NewRequest(http.MethodGet, "/public/v1/brand?tenant_id=t1", nil)
	// No auth context for public endpoint.
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["appName"] != "PublicApp" {
		t.Errorf("expected appName PublicApp, got %v", resp["appName"])
	}
	// Public response must NOT include sensitive fields.
	if _, hasToken := resp["customDomainToken"]; hasToken {
		t.Error("public response must not include customDomainToken")
	}
}
