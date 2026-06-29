package infrastructure_test

import (
	"context"
	stderrors "errors"
	"sync"
	"testing"
	"time"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/billing/domain"
	"github.com/agile-factory/kernel/platform/billing/infrastructure"
)

// ── in-memory stubs to test behaviour without a live DB ──────────────────────

// stubPlanStore is an in-memory store that satisfies domain.PlanRepository.
type stubPlanStore struct {
	mu    sync.Mutex
	plans map[string]*domain.Plan
}

func newStubPlanStore(plans ...*domain.Plan) *stubPlanStore {
	s := &stubPlanStore{plans: map[string]*domain.Plan{}}
	for _, p := range plans {
		s.plans[p.ID] = p
	}
	return s
}

func (s *stubPlanStore) FindByID(_ context.Context, id string) (*domain.Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.plans[id]
	if !ok {
		return nil, kerrors.NotFound("plan not found")
	}
	return p, nil
}

func (s *stubPlanStore) FindAll(_ context.Context) ([]*domain.Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*domain.Plan, 0, len(s.plans))
	for _, p := range s.plans {
		out = append(out, p)
	}
	return out, nil
}

// stubSubStore is an in-memory store satisfying domain.SubscriptionRepository.
type stubSubStore struct {
	mu   sync.Mutex
	subs map[string]*domain.Subscription
}

func newStubSubStore(subs ...*domain.Subscription) *stubSubStore {
	s := &stubSubStore{subs: map[string]*domain.Subscription{}}
	for _, sub := range subs {
		s.subs[sub.TenantID] = sub
	}
	return s
}

func (s *stubSubStore) FindByTenantID(_ context.Context, tenantID string) (*domain.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subs[tenantID]
	if !ok {
		return nil, kerrors.NotFound("subscription not found")
	}
	return sub, nil
}

func (s *stubSubStore) Upsert(_ context.Context, sub *domain.Subscription) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *sub
	s.subs[sub.TenantID] = &cp
	return nil
}

// ── constructor guard — package compiles ──────────────────────────────────────

// TestNewRepos_PackageCompiles ensures the constructors exist and return non-nil.
// (Real Postgres behaviour is tested via integration tests; nil DB panics on use,
// but construction itself is safe since pgxpool.Pool is stored by pointer.)
func TestNewRepos_PackageCompiles(t *testing.T) {
	if infrastructure.NewPostgresPlanRepository(nil) == nil {
		t.Error("expected non-nil plan repo")
	}
	if infrastructure.NewPostgresSubscriptionRepository(nil) == nil {
		t.Error("expected non-nil subscription repo")
	}
}

// ── stub-based logic tests (no DB) ────────────────────────────────────────────

func TestPlanRepo_FindByID_Found(t *testing.T) {
	plan := &domain.Plan{ID: "p-1", Name: "pro", PriceCents: 2000, Currency: "USD", Interval: "MONTHLY"}
	store := newStubPlanStore(plan)
	p, err := store.FindByID(context.Background(), "p-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "pro" {
		t.Errorf("expected pro, got %s", p.Name)
	}
}

func TestPlanRepo_FindByID_NotFound(t *testing.T) {
	store := newStubPlanStore()
	_, err := store.FindByID(context.Background(), "missing")
	if !isNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestPlanRepo_FindAll_Empty(t *testing.T) {
	store := newStubPlanStore()
	plans, err := store.FindAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 0 {
		t.Errorf("expected 0 plans, got %d", len(plans))
	}
}

func TestPlanRepo_FindAll_ReturnsAll(t *testing.T) {
	store := newStubPlanStore(
		&domain.Plan{ID: "p-1", Name: "free", PriceCents: 0, Currency: "USD", Interval: "MONTHLY"},
		&domain.Plan{ID: "p-2", Name: "pro", PriceCents: 2000, Currency: "USD", Interval: "MONTHLY"},
	)
	plans, err := store.FindAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plans) != 2 {
		t.Errorf("expected 2 plans, got %d", len(plans))
	}
}

func TestSubscriptionRepo_FindByTenantID_Found(t *testing.T) {
	now := time.Now()
	sub := &domain.Subscription{
		ID:                 "sub-1",
		TenantID:           "t-1",
		PlanID:             "p-1",
		Status:             domain.StatusActive,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(30 * 24 * time.Hour),
		PaymentMethod:      "STRIPE",
	}
	store := newStubSubStore(sub)
	got, err := store.FindByTenantID(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.StatusActive {
		t.Errorf("expected ACTIVE, got %s", got.Status)
	}
}

func TestSubscriptionRepo_FindByTenantID_NotFound(t *testing.T) {
	store := newStubSubStore()
	_, err := store.FindByTenantID(context.Background(), "t-missing")
	if !isNotFound(err) {
		t.Errorf("expected not-found error, got %v", err)
	}
}

func TestSubscriptionRepo_Upsert_CreateAndUpdate(t *testing.T) {
	store := newStubSubStore()
	ctx := context.Background()
	now := time.Now()

	sub := &domain.Subscription{
		TenantID:           "t-2",
		PlanID:             "p-1",
		Status:             domain.StatusTrialing,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.Add(14 * 24 * time.Hour),
		PaymentMethod:      "STRIPE",
	}
	if err := store.Upsert(ctx, sub); err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	got, _ := store.FindByTenantID(ctx, "t-2")
	if got.Status != domain.StatusTrialing {
		t.Errorf("expected TRIALING, got %s", got.Status)
	}

	// Update to ACTIVE
	sub.Status = domain.StatusActive
	sub.ExternalID = "stripe-sub-123"
	if err := store.Upsert(ctx, sub); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got, _ = store.FindByTenantID(ctx, "t-2")
	if got.Status != domain.StatusActive {
		t.Errorf("expected ACTIVE after update, got %s", got.Status)
	}
	if got.ExternalID != "stripe-sub-123" {
		t.Errorf("expected external ID set, got %q", got.ExternalID)
	}
}

// isNotFound checks for the kernel 404 error pattern.
func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	var appErr *kerrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.HTTPCode == 404
	}
	return false
}
