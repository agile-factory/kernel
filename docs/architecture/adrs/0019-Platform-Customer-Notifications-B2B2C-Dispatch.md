# ADR-0019: Platform Customer Notifications — B2B2C Notification Dispatch

**Status:** Accepted  
**RFC:** RFC-0036  
**Date:** 2026-06-28  
**Authors:** Aziz (review), Lucas (review)

---

## Context

The kernel needs a notification dispatch platform for B2B2C end-customers: transactional emails and SMS triggered by domain events (welcome, OTP reset, billing). Multiple tenants share the platform but require independent template overrides and branded content.

RFC-0036 specifies: per-tenant template overrides, event-driven dispatch via the outbox bus, support for EMAIL and SMS channels, cursor-paginated audit logs, and embedded default templates.

---

## Decisions

### 1. Event handlers stored on Module instance — not global vars (Aziz amendment)

Handlers are registered via `RegisterHandlers(sub events.Subscriber)` on the `*Module` instance:

```go
func (m *Module) RegisterHandlers(sub events.Subscriber) {
    for _, evType := range m.eventTypes {
        evType := evType  // capture loop variable
        sub.Subscribe(evType, func(ctx context.Context, ev events.Event) error {
            return m.handleEvent(ctx, ev)
        })
    }
}
```

A global mutable `map[string]EventHandler` would be a package-level shared state, making modules non-recomposable and tests racy. Instance-scoped handlers align with RFC-0038.

### 2. Default templates parsed at boot — not lazily (Lucas amendment)

`New(cfg Config)` calls `parseDefaultTemplates()` which walks the embedded `templates/` FS via `fs.WalkDir` and parses every file with `text/template`. A template syntax error at boot aborts startup with a clear message rather than failing silently at first send time.

Tenant override templates are also validated at `UpsertTemplate` time before persistence.

### 3. Notification log body intentionally omitted (Lucas amendment)

`CustomerNotificationLog` has no `Body` field. OTP codes, reset links, and invoice amounts must not be stored in plaintext in the database. The log records who was notified, via which channel, with which template, and the delivery outcome — but never the rendered content.

### 4. Default channel preference: EMAIL over SMS

When no channel is explicitly specified, `SendCustomerNotificationUseCase` prefers EMAIL (if `contact.Email != ""`), falls back to SMS (if `contact.Phone != ""`), and defaults to EMAIL otherwise. This avoids sending SMS charges for end-customers who have an email address.

### 5. Tenant template override lookup before default embedded template

`renderTemplate` calls `templateRepo.Find(...)` first. On `ErrTemplateNotFound` it falls back to the embedded default keyed by `locale/key.channel.ext` (e.g. `fr/otp_reset.email.html`). The two-tier lookup allows tenants to customise any template without patching the kernel.

### 6. Permission model

- `CustomerNotificationView` (`customer_notification:view`) — required to list dispatch logs.
- `CustomerTemplateWrite` (`customer_template:write`) — required to upsert or delete a template override.

Event-driven dispatch (`SendCustomerNotificationUseCase.Execute`) is not permission-gated — it runs as an internal system actor triggered by the outbox bus.

### 7. Migrations: no body column, three schema files

- `0001`: `customer_notification_logs` — no `body` column per Lucas amendment.
- `0002`: `customer_templates` — `UNIQUE(tenant_id, key, channel, locale)` enforced in DB.
- `0003`: `tenant_settings` extension — adds `sms_provider`, `sms_sender_name`, `email_from_name`, `email_from_addr` for per-tenant sender identity.

All migrations have paired `.down.sql` files.

### 8. NoopSMSAdapter for dev/test

`cninfra.NewNoopSMSAdapter(log)` is the default SMS adapter — it logs sends to slog and returns `"noop-sms-"+to` as the external ID. Production integrations (Twilio, Orange SMS) are injected via `Config.SMSSender`.

---

## Consequences

- All tenants share the embedded default templates; overrides are tenant-scoped and stored in `customer_templates`.
- Delivery audit is complete (channel, recipient, status, external provider ID, failure reason, sent_at) but message content is never stored.
- Adding a new event type requires: a new `SupportedTemplateKeys` entry, a new `case` in `handleEvent`, a new template file under `templates/`, and appending to `m.eventTypes` in `New`.
- The `NoopSMSAdapter` means SMS is always available in tests without mocking; real SMS is a config-time dependency.
