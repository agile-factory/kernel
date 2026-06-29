package billing_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/billing"
	"github.com/agile-factory/kernel/platform/billing/domain"
	"github.com/agile-factory/kernel/platform/billing/usecase"
	"log/slog"
	"os"
)

// ─── Stubs ───────────────────────────────────────────────────────────────────

type stubPlanRepo struct {
	plans []*domain.Plan
}

func (r *stubPlanRepo) FindByID(_ context.Context, id string) (*domain.Plan, error) {
	for _, p := range r.plans {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, apperrors.NotFound("plan not found")
}

func (r *stubPlanRepo) FindAll(_ context.Context) ([]*domain.Plan, error) {
	return r.plans, nil
}

type stubSubRepo struct {
	subs map[string]*domain.Subscription
}

func newStubSubRepo(subs ...*domain.Subscription) *stubSubRepo {
	r := &stubSubRepo{subs: map[string]*domain.Subscription{}}
	for _, s := range subs {
		r.subs[s.TenantID] = s
	}
	return r
}

func (r *stubSubRepo) FindByTenantID(_ context.Context, tenantID string) (*domain.Subscription, error) {
	s, ok := r.subs[tenantID]
	if !ok {
		return nil, apperrors.NotFound("subscription not found")
	}
	return s, nil
}

func (r *stubSubRepo) Upsert(_ context.Context, s *domain.Subscription) error {
	cp := *s
	r.subs[s.TenantID] = &cp
	return nil
}

type stubProcessor struct{}

func (p *stubProcessor) CreateCheckoutSession(_ context.Context, _ domain.BillingTenant, _ domain.Plan) (domain.CheckoutResult, error) {
	return domain.CheckoutResult{RedirectURL: "https://checkout.stripe.com/test", ExternalID: "cs_test"}, nil
}

func (p *stubProcessor) HandleWebhook(_ context.Context, _ string, _ []byte) (domain.PaymentEvent, error) {
	return domain.PaymentEvent{Type: domain.PaymentSucceeded, TenantID: "t1", Amount: 999, Currency: "USD"}, nil
}

func (p *stubProcessor) RecordPayment(_ context.Context, _ string, _ string, _ int64, _ string) error {
	return nil
}

type permCheckerStub struct{ allow bool }

func (c *permCheckerStub) Require(_ context.Context, _ permission.Permission) error {
	if c.allow {
		return nil
	}
	return apperrors.PermissionDenied("no permission")
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func testModule(subRepo *stubSubRepo, planRepo *stubPlanRepo, allow bool) *billing.Module {
	mod, err := billing.New(billing.Config{
		SubRepo:   subRepo,
		PlanRepo:  planRepo,
		Processor: &stubProcessor{},
		Checker:   &permCheckerStub{allow: allow},
		Logger:    testLogger(),
	})
	if err != nil {
		panic("testModule: " + err.Error())
	}
	return mod
}

func ctxWithUser(userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = kernelauth.WithUserID(ctx, userID)
	ctx = kernelauth.WithTenantID(ctx, tenantID)
	return ctx
}

// ─── UseCase tests ────────────────────────────────────────────────────────────

func TestListPlans_ReturnsList(t *testing.T) {
	planRepo := &stubPlanRepo{plans: []*domain.Plan{
		{ID: "plan-1", Name: "free", PriceCents: 0, Currency: "USD", Interval: "MONTHLY"},
		{ID: "plan-2", Name: "starter", PriceCents: 999, Currency: "USD", Interval: "MONTHLY"},
	}}
	uc := usecase.NewListPlansUseCase(planRepo)

	out, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(out.Plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(out.Plans))
	}
}

func TestGetSubscription_Existing(t *testing.T) {
	sub := &domain.Subscription{
		ID:                 "sub-1",
		TenantID:           "t1",
		PlanID:             "plan-1",
		Status:             domain.StatusActive,
		CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		PaymentMethod:      "STRIPE",
	}
	subRepo := newStubSubRepo(sub)
	uc := usecase.NewGetSubscriptionUseCase(subRepo)

	out, err := uc.Execute(context.Background(), usecase.GetSubscriptionInput{TenantID: "t1"})
	if err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if out.Subscription.Status != domain.StatusActive {
		t.Errorf("expected ACTIVE, got %s", out.Subscription.Status)
	}
}

func TestGetSubscription_NotFound(t *testing.T) {
	subRepo := newStubSubRepo()
	uc := usecase.NewGetSubscriptionUseCase(subRepo)

	_, err := uc.Execute(context.Background(), usecase.GetSubscriptionInput{TenantID: "missing"})
	if err == nil {
		t.Error("expected 404 error")
	}
	if e, ok := err.(*apperrors.AppError); !ok || e.HTTPCode != 404 {
		t.Errorf("expected HTTP 404, got %v", err)
	}
}

// ─── HTTP tests ───────────────────────────────────────────────────────────────

func TestHTTP_ListPlans_Public(t *testing.T) {
	planRepo := &stubPlanRepo{plans: []*domain.Plan{
		{ID: "plan-1", Name: "free", PriceCents: 0, Currency: "USD", Interval: "MONTHLY"},
	}}
	mod := testModule(newStubSubRepo(), planRepo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/plans", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1 plan, got %d", len(resp))
	}
	if resp[0]["name"] != "free" {
		t.Errorf("expected name=free, got %v", resp[0]["name"])
	}
}

func TestHTTP_GetSubscription_Existing(t *testing.T) {
	sub := &domain.Subscription{
		ID:                 "sub-1",
		TenantID:           "t1",
		PlanID:             "plan-1",
		Status:             domain.StatusActive,
		CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		PaymentMethod:      "STRIPE",
	}
	subRepo := newStubSubRepo(sub)
	planRepo := &stubPlanRepo{}
	mod := testModule(subRepo, planRepo, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/tenants/t1/subscription", nil)
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
	if resp["status"] != "ACTIVE" {
		t.Errorf("expected status=ACTIVE, got %v", resp["status"])
	}
}

func TestHTTP_GetSubscription_Unauthenticated(t *testing.T) {
	mod := testModule(newStubSubRepo(), &stubPlanRepo{}, true)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/tenants/t1/subscription", nil)
	// No auth context
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHTTP_GetSubscription_PermissionDenied(t *testing.T) {
	mod := testModule(newStubSubRepo(), &stubPlanRepo{}, false)

	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/api/v1/tenants/t1/subscription", nil)
	req = req.WithContext(ctxWithUser("u1", "t1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestNew_MissingConfig_ReturnsError(t *testing.T) {
	subRepo := newStubSubRepo()
	planRepo := &stubPlanRepo{}
	checker := &permCheckerStub{allow: true}
	proc := &stubProcessor{}
	logger := testLogger()

	cases := []struct {
		name string
		cfg  billing.Config
	}{
		{"no SubRepo", billing.Config{PlanRepo: planRepo, Processor: proc, Checker: checker, Logger: logger}},
		{"no PlanRepo", billing.Config{SubRepo: subRepo, Processor: proc, Checker: checker, Logger: logger}},
		{"no Processor", billing.Config{SubRepo: subRepo, PlanRepo: planRepo, Checker: checker, Logger: logger}},
		{"no Checker", billing.Config{SubRepo: subRepo, PlanRepo: planRepo, Processor: proc, Logger: logger}},
		{"no Logger", billing.Config{SubRepo: subRepo, PlanRepo: planRepo, Processor: proc, Checker: checker}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := billing.New(tc.cfg)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
