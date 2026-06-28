// Package notification is the B2B email-first notification platform.
//
// It implements RFC-0019: handlers subscribe to the outbox (kernel/events)
// for the catalogued event types, render the matching templates per locale,
// and deliver via an EmailSender. Each attempt is logged in notification_logs
// so an operator can audit / replay failures.
//
// Currently the platform exposes no HTTP routes (admin views over
// notification_logs land later). Consumers wire it like so:
//
//	notifMod := notification.New(notification.Config{...})
//	notifMod.Subscribe(bus)             // register handlers
//	go bus.Run(ctx)                     // run dispatcher
package notification

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	notifdomain "github.com/agile-factory/kernel/platform/notification/domain"
	"github.com/google/uuid"
)

// Migrations holds the SQL files this platform owns. Composed by the
// consumer via kernel/migrator alongside the other modules.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Dispatcher is the public producer-side contract — any code that wants
// to send a notification depends on this interface, not on the Module.
type Dispatcher interface {
	Dispatch(ctx context.Context, event Event) error
}

// Event mirrors notifdomain.Event for the public surface. Re-exported so
// callers don't have to import the internal domain package.
type Event = notifdomain.Event

// EventType re-export.
type EventType = notifdomain.EventType

const (
	EventEmailVerificationRequested = notifdomain.EVENT_EMAIL_VERIFICATION_REQUESTED
	EventMemberInvited              = notifdomain.EVENT_MEMBER_INVITED
	EventInviteAccepted             = notifdomain.EVENT_INVITE_ACCEPTED
	EventPasswordResetRequested     = notifdomain.EVENT_PASSWORD_RESET_REQUESTED
)

// Config captures the dependencies + tunables of the notification Module.
type Config struct {
	LogRepo     notifdomain.NotificationLogRepository
	EmailSender EmailSender
	Logger      *slog.Logger

	// From is the From: header on outgoing emails. Required.
	From string
	// BaseURL is the public URL of the SaaS app — used to compose verify /
	// accept links inside templates. Required.
	BaseURL string
	// DefaultLocale falls back to "en" when empty.
	DefaultLocale string
}

// Module is the self-contained notification platform.
type Module struct {
	cfg      Config
	renderer *renderer
}

// New builds the Module. It parses the embedded templates eagerly so any
// template error surfaces at startup, not on the first event.
func New(cfg Config) (*Module, error) {
	if cfg.LogRepo == nil {
		return nil, errors.New("notification.Config: LogRepo is required")
	}
	if cfg.EmailSender == nil {
		return nil, errors.New("notification.Config: EmailSender is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("notification.Config: Logger is required")
	}
	if cfg.From == "" {
		return nil, errors.New("notification.Config: From is required")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("notification.Config: BaseURL is required")
	}
	if cfg.DefaultLocale == "" {
		cfg.DefaultLocale = "en"
	}
	r, err := newRenderer()
	if err != nil {
		return nil, fmt.Errorf("notification: %w", err)
	}
	return &Module{cfg: cfg, renderer: r}, nil
}

// Subscribe wires the outbox event types this module handles. Call this
// before bus.Run so the first batch of pending events finds the handlers.
//
// Mapping (outbox event type → notification EventType + handler):
//
//   - auth.signup_owner    → EmailVerificationRequested
//   - auth.invite_accepted → InviteAccepted (notify owner — currently a no-op stub)
//   - member.invited       → MemberInvited
func (m *Module) Subscribe(bus events.Subscriber) {
	bus.Subscribe("auth.signup_owner", m.handleSignupOwner)
	bus.Subscribe("member.invited", m.handleMemberInvited)
	// auth.invite_accepted is published but the v0.1 template is not yet
	// shipped — register a no-op handler so the event is marked published
	// and doesn't pile up in the outbox.
	bus.Subscribe("auth.invite_accepted", m.handleInviteAccepted)
}

// RegisterRoutes is a no-op for v0.1.0 — no HTTP surface yet.
func (m *Module) RegisterRoutes(_ *http.ServeMux) {}

// Dispatch sends one event directly, skipping the outbox. Useful for code
// paths outside a transaction (e.g. a scheduled job) or for tests.
func (m *Module) Dispatch(ctx context.Context, event Event) error {
	switch event.Type {
	case EventEmailVerificationRequested:
		return m.sendEmail(ctx, event, "email_verification")
	case EventMemberInvited:
		return m.sendEmail(ctx, event, "member_invited")
	case EventInviteAccepted:
		return m.sendEmail(ctx, event, "invite_accepted")
	default:
		return fmt.Errorf("notification: no template wired for %q", event.Type)
	}
}

// ─── Outbox handlers ─────────────────────────────────────────────────────────

func (m *Module) handleSignupOwner(ctx context.Context, ev events.Event) error {
	email, _ := ev.Payload["email"].(string)
	token, _ := ev.Payload["verification_token"].(string)
	if email == "" || token == "" {
		return errors.New("notification: signup_owner event missing email or verification_token")
	}
	tenantName, _ := ev.Payload["tenant_name"].(string)
	if tenantName == "" {
		tenantName = "your workspace"
	}
	return m.Dispatch(ctx, Event{
		Type:    EventEmailVerificationRequested,
		Email:   email,
		Locale:  m.cfg.DefaultLocale,
		Payload: map[string]any{
			"TenantName": tenantName,
			"VerifyURL":  m.cfg.BaseURL + "/verify-email?token=" + token,
		},
	})
}

func (m *Module) handleMemberInvited(ctx context.Context, ev events.Event) error {
	email, _ := ev.Payload["email"].(string)
	token, _ := ev.Payload["token"].(string)
	role, _ := ev.Payload["role"].(string)
	if email == "" || token == "" {
		return errors.New("notification: member.invited event missing email or token")
	}
	tenantName, _ := ev.Payload["tenant_name"].(string)
	if tenantName == "" {
		tenantName = "the workspace"
	}
	inviterName, _ := ev.Payload["inviter_name"].(string)
	if inviterName == "" {
		inviterName = "A workspace admin"
	}
	expiresAt, _ := ev.Payload["expires_at"].(string)

	return m.Dispatch(ctx, Event{
		Type:     EventMemberInvited,
		Email:    email,
		TenantID: ev.Meta.TenantID,
		Locale:   m.cfg.DefaultLocale,
		Payload: map[string]any{
			"TenantName":  tenantName,
			"InviterName": inviterName,
			"Role":        role,
			"AcceptURL":   m.cfg.BaseURL + "/accept-invite?token=" + token,
			"ExpiresAt":   expiresAt,
		},
	})
}

// handleInviteAccepted notifies the inviter / tenant owner when an invite is
// consumed. The event payload from AcceptInviteUseCase carries:
//
//	invitee_name, invitee_email — the new member
//	inviter_id, inviter_email  — who sent the invite
//	inviter_name               — display name (may be empty → falls back to email prefix)
//	role                       — the membership role
//	tenant_name                — workspace display name
func (m *Module) handleInviteAccepted(ctx context.Context, ev events.Event) error {
	inviterEmail, _ := ev.Payload["inviter_email"].(string)
	if inviterEmail == "" {
		// No inviter to notify (platform-onboarded tenant — no invite chain).
		return nil
	}
	inviteeName, _ := ev.Payload["invitee_name"].(string)
	inviteeEmail, _ := ev.Payload["invitee_email"].(string)
	inviterName, _ := ev.Payload["inviter_name"].(string)
	if inviterName == "" {
		for i, c := range inviterEmail {
			if c == '@' {
				inviterName = inviterEmail[:i]
				break
			}
		}
	}
	role, _ := ev.Payload["role"].(string)
	tenantName, _ := ev.Payload["tenant_name"].(string)
	if tenantName == "" {
		tenantName = "your workspace"
	}
	inviterID, _ := ev.Payload["inviter_id"].(string)

	return m.Dispatch(ctx, Event{
		Type:        EventInviteAccepted,
		Email:       inviterEmail,
		RecipientID: inviterID,
		TenantID:    ev.Meta.TenantID,
		Locale:      m.cfg.DefaultLocale,
		Payload: map[string]any{
			"InviteeName":  inviteeName,
			"InviteeEmail": inviteeEmail,
			"InviterName":  inviterName,
			"Role":         role,
			"TenantName":   tenantName,
			"TeamURL":      m.cfg.BaseURL + "/team",
		},
	})
}

// ─── Internal ─────────────────────────────────────────────────────────────────

func (m *Module) sendEmail(ctx context.Context, event Event, templateName string) error {
	subject, text, html, err := m.renderer.render(event.Locale, templateName, event.Payload)
	if err != nil {
		return err
	}

	logRow := notifdomain.NewPending(
		uuid.NewString(),
		event.TenantID,
		event.RecipientID,
		event.Type,
		notifdomain.CHANNEL_EMAIL,
		event.Payload,
	)
	if err := m.cfg.LogRepo.Save(ctx, logRow); err != nil {
		// Log persistence failure shouldn't block delivery, but we surface
		// the error so the outbox retries (the row will be re-created on
		// retry; this is acceptable since notification_logs is an append-
		// only audit, not a state machine).
		m.cfg.Logger.Warn("notification: log persist failed", "error", err)
	}

	msg := EmailMessage{
		From:    m.cfg.From,
		To:      event.Email,
		Subject: subject,
		Text:    text,
		HTML:    html,
	}
	externalID, sendErr := m.cfg.EmailSender.Send(ctx, msg)
	if sendErr != nil {
		logRow.MarkFailed(sendErr.Error())
		if err := m.cfg.LogRepo.Update(ctx, logRow); err != nil {
			m.cfg.Logger.Warn("notification: log update (failed) failed", "error", err)
		}
		return sendErr
	}
	logRow.MarkSent(externalID)
	if err := m.cfg.LogRepo.Update(ctx, logRow); err != nil {
		m.cfg.Logger.Warn("notification: log update (sent) failed", "error", err)
	}
	m.cfg.Logger.Info("notification: email sent",
		"event_type", event.Type,
		"to", event.Email,
		"external_id", externalID,
		"sent_at", time.Now().UTC(),
	)
	return nil
}
