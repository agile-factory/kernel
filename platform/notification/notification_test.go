package notification_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"os"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/notification"
	notifdomain "github.com/agile-factory/kernel/platform/notification/domain"
	"github.com/google/uuid"
)

// ─── Stubs ───────────────────────────────────────────────────────────────────

type stubLogRepo struct {
	saved   []*notifdomain.NotificationLog
	updated []*notifdomain.NotificationLog
}

func (r *stubLogRepo) Save(_ context.Context, l *notifdomain.NotificationLog) error {
	// capture a copy so mutations by MarkSent/MarkFailed don't affect saved state
	cp := *l
	r.saved = append(r.saved, &cp)
	return nil
}
func (r *stubLogRepo) Update(_ context.Context, l *notifdomain.NotificationLog) error {
	cp := *l
	r.updated = append(r.updated, &cp)
	return nil
}
func (r *stubLogRepo) ListByTenant(_ context.Context, _ string, _, _ int) ([]*notifdomain.NotificationLog, error) {
	return nil, nil
}

type stubSender struct {
	sent     []notification.EmailMessage
	failNext bool
}

func (s *stubSender) Send(_ context.Context, msg notification.EmailMessage) (string, error) {
	if s.failNext {
		s.failNext = false
		return "", errors.New("smtp: server unavailable")
	}
	s.sent = append(s.sent, msg)
	return "stub-" + uuid.NewString(), nil
}

func testModule(repo *stubLogRepo, sender *stubSender) *notification.Module {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	mod, err := notification.New(notification.Config{
		LogRepo:     repo,
		EmailSender: sender,
		Logger:      logger,
		From:        "noreply@example.com",
		BaseURL:     "https://app.example.com",
	})
	if err != nil {
		panic("testModule: " + err.Error())
	}
	return mod
}

// ─── Renderer (template smoke test) ──────────────────────────────────────────

func TestDispatch_EmailVerification_Renders(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	err := mod.Dispatch(context.Background(), notification.Event{
		Type:   notification.EventEmailVerificationRequested,
		Email:  "user@example.com",
		Locale: "en",
		Payload: map[string]any{
			"TenantName": "Acme Corp",
			"VerifyURL":  "https://app.example.com/verify-email?token=abc123",
		},
	})
	if err != nil {
		t.Fatalf("Dispatch email_verification: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(sender.sent))
	}
	if sender.sent[0].To != "user@example.com" {
		t.Errorf("wrong To: %s", sender.sent[0].To)
	}
	if len(repo.saved) != 1 {
		t.Errorf("expected 1 log row saved, got %d", len(repo.saved))
	}
	if repo.saved[0].Status != notifdomain.STATUS_PENDING {
		t.Errorf("initial status should be PENDING, got %s", repo.saved[0].Status)
	}
	if len(repo.updated) != 1 || repo.updated[0].Status != notifdomain.STATUS_SENT {
		t.Errorf("log row should be SENT after delivery, got %+v", repo.updated)
	}
}

func TestDispatch_MemberInvited_French(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	err := mod.Dispatch(context.Background(), notification.Event{
		Type:   notification.EventMemberInvited,
		Email:  "invitee@example.com",
		Locale: "fr",
		Payload: map[string]any{
			"TenantName":  "Société Dupont",
			"InviterName": "Marie",
			"Role":        "ROLE_AGENT",
			"AcceptURL":   "https://app.example.com/accept-invite?token=xyz",
			"ExpiresAt":   "2026-07-05",
		},
	})
	if err != nil {
		t.Fatalf("Dispatch member_invited fr: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
}

func TestDispatch_InviteAccepted_NotifiesInviter(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	err := mod.Dispatch(context.Background(), notification.Event{
		Type:   notification.EventInviteAccepted,
		Email:  "boss@example.com",
		Locale: "en",
		Payload: map[string]any{
			"InviteeName":  "Alice",
			"InviteeEmail": "alice@example.com",
			"InviterName":  "Bob",
			"Role":         "ROLE_VIEWER",
			"TenantName":   "Acme",
			"TeamURL":      "https://app.example.com/team",
		},
	})
	if err != nil {
		t.Fatalf("Dispatch invite_accepted: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
	if sender.sent[0].To != "boss@example.com" {
		t.Errorf("email should go to inviter, got %s", sender.sent[0].To)
	}
}

func TestDispatch_InviteAccepted_French(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	err := mod.Dispatch(context.Background(), notification.Event{
		Type:   notification.EventInviteAccepted,
		Email:  "patron@exemple.fr",
		Locale: "fr",
		Payload: map[string]any{
			"InviteeName":  "Claire",
			"InviteeEmail": "claire@exemple.fr",
			"InviterName":  "Patron",
			"Role":         "ROLE_ADMIN",
			"TenantName":   "Société Test",
			"TeamURL":      "https://app.example.com/team",
		},
	})
	if err != nil {
		t.Fatalf("Dispatch invite_accepted fr: %v", err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.sent))
	}
}

func TestDispatch_UnknownType_ReturnsError(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	err := mod.Dispatch(context.Background(), notification.Event{
		Type:  "UNKNOWN_EVENT",
		Email: "x@example.com",
	})
	if err == nil {
		t.Error("expected error for unknown event type")
	}
}

func TestDispatch_SenderFailure_LogsAsFailed(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{failNext: true}
	mod := testModule(repo, sender)

	err := mod.Dispatch(context.Background(), notification.Event{
		Type:   notification.EventEmailVerificationRequested,
		Email:  "user@example.com",
		Locale: "en",
		Payload: map[string]any{
			"TenantName": "Test",
			"VerifyURL":  "https://app.example.com/verify-email?token=t",
		},
	})
	if err == nil {
		t.Error("expected error when sender fails")
	}
	// Log row should be marked FAILED
	if len(repo.updated) != 1 || repo.updated[0].Status != notifdomain.STATUS_FAILED {
		t.Errorf("log row should be FAILED, got %+v", repo.updated)
	}
}

// ─── buildMIME smoke test ────────────────────────────────────────────────────

func TestConsoleSender_ReturnsExternalID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	sender := notification.NewConsoleSender(logger)

	id1, err := sender.Send(context.Background(), notification.EmailMessage{
		From: "a@b.com", To: "c@d.com", Subject: "hi", Text: "hello", HTML: "<p>hello</p>",
	})
	if err != nil {
		t.Fatalf("ConsoleSender.Send: %v", err)
	}
	if id1 == "" {
		t.Error("ConsoleSender should return non-empty external ID")
	}

	id2, _ := sender.Send(context.Background(), notification.EmailMessage{
		From: "a@b.com", To: "e@f.com", Subject: "hi2",
	})
	if id1 == id2 {
		t.Error("ConsoleSender should return distinct IDs per send")
	}
}

// ─── New() validation ────────────────────────────────────────────────────────

func TestNew_MissingConfigReturnsError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	repo := &stubLogRepo{}
	sender := &stubSender{}

	cases := []struct {
		name string
		cfg  notification.Config
	}{
		{"no LogRepo", notification.Config{EmailSender: sender, Logger: logger, From: "a@b.com", BaseURL: "http://x"}},
		{"no EmailSender", notification.Config{LogRepo: repo, Logger: logger, From: "a@b.com", BaseURL: "http://x"}},
		{"no Logger", notification.Config{LogRepo: repo, EmailSender: sender, From: "a@b.com", BaseURL: "http://x"}},
		{"no From", notification.Config{LogRepo: repo, EmailSender: sender, Logger: logger, BaseURL: "http://x"}},
		{"no BaseURL", notification.Config{LogRepo: repo, EmailSender: sender, Logger: logger, From: "a@b.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := notification.New(tc.cfg)
			if err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

// ─── Outbox handler integration ──────────────────────────────────────────────

func makeEvent(eventType string, payload map[string]any) events.Event {
	return events.Event{
		Type:    eventType,
		Payload: payload,
	}
}

func TestHandleSignupOwner_SendsVerificationEmail(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	// Simulate what the bus would call
	bus := &fakeSubscriber{handlers: map[string]events.EventHandler{}}
	mod.Subscribe(bus)

	handler := bus.handlers["auth.signup_owner"]
	if handler == nil {
		t.Fatal("handler for auth.signup_owner not registered")
	}
	err := handler(context.Background(), makeEvent("auth.signup_owner", map[string]any{
		"email":              "new@example.com",
		"verification_token": "tok123",
		"tenant_name":        "Acme",
	}))
	if err != nil {
		t.Fatalf("handleSignupOwner: %v", err)
	}
	if len(sender.sent) != 1 || sender.sent[0].To != "new@example.com" {
		t.Errorf("expected email to new@example.com, got %v", sender.sent)
	}
}

func TestHandleInviteAccepted_NoInviter_IsNoop(t *testing.T) {
	repo := &stubLogRepo{}
	sender := &stubSender{}
	mod := testModule(repo, sender)

	bus := &fakeSubscriber{handlers: map[string]events.EventHandler{}}
	mod.Subscribe(bus)

	handler := bus.handlers["auth.invite_accepted"]
	if handler == nil {
		t.Fatal("handler for auth.invite_accepted not registered")
	}
	// Empty inviter_email — should be a no-op (owner-onboarded tenant)
	err := handler(context.Background(), makeEvent("auth.invite_accepted", map[string]any{
		"invitee_email": "alice@example.com",
		"inviter_email": "",
	}))
	if err != nil {
		t.Fatalf("handleInviteAccepted no-op: %v", err)
	}
	if len(sender.sent) != 0 {
		t.Errorf("no email should be sent when inviter_email is empty, got %d", len(sender.sent))
	}
}

// fakeSubscriber captures Subscribe calls for testing outbox handlers.
type fakeSubscriber struct {
	handlers map[string]events.EventHandler
}

func (f *fakeSubscriber) Subscribe(eventType string, handler events.EventHandler) {
	f.handlers[eventType] = handler
}
