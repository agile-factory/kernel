package customer_notifications_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	cn "github.com/agile-factory/kernel/platform/customer_notifications"
	cninfra "github.com/agile-factory/kernel/platform/customer_notifications/infrastructure"
	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/agile-factory/kernel/platform/customer_notifications/usecase"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
)

// ── Stub implementations ─────────────────────────────────────────────────────

type stubLogRepo struct {
	mu   sync.Mutex
	logs []*domain.CustomerNotificationLog
}

func (r *stubLogRepo) Create(ctx context.Context, log *domain.CustomerNotificationLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	log.ID = "log-" + time.Now().Format("150405.000000")
	log.CreatedAt = time.Now()
	r.logs = append(r.logs, log)
	return nil
}

func (r *stubLogRepo) UpdateStatus(_ context.Context, id string, status domain.DeliveryStatus, externalID, failureReason string, sentAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.logs {
		if l.ID == id {
			l.Status = status
			l.ExternalID = externalID
			l.FailureReason = failureReason
			l.SentAt = sentAt
		}
	}
	return nil
}

func (r *stubLogRepo) List(_ context.Context, opts domain.ListLogsOptions) ([]*domain.CustomerNotificationLog, string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.CustomerNotificationLog
	for _, l := range r.logs {
		if l.TenantID == opts.TenantID {
			out = append(out, l)
		}
	}
	return out, "", nil
}

type stubTemplateRepo struct {
	mu        sync.Mutex
	templates map[string]*domain.CustomerTemplate // key: tenantID+"/"+key+"/"+channel+"/"+locale
}

func newStubTemplateRepo() *stubTemplateRepo {
	return &stubTemplateRepo{templates: map[string]*domain.CustomerTemplate{}}
}

func (r *stubTemplateRepo) repoKey(tenantID, key string, ch domain.Channel, locale string) string {
	return tenantID + "/" + key + "/" + string(ch) + "/" + locale
}

func (r *stubTemplateRepo) Upsert(_ context.Context, t *domain.CustomerTemplate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[r.repoKey(t.TenantID, t.Key, t.Channel, t.Locale)] = t
	return nil
}

func (r *stubTemplateRepo) Find(_ context.Context, tenantID, key string, ch domain.Channel, locale string) (*domain.CustomerTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.templates[r.repoKey(tenantID, key, ch, locale)]
	if !ok {
		return nil, domain.ErrTemplateNotFound
	}
	return t, nil
}

func (r *stubTemplateRepo) Delete(_ context.Context, tenantID, key string, ch domain.Channel, locale string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.templates, r.repoKey(tenantID, key, ch, locale))
	return nil
}

func (r *stubTemplateRepo) ListByTenant(_ context.Context, tenantID string) ([]*domain.CustomerTemplate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*domain.CustomerTemplate
	for _, t := range r.templates {
		if t.TenantID == tenantID {
			out = append(out, t)
		}
	}
	return out, nil
}

type stubCustomerResolver struct {
	contact *domain.EndCustomerContact
	err     error
}

func (r *stubCustomerResolver) GetContact(_ context.Context, _, _ string) (*domain.EndCustomerContact, error) {
	return r.contact, r.err
}

type stubBrandResolver struct{}

func (r *stubBrandResolver) GetBrand(_ context.Context, _ string) (*domain.BrandData, error) {
	return &domain.BrandData{AppName: "TestApp"}, nil
}

type stubEmailSender struct {
	mu      sync.Mutex
	sends   []emailSend
	sendErr error
}

type emailSend struct {
	to, subject, body string
}

func (s *stubEmailSender) Send(_ context.Context, to, subject, body string) error {
	if s.sendErr != nil {
		return s.sendErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sends = append(s.sends, emailSend{to, subject, body})
	return nil
}

type permCheckerStub struct {
	allow bool
}

func (c *permCheckerStub) Require(_ context.Context, _ permission.Permission) error {
	if c.allow {
		return nil
	}
	return errors.New("permission denied")
}

type stubSubscriber struct {
	mu       sync.Mutex
	handlers map[string][]events.EventHandler
}

func newStubSubscriber() *stubSubscriber {
	return &stubSubscriber{handlers: map[string][]events.EventHandler{}}
}

func (s *stubSubscriber) Subscribe(eventType string, handler events.EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[eventType] = append(s.handlers[eventType], handler)
}

func (s *stubSubscriber) trigger(ctx context.Context, ev events.Event) error {
	s.mu.Lock()
	hs := s.handlers[ev.Type]
	s.mu.Unlock()
	for _, h := range hs {
		if err := h(ctx, ev); err != nil {
			return err
		}
	}
	return nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func newModule(t *testing.T, logRepo domain.NotificationLogRepository, tmplRepo domain.CustomerTemplateRepository, emailSender domain.EmailSender, allowPerm bool) *cn.Module {
	t.Helper()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	mod, err := cn.New(cn.Config{
		LogRepo:          logRepo,
		TemplateRepo:     tmplRepo,
		CustomerResolver: &stubCustomerResolver{contact: &domain.EndCustomerContact{Email: "user@example.com", FirstName: "Alice"}},
		BrandResolver:    &stubBrandResolver{},
		EmailSender:      emailSender,
		SMSSender:        cninfra.NewNoopSMSAdapter(log),
		Checker:          &permCheckerStub{allow: allowPerm},
		Logger:           log,
	})
	if err != nil {
		t.Fatalf("cn.New: %v", err)
	}
	return mod
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestNew_MissingDeps_Errors(t *testing.T) {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cases := []struct {
		name string
		cfg  cn.Config
	}{
		{"no LogRepo", cn.Config{TemplateRepo: newStubTemplateRepo(), CustomerResolver: &stubCustomerResolver{}, BrandResolver: &stubBrandResolver{}, EmailSender: &stubEmailSender{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Checker: &permCheckerStub{}, Logger: log}},
		{"no TemplateRepo", cn.Config{LogRepo: &stubLogRepo{}, CustomerResolver: &stubCustomerResolver{}, BrandResolver: &stubBrandResolver{}, EmailSender: &stubEmailSender{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Checker: &permCheckerStub{}, Logger: log}},
		{"no CustomerResolver", cn.Config{LogRepo: &stubLogRepo{}, TemplateRepo: newStubTemplateRepo(), BrandResolver: &stubBrandResolver{}, EmailSender: &stubEmailSender{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Checker: &permCheckerStub{}, Logger: log}},
		{"no BrandResolver", cn.Config{LogRepo: &stubLogRepo{}, TemplateRepo: newStubTemplateRepo(), CustomerResolver: &stubCustomerResolver{}, EmailSender: &stubEmailSender{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Checker: &permCheckerStub{}, Logger: log}},
		{"no EmailSender", cn.Config{LogRepo: &stubLogRepo{}, TemplateRepo: newStubTemplateRepo(), CustomerResolver: &stubCustomerResolver{}, BrandResolver: &stubBrandResolver{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Checker: &permCheckerStub{}, Logger: log}},
		{"no SMSSender", cn.Config{LogRepo: &stubLogRepo{}, TemplateRepo: newStubTemplateRepo(), CustomerResolver: &stubCustomerResolver{}, BrandResolver: &stubBrandResolver{}, EmailSender: &stubEmailSender{}, Checker: &permCheckerStub{}, Logger: log}},
		{"no Checker", cn.Config{LogRepo: &stubLogRepo{}, TemplateRepo: newStubTemplateRepo(), CustomerResolver: &stubCustomerResolver{}, BrandResolver: &stubBrandResolver{}, EmailSender: &stubEmailSender{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Logger: log}},
		{"no Logger", cn.Config{LogRepo: &stubLogRepo{}, TemplateRepo: newStubTemplateRepo(), CustomerResolver: &stubCustomerResolver{}, BrandResolver: &stubBrandResolver{}, EmailSender: &stubEmailSender{}, SMSSender: cninfra.NewNoopSMSAdapter(log), Checker: &permCheckerStub{}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := cn.New(tc.cfg)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestRegisterHandlers_SubscribesAllEventTypes(t *testing.T) {
	mod := newModule(t, &stubLogRepo{}, newStubTemplateRepo(), &stubEmailSender{}, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	expected := []string{
		"end_customer.created",
		"customer_auth.registered",
		"customer_auth.password_reset_requested",
		"billing.invoice_paid",
		"billing.payment_reminder",
	}
	for _, evType := range expected {
		if _, ok := sub.handlers[evType]; !ok {
			t.Errorf("expected handler for %q, not found", evType)
		}
	}
}

func TestHandleEvent_Welcome(t *testing.T) {
	logRepo := &stubLogRepo{}
	emailSender := &stubEmailSender{}
	mod := newModule(t, logRepo, newStubTemplateRepo(), emailSender, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	err := sub.trigger(ctx, events.Event{
		Type: "end_customer.created",
		Payload: map[string]any{
			"end_customer_id": "ec-1",
			"tenant_id":       "t-1",
		},
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	emailSender.mu.Lock()
	defer emailSender.mu.Unlock()
	if len(emailSender.sends) == 0 {
		t.Error("expected email to be sent")
	}
	if emailSender.sends[0].to != "user@example.com" {
		t.Errorf("unexpected recipient: %s", emailSender.sends[0].to)
	}

	logRepo.mu.Lock()
	defer logRepo.mu.Unlock()
	if len(logRepo.logs) == 0 {
		t.Error("expected notification log entry")
	}
	if logRepo.logs[0].TemplateKey != "welcome" {
		t.Errorf("unexpected template key: %s", logRepo.logs[0].TemplateKey)
	}
	if logRepo.logs[0].Status != domain.StatusSent {
		t.Errorf("unexpected status: %s", logRepo.logs[0].Status)
	}
}

func TestHandleEvent_OTPReset(t *testing.T) {
	logRepo := &stubLogRepo{}
	emailSender := &stubEmailSender{}
	mod := newModule(t, logRepo, newStubTemplateRepo(), emailSender, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	err := sub.trigger(ctx, events.Event{
		Type: "customer_auth.password_reset_requested",
		Payload: map[string]any{
			"end_customer_id": "ec-2",
			"tenant_id":       "t-1",
			"reset_token":     "123456",
		},
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	logRepo.mu.Lock()
	defer logRepo.mu.Unlock()
	if len(logRepo.logs) == 0 {
		t.Fatal("expected log entry")
	}
	if logRepo.logs[0].TemplateKey != "otp_reset" {
		t.Errorf("expected otp_reset, got %s", logRepo.logs[0].TemplateKey)
	}
}

func TestHandleEvent_InvoicePaid(t *testing.T) {
	logRepo := &stubLogRepo{}
	emailSender := &stubEmailSender{}
	mod := newModule(t, logRepo, newStubTemplateRepo(), emailSender, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	err := sub.trigger(ctx, events.Event{
		Type: "billing.invoice_paid",
		Payload: map[string]any{
			"end_customer_id": "ec-3",
			"tenant_id":       "t-1",
			"amount":          "5000",
			"currency":        "XOF",
		},
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	logRepo.mu.Lock()
	defer logRepo.mu.Unlock()
	if logRepo.logs[0].TemplateKey != "invoice_paid" {
		t.Errorf("expected invoice_paid, got %s", logRepo.logs[0].TemplateKey)
	}
}

func TestHandleEvent_PaymentReminder(t *testing.T) {
	logRepo := &stubLogRepo{}
	emailSender := &stubEmailSender{}
	mod := newModule(t, logRepo, newStubTemplateRepo(), emailSender, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	err := sub.trigger(ctx, events.Event{
		Type: "billing.payment_reminder",
		Payload: map[string]any{
			"end_customer_id": "ec-5",
			"tenant_id":       "t-1",
			"amount":          "2500",
			"currency":        "XOF",
		},
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	logRepo.mu.Lock()
	defer logRepo.mu.Unlock()
	if len(logRepo.logs) == 0 {
		t.Fatal("expected log entry")
	}
	if logRepo.logs[0].TemplateKey != "payment_reminder" {
		t.Errorf("expected payment_reminder, got %s", logRepo.logs[0].TemplateKey)
	}
	if logRepo.logs[0].Status != domain.StatusSent {
		t.Errorf("expected SENT status, got %s", logRepo.logs[0].Status)
	}
}

func TestHandleEvent_MissingIDs_Skipped(t *testing.T) {
	logRepo := &stubLogRepo{}
	mod := newModule(t, logRepo, newStubTemplateRepo(), &stubEmailSender{}, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	// missing end_customer_id — should return nil (not an error, just skip)
	err := sub.trigger(ctx, events.Event{
		Type:    "end_customer.created",
		Payload: map[string]any{"tenant_id": "t-1"},
	})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	logRepo.mu.Lock()
	defer logRepo.mu.Unlock()
	if len(logRepo.logs) != 0 {
		t.Error("expected no log entry when end_customer_id missing")
	}
}

func TestHandleEvent_CustomerNotFound_PropagatesError(t *testing.T) {
	resolver := &stubCustomerResolver{err: domain.ErrEndCustomerNotFound}
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	mod, err := cn.New(cn.Config{
		LogRepo:          &stubLogRepo{},
		TemplateRepo:     newStubTemplateRepo(),
		CustomerResolver: resolver,
		BrandResolver:    &stubBrandResolver{},
		EmailSender:      &stubEmailSender{},
		SMSSender:        cninfra.NewNoopSMSAdapter(log),
		Checker:          &permCheckerStub{allow: true},
		Logger:           log,
	})
	if err != nil {
		t.Fatalf("cn.New: %v", err)
	}
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	triggerErr := sub.trigger(ctx, events.Event{
		Type:    "end_customer.created",
		Payload: map[string]any{"end_customer_id": "ec-missing", "tenant_id": "t-1"},
	})
	if triggerErr == nil {
		t.Error("expected error when customer not found")
	}
}

func TestUpsertTemplate_ValidTemplate(t *testing.T) {
	tmplRepo := newStubTemplateRepo()
	mod := newModule(t, &stubLogRepo{}, tmplRepo, &stubEmailSender{}, true)
	_ = mod // module created; test via use case directly for finer control

	uc := usecase.NewUpsertCustomerTemplateUseCase(tmplRepo, &permCheckerStub{allow: true})
	ctx := context.Background()
	err := uc.Execute(ctx, "t-1", usecase.UpsertCustomerTemplateInput{
		Key:      "welcome",
		Channel:  domain.ChannelEmail,
		Locale:   "fr",
		BodyTmpl: "Bonjour {{.CustomerFirstName}}",
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	tmpl, findErr := tmplRepo.Find(ctx, "t-1", "welcome", domain.ChannelEmail, "fr")
	if findErr != nil {
		t.Fatalf("find: %v", findErr)
	}
	if tmpl.BodyTmpl != "Bonjour {{.CustomerFirstName}}" {
		t.Errorf("unexpected body: %s", tmpl.BodyTmpl)
	}
}

func TestUpsertTemplate_InvalidSyntax(t *testing.T) {
	tmplRepo := newStubTemplateRepo()
	uc := usecase.NewUpsertCustomerTemplateUseCase(tmplRepo, &permCheckerStub{allow: true})
	ctx := context.Background()
	err := uc.Execute(ctx, "t-1", usecase.UpsertCustomerTemplateInput{
		Key:      "welcome",
		Channel:  domain.ChannelEmail,
		Locale:   "fr",
		BodyTmpl: "{{unclosed",
	})
	if !errors.Is(err, domain.ErrTemplateInvalid) {
		t.Errorf("expected ErrTemplateInvalid, got %v", err)
	}
}

func TestUpsertTemplate_UnsupportedKey(t *testing.T) {
	tmplRepo := newStubTemplateRepo()
	uc := usecase.NewUpsertCustomerTemplateUseCase(tmplRepo, &permCheckerStub{allow: true})
	ctx := context.Background()
	err := uc.Execute(ctx, "t-1", usecase.UpsertCustomerTemplateInput{
		Key:      "unknown_key",
		Channel:  domain.ChannelEmail,
		Locale:   "fr",
		BodyTmpl: "body",
	})
	if !errors.Is(err, domain.ErrUnsupportedTemplateKey) {
		t.Errorf("expected ErrUnsupportedTemplateKey, got %v", err)
	}
}

func TestUpsertTemplate_PermissionDenied(t *testing.T) {
	tmplRepo := newStubTemplateRepo()
	uc := usecase.NewUpsertCustomerTemplateUseCase(tmplRepo, &permCheckerStub{allow: false})
	ctx := context.Background()
	err := uc.Execute(ctx, "t-1", usecase.UpsertCustomerTemplateInput{
		Key:      "welcome",
		Channel:  domain.ChannelEmail,
		Locale:   "fr",
		BodyTmpl: "body",
	})
	if err == nil {
		t.Error("expected permission error")
	}
}

func TestDeleteTemplate(t *testing.T) {
	tmplRepo := newStubTemplateRepo()
	ctx := context.Background()
	_ = tmplRepo.Upsert(ctx, &domain.CustomerTemplate{
		TenantID: "t-1", Key: "welcome", Channel: domain.ChannelEmail, Locale: "fr",
		BodyTmpl: "body",
	})

	uc := usecase.NewDeleteCustomerTemplateUseCase(tmplRepo, &permCheckerStub{allow: true})
	if err := uc.Execute(ctx, "t-1", "welcome", domain.ChannelEmail, "fr"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := tmplRepo.Find(ctx, "t-1", "welcome", domain.ChannelEmail, "fr"); !errors.Is(err, domain.ErrTemplateNotFound) {
		t.Error("expected ErrTemplateNotFound after delete")
	}
}

func TestListLogs_ReturnsLogs(t *testing.T) {
	logRepo := &stubLogRepo{}
	ctx := context.Background()
	_ = logRepo.Create(ctx, &domain.CustomerNotificationLog{TenantID: "t-1", Channel: domain.ChannelEmail, Status: domain.StatusSent, TemplateKey: "welcome"})

	uc := usecase.NewListCustomerNotificationLogsUseCase(logRepo, &permCheckerStub{allow: true})
	out, err := uc.Execute(ctx, "t-1", usecase.ListCustomerNotificationLogsInput{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(out.Logs) == 0 {
		t.Error("expected at least one log")
	}
}

func TestListLogs_PermissionDenied(t *testing.T) {
	logRepo := &stubLogRepo{}
	uc := usecase.NewListCustomerNotificationLogsUseCase(logRepo, &permCheckerStub{allow: false})
	_, err := uc.Execute(context.Background(), "t-1", usecase.ListCustomerNotificationLogsInput{})
	if err == nil {
		t.Error("expected permission error")
	}
}

func TestHandleEvent_EmailSendFailure_LogsFailedStatus(t *testing.T) {
	logRepo := &stubLogRepo{}
	emailSender := &stubEmailSender{sendErr: errors.New("smtp timeout")}
	mod := newModule(t, logRepo, newStubTemplateRepo(), emailSender, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	ctx := context.Background()
	err := sub.trigger(ctx, events.Event{
		Type: "end_customer.created",
		Payload: map[string]any{
			"end_customer_id": "ec-fail",
			"tenant_id":       "t-1",
		},
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	logRepo.mu.Lock()
	defer logRepo.mu.Unlock()
	if len(logRepo.logs) == 0 {
		t.Fatal("expected log entry")
	}
	if logRepo.logs[0].Status != domain.StatusFailed {
		t.Errorf("expected FAILED status, got %s", logRepo.logs[0].Status)
	}
	if logRepo.logs[0].FailureReason == "" {
		t.Error("expected failure reason to be set")
	}
}

func TestHandleEvent_TenantOverrideTemplate(t *testing.T) {
	logRepo := &stubLogRepo{}
	emailSender := &stubEmailSender{}
	tmplRepo := newStubTemplateRepo()

	// Pre-seed a tenant override
	ctx := context.Background()
	_ = tmplRepo.Upsert(ctx, &domain.CustomerTemplate{
		TenantID: "t-override",
		Key:      "welcome",
		Channel:  domain.ChannelEmail,
		Locale:   "fr",
		BodyTmpl: "Bienvenue {{.CustomerFirstName}} depuis override!",
	})

	mod := newModule(t, logRepo, tmplRepo, emailSender, true)
	sub := newStubSubscriber()
	mod.RegisterHandlers(sub)

	err := sub.trigger(ctx, events.Event{
		Type: "end_customer.created",
		Payload: map[string]any{
			"end_customer_id": "ec-4",
			"tenant_id":       "t-override",
		},
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}

	emailSender.mu.Lock()
	defer emailSender.mu.Unlock()
	if len(emailSender.sends) == 0 {
		t.Fatal("expected email to be sent")
	}
	if !strings.Contains(emailSender.sends[0].body, "depuis override") {
		t.Errorf("expected override body, got: %s", emailSender.sends[0].body)
	}
}
