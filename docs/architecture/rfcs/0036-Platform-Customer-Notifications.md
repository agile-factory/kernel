# _RFC #0036:_ Platform `customer_notifications` — Notifications Clients Finaux B2B2C

**date:** `2026 Jun 28`

---

## **Status:** `ACCEPTED`

> **Supersède (sections impactées) :** [RFC-0011 — Platform Notification Multi-Canal](0011-Platform-Notification-Multi-Canal.md) *(canal SMS end-customer)*

> **Review Aziz** : dispatcher proprement séparé de `platform/notification` (staff). Port `EmailSender` partagé avec RFC-0019 est une bonne décision. Les colonnes `tenant_settings` ajoutées (`sms_provider`, etc.) doivent être dans `customer_notifications/migrations/` — pas dans `notification/migrations/`.
> **Review Lucas** : le `NoopSMSAdapter` doit être l'adapter par défaut en dev (`DEFAULT 'NOOP'` dans tenant_settings) — conforme. S'assurer que les templates `text/template` sont parsés au démarrage du module (not lazily) pour détecter les erreurs de syntaxe au boot.

---

## **Summary**

Ce RFC définit le module `platform/customer_notifications` : système de notifications orienté **end-customers** (RFC-0033).

Différences fondamentales avec `platform/notification` (RFC-0019, staff-facing) :

| Dimension | `platform/notification` | `platform/customer_notifications` |
|-----------|------------------------|-----------------------------------|
| Audience | Members du workspace (staff) | End-customers (clients finaux) |
| Canaux | Email seul | Email + **SMS** (CEDEAO-first) |
| Templates | Globaux (plateforme) | Per-tenant (brandés white_label) |
| Trigger | Events internes plateforme | Events B2B2C (invoice_paid, welcome, OTP) |
| Branding | Logo plateforme | Logo tenant (RFC-0035) |

---

## **Motivation**

### Problème 1 — SMS indispensable en CEDEAO

Dans les marchés CEDEAO (Sénégal, Côte d'Ivoire, Mali, Burkina…), une fraction significative des end-customers n'a pas d'adresse email active mais utilise un téléphone mobile. Le SMS est le canal de notification de facto pour :
- Confirmations de paiement Wave/Orange Money
- OTP reset password (RFC-0034)
- Quittances locatives, reçus de commande
- Rappels d'échéance

RFC-0019 couvre le canal email pour le staff. Ce RFC ajoute le canal SMS pour les end-customers.

### Problème 2 — Templates per-tenant

En B2B2C, chaque tenant veut envoyer des emails avec **son** logo, **ses** couleurs et **son** domaine d'expéditeur (`notifications@monentreprise.sn`). `platform/notification` utilise des templates globaux non brandables.

### Problème 3 — Consommation des événements end_customer

Les événements outbox émis par RFC-0033 (`end_customer.created`, `billing.invoice_paid`, etc.) doivent déclencher des notifications. Il faut un dispatcher qui mappe event_type → template → canal.

---

## **Proposal**

### 1. Modèle domaine

```go
// platform/customer_notifications/domain/notification.go
package domain

import "time"

type Channel string

const (
    ChannelEmail Channel = "EMAIL"
    ChannelSMS   Channel = "SMS"
)

type DeliveryStatus string

const (
    StatusPending   DeliveryStatus = "PENDING"
    StatusSent      DeliveryStatus = "SENT"
    StatusFailed    DeliveryStatus = "FAILED"
    StatusBounced   DeliveryStatus = "BOUNCED"
)

type CustomerNotificationLog struct {
    ID              string
    TenantID        string
    EndCustomerID   string
    Channel         Channel
    Recipient       string        // email ou numéro E.164
    TemplateKey     string        // ex: "invoice_paid", "welcome", "otp_reset"
    Subject         string        // email seulement
    Body            string        // HTML pour email, texte pour SMS
    Status          DeliveryStatus
    ExternalID      string        // ID retourné par le provider (SES message ID, SMS ref)
    FailureReason   string
    SentAt          *time.Time
    CreatedAt       time.Time
}
```

```go
// platform/customer_notifications/domain/template.go
package domain

// Template per-tenant, stocké en DB (override des defaults globaux).
type CustomerTemplate struct {
    ID          string
    TenantID    string
    Key         string    // "welcome", "invoice_paid", "otp_reset", "payment_reminder"
    Channel     Channel
    Locale      string    // "fr" | "en" | "wo" (wolof)
    SubjectTmpl string    // text/template pour email
    BodyTmpl    string    // text/template (HTML email ou texte SMS)
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### 2. Port Sender (canal abstrait)

```go
// platform/customer_notifications/domain/sender.go
package domain

import "context"

// EmailSender — même port que RFC-0019, réutilisé.
type EmailSender interface {
    Send(ctx context.Context, to, subject, htmlBody string) error
}

// SMSSender — nouveau port.
type SMSSender interface {
    Send(ctx context.Context, to, body string) (externalID string, err error)
}
```

### 3. Adapters SMS

| Adapter | Provider | Marché | Mécanique |
|---------|---------|--------|-----------|
| `OrangeSMSAdapter` | Orange SMS Business API | CEDEAO 15 pays | REST + HMAC |
| `TwilioSMSAdapter` | Twilio | International | REST |
| `AfricasTalkingAdapter` | Africa's Talking | Afrique subsaharienne | REST |
| `NoopSMSAdapter` | — | Tous | Log seulement (dev/test) |

Sélection via `tenant_settings.sms_provider` (`ORANGE` | `TWILIO` | `AFRICAS_TALKING` | `NOOP`).

Le port `SMSSender` permet de swapper le provider sans changer les use cases.

### 4. Dispatcher d'événements

```go
// platform/customer_notifications/dispatcher.go
package customernotifications

// Mappe event_type → handler de notification
var eventHandlers = map[string]HandlerFunc{
    "end_customer.created":       handleWelcome,
    "billing.invoice_paid":       handleInvoicePaid,
    "billing.payment_reminder":   handlePaymentReminder,
    "customer_auth.registered":   handleCustomerAuthWelcome,
    "customer_auth.password_reset_requested": handleOTPReset,
}
```

Le dispatcher s'abonne aux événements outbox via `events.Dispatcher.Subscribe(eventType, handler)`.

### 5. Use cases

#### `SendCustomerNotificationUseCase`

```
1. Charger EndCustomer (pour email et phone)
2. Charger template : tenant custom si existe, sinon template global du fichier embed
3. Résoudre locale : tenant_settings.locale, sinon "fr"
4. Appliquer branding white_label (RFC-0035) : logo, primary_color dans le template email
5. Rendu text/template avec les données du payload
6. Appeler EmailSender.Send() ou SMSSender.Send() selon le canal
7. INSERT customer_notification_logs {status: SENT | FAILED}
8. Si échec : retry policy (max 3, backoff exponentiel) via event_outbox
```

#### `UpsertCustomerTemplateUseCase`

```
1. Permission : OWNER ou ADMIN
2. Valider : key dans liste des keys supportées, locale valide, template parseable
3. Upsert dans customer_templates
```

#### `ListCustomerNotificationLogsUseCase`

```
1. Permission : OWNER ou ADMIN
2. Cursor-pagination sur (created_at DESC, id)
3. Filtres : end_customer_id, channel, status, template_key
```

### 6. Templates par défaut (embed.FS)

```
platform/customer_notifications/templates/
├── fr/
│   ├── welcome.email.html
│   ├── welcome.sms.txt
│   ├── invoice_paid.email.html
│   ├── invoice_paid.sms.txt
│   ├── payment_reminder.email.html
│   ├── payment_reminder.sms.txt
│   ├── otp_reset.email.html
│   └── otp_reset.sms.txt
└── en/
    ├── welcome.email.html
    ├── invoice_paid.email.html
    └── otp_reset.email.html
```

Variables disponibles dans les templates :
```
{{.CustomerFirstName}}   {{.CustomerLastName}}
{{.TenantAppName}}       {{.TenantLogoURL}}
{{.TenantPrimaryColor}}  {{.SupportEmail}}
{{.InvoiceAmount}}       {{.InvoiceCurrency}}
{{.OTPCode}}             {{.ResetLink}}
```

### 7. Schéma SQL

```sql
-- platform/customer_notifications/migrations/0001_create_customer_notification_logs.up.sql

CREATE TABLE customer_notification_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    end_customer_id  UUID REFERENCES end_customers(id) ON DELETE SET NULL,
    channel          VARCHAR(10) NOT NULL CHECK (channel IN ('EMAIL','SMS')),
    recipient        VARCHAR(255) NOT NULL,
    template_key     VARCHAR(100) NOT NULL,
    subject          VARCHAR(500),
    body             TEXT NOT NULL,
    status           VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                         CHECK (status IN ('PENDING','SENT','FAILED','BOUNCED')),
    external_id      VARCHAR(255),
    failure_reason   TEXT,
    sent_at          TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cn_logs_tenant_customer
    ON customer_notification_logs(tenant_id, end_customer_id, created_at DESC);

CREATE INDEX idx_cn_logs_status
    ON customer_notification_logs(status, created_at)
    WHERE status = 'PENDING';
```

```sql
-- platform/customer_notifications/migrations/0002_create_customer_templates.up.sql

CREATE TABLE customer_templates (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key          VARCHAR(100) NOT NULL,
    channel      VARCHAR(10) NOT NULL CHECK (channel IN ('EMAIL','SMS')),
    locale       VARCHAR(10) NOT NULL DEFAULT 'fr',
    subject_tmpl VARCHAR(500),
    body_tmpl    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, key, channel, locale)
);
```

```sql
-- tenant_settings : ajout SMS provider
ALTER TABLE tenant_settings
    ADD COLUMN sms_provider     VARCHAR(30) NOT NULL DEFAULT 'NOOP'
                   CHECK (sms_provider IN ('ORANGE','TWILIO','AFRICAS_TALKING','NOOP')),
    ADD COLUMN sms_sender_name  VARCHAR(11),   -- alphanumeric sender ID (max 11 chars)
    ADD COLUMN email_from_name  VARCHAR(100),  -- "NomApp Support"
    ADD COLUMN email_from_addr  VARCHAR(255);  -- notifications@domaine-tenant.com
```

### 8. API HTTP

```
# Templates (OWNER/ADMIN)
GET    /api/v1/tenants/{id}/customer-templates               ListCustomerTemplates
PUT    /api/v1/tenants/{id}/customer-templates/{key}/{chan}   UpsertCustomerTemplate
DELETE /api/v1/tenants/{id}/customer-templates/{key}/{chan}   DeleteCustomerTemplate (revert to default)

# Logs (OWNER/ADMIN)
GET    /api/v1/tenants/{id}/customer-notification-logs       ListCustomerNotificationLogs (cursor)
```

---

## **Drawbacks**

1. **Coût SMS** : les SMS ont un coût variable par opérateur. Pas de rate-limiting ou de budget par tenant dans cette v1. À ajouter en P2 (RFC-0026 Usage Metering).
2. **Templates HTML email** : rendu correct dans les clients mail est complexe (Outlook, Gmail). Les templates par défaut sont délibérément simples (table-based). Les tenants qui veulent des templates riches doivent les fournir eux-mêmes.
3. **Deux systèmes de notification** : `platform/notification` (staff) et `platform/customer_notifications` (end-customers). Risque de duplication de logique. Mitigation : le port `EmailSender` est partagé.

---

## **Alternatives**

### Étendre `platform/notification` avec un flag `audience: STAFF | CUSTOMER`

- ✅ Une seule table, un seul dispatcher
- ❌ Templates per-tenant non supportés dans RFC-0019
- ❌ Canal SMS couplé dans le module staff
- **Verdict : rejeté** — séparation nette reflète la différence fonctionnelle

### Provider SMS unique (Twilio seulement)

- ✅ Intégration plus simple
- ❌ Pas de support des numéros CEDEAO locaux (Orange, Free, MTN) à des tarifs compétitifs
- **Verdict : rejeté** — le port `SMSSender` permet d'ajouter des adapters sans refactor

---

## **Plan d'adoption**

1. RFC-0033 et RFC-0035 acceptés (end_customers + white_label pour le branding des templates)
2. Review Aziz + Lucas
3. Implémenter : migrations → domain → adapters SMS (Noop en premier) → usecase → dispatcher → delivery → tests
4. Intégrer dans examples avec `B2B2C_ENABLED`
5. ADR post-implémentation

---

## **References**

- [RFC-0033 — Platform end_customers](0033-Platform-End-Customers.md)
- [RFC-0034 — Platform customer_auth](0034-Platform-Customer-Auth.md) *(OTP reset password)*
- [RFC-0035 — Platform white_label](0035-Platform-White-Label.md) *(branding templates)*
- [RFC-0019 — Platform Notification B2B](0019-Platform-Notification-B2B-Email-First.md) *(port EmailSender partagé)*
- [RFC-0015 — Event Bus Outbox](0015-Event-Bus-Interne.md) *(déclenchement notifications)*
- [RFC-0020 — Platform Billing Merchant Mode](0020-Platform-Billing-Hybride-Merchant-Mode.md) *(invoice_paid event)*
- [RFC-0011 — Platform Notification Multi-Canal](0011-Platform-Notification-Multi-Canal.md) *(supersédé pour canal SMS)*
