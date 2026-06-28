package end_customers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/end_customers"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
	"github.com/agile-factory/kernel/platform/end_customers/usecase"
)

// ─── Stubs ────────────────────────────────────────────────────────────────────

// stubRepo is an in-memory EndCustomerRepository for unit tests.
type stubRepo struct {
	records map[string]*domain.EndCustomer // keyed by id
}

func newStubRepo(ecs ...*domain.EndCustomer) *stubRepo {
	r := &stubRepo{records: map[string]*domain.EndCustomer{}}
	for _, ec := range ecs {
		cp := *ec
		r.records[ec.ID] = &cp
	}
	return r
}

func (r *stubRepo) Create(_ context.Context, ec *domain.EndCustomer) error {
	// check email uniqueness (simplified)
	for _, existing := range r.records {
		if existing.Email != "" && existing.Email == ec.Email && existing.TenantID == ec.TenantID {
			return domain.ErrEndCustomerConflict
		}
	}
	cp := *ec
	r.records[ec.ID] = &cp
	return nil
}

func (r *stubRepo) FindByID(_ context.Context, tenantID, id string) (*domain.EndCustomer, error) {
	ec, ok := r.records[id]
	if !ok || ec.TenantID != tenantID || ec.DeletedAt != nil {
		return nil, domain.ErrEndCustomerNotFound
	}
	cp := *ec
	return &cp, nil
}

func (r *stubRepo) FindByEmail(_ context.Context, tenantID, email string) (*domain.EndCustomer, error) {
	for _, ec := range r.records {
		if ec.TenantID == tenantID && ec.Email == email && ec.DeletedAt == nil {
			cp := *ec
			return &cp, nil
		}
	}
	return nil, domain.ErrEndCustomerNotFound
}

func (r *stubRepo) FindByExternalID(_ context.Context, tenantID, extID string) (*domain.EndCustomer, error) {
	for _, ec := range r.records {
		if ec.TenantID == tenantID && ec.ExternalID == extID && ec.DeletedAt == nil {
			cp := *ec
			return &cp, nil
		}
	}
	return nil, domain.ErrEndCustomerNotFound
}

func (r *stubRepo) Update(_ context.Context, ec *domain.EndCustomer) error {
	if _, ok := r.records[ec.ID]; !ok {
		return domain.ErrEndCustomerNotFound
	}
	cp := *ec
	r.records[ec.ID] = &cp
	return nil
}

func (r *stubRepo) SoftDelete(_ context.Context, tenantID, id string) error {
	ec, ok := r.records[id]
	if !ok || ec.TenantID != tenantID || ec.DeletedAt != nil {
		return domain.ErrEndCustomerNotFound
	}
	now := time.Now().UTC()
	ec.Status = domain.StatusDeleted
	ec.DeletedAt = &now
	return nil
}

func (r *stubRepo) List(_ context.Context, params domain.ListParams) ([]*domain.EndCustomer, string, error) {
	var result []*domain.EndCustomer
	for _, ec := range r.records {
		if ec.TenantID != params.TenantID || ec.DeletedAt != nil {
			continue
		}
		if params.Status != nil && ec.Status != *params.Status {
			continue
		}
		cp := *ec
		result = append(result, &cp)
	}
	return result, "", nil
}

func (r *stubRepo) Count(_ context.Context, tenantID string, status *domain.EndCustomerStatus) (int64, error) {
	var n int64
	for _, ec := range r.records {
		if ec.TenantID != tenantID || ec.DeletedAt != nil {
			continue
		}
		if status != nil && ec.Status != *status {
			continue
		}
		n++
	}
	return n, nil
}

// permCheckerStub grants or denies all permissions.
type permCheckerStub struct{ allow bool }

func (p *permCheckerStub) Require(_ context.Context, _ permission.Permission) error {
	if p.allow {
		return nil
	}
	return apperrors.PermissionDenied("test: permission denied")
}

// stubPublisher records published events.
type stubPublisher struct {
	events []events.Event
}

func (p *stubPublisher) Publish(_ context.Context, ev events.Event) error {
	p.events = append(p.events, ev)
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// ctxWithIdentity injects tenantID and userID into context (mirrors kernel/auth middleware).
func ctxWithIdentity(tenantID, userID string) context.Context {
	ctx := context.Background()
	ctx = kernelauth.WithTenantID(ctx, tenantID)
	ctx = kernelauth.WithUserID(ctx, userID)
	return ctx
}

func sampleEC(id, tenantID string) *domain.EndCustomer {
	return &domain.EndCustomer{
		ID:        id,
		TenantID:  tenantID,
		Email:     "alice@example.com",
		FirstName: "Alice",
		LastName:  "Sow",
		Status:    domain.StatusActive,
		Tags:      []string{"vip"},
		Metadata:  map[string]string{"country": "SN"},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}

// ─── Domain model tests ───────────────────────────────────────────────────────

func TestEndCustomer_FullName(t *testing.T) {
	ec := &domain.EndCustomer{FirstName: "Cheikh", LastName: "Nakamoto"}
	if got := ec.FullName(); got != "Cheikh Nakamoto" {
		t.Errorf("FullName: got %q", got)
	}
}

func TestEndCustomer_Suspend_FromActive(t *testing.T) {
	ec := &domain.EndCustomer{Status: domain.StatusActive}
	if err := ec.Suspend(); err != nil {
		t.Fatalf("Suspend active: %v", err)
	}
	if ec.Status != domain.StatusSuspended {
		t.Errorf("expected SUSPENDED, got %s", ec.Status)
	}
}

func TestEndCustomer_Suspend_FromSuspended_Errors(t *testing.T) {
	ec := &domain.EndCustomer{Status: domain.StatusSuspended}
	if err := ec.Suspend(); err == nil {
		t.Error("expected error suspending already-suspended customer")
	}
}

func TestEndCustomer_Restore_FromSuspended(t *testing.T) {
	ec := &domain.EndCustomer{Status: domain.StatusSuspended}
	if err := ec.Restore(); err != nil {
		t.Fatalf("Restore suspended: %v", err)
	}
	if ec.Status != domain.StatusActive {
		t.Errorf("expected ACTIVE, got %s", ec.Status)
	}
}

func TestEndCustomer_Restore_FromActive_Errors(t *testing.T) {
	ec := &domain.EndCustomer{Status: domain.StatusActive}
	if err := ec.Restore(); err == nil {
		t.Error("expected error restoring active customer")
	}
}

// ─── Use case tests (no DB) ───────────────────────────────────────────────────

func TestCreateEndCustomer_Success(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}

	uc := usecase.NewCreateEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	out, err := uc.Execute(ctx, usecase.CreateEndCustomerInput{
		Email:     "bob@example.com",
		FirstName: "Bob",
		LastName:  "Diallo",
		Tags:      []string{"customer"},
		Metadata:  map[string]string{"region": "dakar"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.EndCustomer.ID == "" {
		t.Error("expected non-empty ID")
	}
	if out.EndCustomer.Status != domain.StatusActive {
		t.Errorf("expected ACTIVE, got %s", out.EndCustomer.Status)
	}
	// event published
	if len(pub.events) != 1 || pub.events[0].Type != "end_customer.created" {
		t.Errorf("expected end_customer.created event, got %v", pub.events)
	}
}

func TestCreateEndCustomer_MissingFirstName_Errors(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewCreateEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	_, err := uc.Execute(ctx, usecase.CreateEndCustomerInput{
		Email:    "bob@example.com",
		LastName: "Diallo",
	})
	if err == nil {
		t.Error("expected error when first_name missing")
	}
}

func TestCreateEndCustomer_NoEmailNoPhone_Errors(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewCreateEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	_, err := uc.Execute(ctx, usecase.CreateEndCustomerInput{
		FirstName: "Bob",
		LastName:  "Diallo",
	})
	if err == nil {
		t.Error("expected error when both email and phone missing")
	}
}

func TestCreateEndCustomer_MetadataTooManyKeys_Errors(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewCreateEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	meta := make(map[string]string, 21)
	for i := range 21 {
		meta[string(rune('a'+i))] = "v"
	}
	_, err := uc.Execute(ctx, usecase.CreateEndCustomerInput{
		Email:     "bob@example.com",
		FirstName: "Bob",
		LastName:  "Diallo",
		Metadata:  meta,
	})
	if err == nil {
		t.Error("expected error when metadata exceeds 20 keys")
	}
}

func TestCreateEndCustomer_PermissionDenied(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: false}
	uc := usecase.NewCreateEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	_, err := uc.Execute(ctx, usecase.CreateEndCustomerInput{
		Email:     "bob@example.com",
		FirstName: "Bob",
		LastName:  "Diallo",
	})
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPCode != http.StatusForbidden {
		t.Errorf("expected 403, got %v", err)
	}
}

func TestListEndCustomers_Success(t *testing.T) {
	ec1 := sampleEC("id-1", "tenant-1")
	ec2 := sampleEC("id-2", "tenant-1")
	ec2.Email = "bob@example.com"
	repo := newStubRepo(ec1, ec2)
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewListEndCustomersUseCase(repo, checker)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	out, err := uc.Execute(ctx, usecase.ListEndCustomersInput{Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(out.Items))
	}
}

func TestSuspendEndCustomer_Success(t *testing.T) {
	ec := sampleEC("id-1", "tenant-1")
	repo := newStubRepo(ec)
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewSuspendEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	if err := uc.Execute(ctx, "id-1"); err != nil {
		t.Fatalf("Suspend: %v", err)
	}

	stored, _ := repo.FindByID(ctx, "tenant-1", "id-1")
	if stored.Status != domain.StatusSuspended {
		t.Errorf("expected SUSPENDED after suspend, got %s", stored.Status)
	}
	if len(pub.events) != 1 || pub.events[0].Type != "end_customer.suspended" {
		t.Errorf("expected end_customer.suspended event, got %v", pub.events)
	}
}

func TestRestoreEndCustomer_AfterSuspend(t *testing.T) {
	ec := sampleEC("id-1", "tenant-1")
	ec.Status = domain.StatusSuspended
	repo := newStubRepo(ec)
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewRestoreEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	if err := uc.Execute(ctx, "id-1"); err != nil {
		t.Fatalf("Restore: %v", err)
	}

	stored, _ := repo.FindByID(ctx, "tenant-1", "id-1")
	if stored.Status != domain.StatusActive {
		t.Errorf("expected ACTIVE after restore, got %s", stored.Status)
	}
}

func TestDeleteEndCustomer_Success(t *testing.T) {
	ec := sampleEC("id-1", "tenant-1")
	repo := newStubRepo(ec)
	pub := &stubPublisher{}
	checker := &permCheckerStub{allow: true}
	uc := usecase.NewDeleteEndCustomerUseCase(nil, repo, checker, pub)

	ctx := ctxWithIdentity("tenant-1", "actor-1")
	if err := uc.Execute(ctx, "id-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := repo.FindByID(ctx, "tenant-1", "id-1")
	if !errors.Is(err, domain.ErrEndCustomerNotFound) {
		t.Error("expected ErrEndCustomerNotFound after soft-delete")
	}
	if len(pub.events) != 1 || pub.events[0].Type != "end_customer.deleted" {
		t.Errorf("expected end_customer.deleted event, got %v", pub.events)
	}
}

// ─── HTTP handler integration tests ──────────────────────────────────────────

// buildTestModule wires a Module backed by stubs.
// db is nil because stubs bypass txctx.ExecInTx (publisher is also stubbed).
func buildTestModule(t *testing.T, repo domain.EndCustomerRepository, pub events.Publisher) (*end_customers.Module, *http.ServeMux) {
	t.Helper()
	mod, err := end_customers.New(end_customers.Config{
		DB:        nil, // stubs bypass ExecInTx
		Repo:      repo,
		Checker:   &permCheckerStub{allow: true},
		Publisher: pub,
		Logger:    testLogger(),
	})
	if err != nil {
		t.Fatalf("end_customers.New: %v", err)
	}
	mux := http.NewServeMux()
	mod.RegisterRoutes(mux)
	return mod, mux
}

func TestHTTP_Create_201(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	_, mux := buildTestModule(t, repo, pub)

	body := `{"firstName":"Alice","lastName":"Sow","email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/end-customers", bytes.NewBufferString(body))
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["id"] == "" || resp["id"] == nil {
		t.Error("expected non-empty id in response")
	}
}

func TestHTTP_Create_BadJSON_400(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	_, mux := buildTestModule(t, repo, pub)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/end-customers", bytes.NewBufferString("not-json"))
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHTTP_Get_404(t *testing.T) {
	repo := newStubRepo()
	pub := &stubPublisher{}
	_, mux := buildTestModule(t, repo, pub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/end-customers/nonexistent", nil)
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHTTP_List_200(t *testing.T) {
	ec := sampleEC("id-1", "t1")
	repo := newStubRepo(ec)
	pub := &stubPublisher{}
	_, mux := buildTestModule(t, repo, pub)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/end-customers", nil)
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	items, _ := resp["items"].([]any)
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestHTTP_Delete_NoContent(t *testing.T) {
	ec := sampleEC("id-1", "t1")
	repo := newStubRepo(ec)
	pub := &stubPublisher{}
	_, mux := buildTestModule(t, repo, pub)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/end-customers/id-1", nil)
	req = req.WithContext(ctxWithIdentity("t1", "u1"))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d — body: %s", w.Code, w.Body.String())
	}
}

// ─── Module wiring tests ──────────────────────────────────────────────────────

func TestNew_MissingDeps_Errors(t *testing.T) {
	cases := []struct {
		name string
		cfg  end_customers.Config
	}{
		{"no Repo", end_customers.Config{DB: nil, Repo: nil, Checker: &permCheckerStub{}, Publisher: &stubPublisher{}, Logger: testLogger()}},
		{"no Checker", end_customers.Config{DB: nil, Repo: newStubRepo(), Checker: nil, Publisher: &stubPublisher{}, Logger: testLogger()}},
		{"no Publisher", end_customers.Config{DB: nil, Repo: newStubRepo(), Checker: &permCheckerStub{}, Publisher: nil, Logger: testLogger()}},
		{"no Logger", end_customers.Config{DB: nil, Repo: newStubRepo(), Checker: &permCheckerStub{}, Publisher: &stubPublisher{}, Logger: nil}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := end_customers.New(tc.cfg)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}
