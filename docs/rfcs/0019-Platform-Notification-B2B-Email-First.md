# _RFC #0019:_ Platform `notification` — B2B Email-First

**date:** `2026 May 12`

> **Supersède :** [RFC-0011 — Platform Notification Multi-Canal](0011-Platform-Notification-Multi-Canal.md)

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC redéfinit la plateforme `notification` avec une orientation **B2B email-first** pour les **memberships** (utilisateurs internes de l'organisation), en distinguant clairement les canaux selon le profil destinataire.

Architecture orientée événements : les use cases émettent des events dans l'**outbox transactionnel** (RFC-0015 v1.1), `platform/notification` les consomme et dispatche selon le canal configuré.

Canaux par profil :
- **B2B / Memberships** : Email-first (canal pro), webhooks outbound optionnels, SMS/WhatsApp en option (OTP + alertes critiques uniquement)
- **B2B2C / EndCustomers** : Multi-canal d'office (Email + SMS + WhatsApp + Push) — défini dans RFC-0036

---

## **Motivation**

RFC-0011 traitait les canaux SMS/WhatsApp sur le même plan que l'email pour tous les utilisateurs. En B2B, les membres d'une organisation sont des professionnels qui :
- Lisent leurs emails (canal principal et fiable)
- Ne veulent pas de SMS non sollicités de leur outil de travail
- Utilisent éventuellement WhatsApp pour des alertes critiques, jamais pour des notifications routinières

La réorientation permet de :
1. Implémenter la v1 rapidement (email seul) sans bloquer le delivery B2B.
2. Garder SMS/WhatsApp comme **options activables par tenant** (pour OTP et alertes critiques).
3. Préparer le terrain pour RFC-0036 (end-customer multi-canal B2B2C).

---

## **Proposal**

### 1. Architecture — Event-driven via Outbox

```
Use Case → INSERT event_outbox (même transaction métier)
                ↓
         OutboxDispatcher (polling SKIP LOCKED)
                ↓
     NotificationConsumer.Handle(event)
                ↓
     ChannelSelector.Select(event, recipient_prefs)
                ↓
     EmailSender / SMSSender / WebhookSender
```

**Garantie** : l'event de notification est inséré dans la même transaction que l'opération métier. Si la transaction est committée, la notification **sera** envoyée (at-least-once). Un échec d'envoi → retry (backoff exponentiel, DLQ après 10 échecs).

### 2. Canaux — Priorisation B2B Memberships

| Canal | Profil B2B (memberships) | Implémentation |
|---|---|---|
| **Email** | **Principal — v1** | SMTP abstrait (Resend / Postmark / SMTP direct) |
| **Webhook outbound** | Optionnel — v2 | Tenant configure un endpoint webhook (RFC-0023) |
| **In-app** | Optionnel — v3 | WebSocket / SSE, notifications non lues |
| **SMS** | Option tenant — v2 | OTP only + alertes critiques (Twilio / Orange SMS) |
| **WhatsApp** | Option tenant — v3 | OTP + rich alerts (Meta Cloud API / Twilio WhatsApp) |
| **Push mobile** | Option tenant — v3 | Firebase FCM (si app mobile SaaS) |

### 3. Interface `Dispatcher` (port)

```go
// platform/notification/dispatcher.go
package notification

import "context"

// Dispatcher est le port d'émission de notifications.
// Les use cases dépendent de cette interface, jamais de l'implémentation.
type Dispatcher interface {
    Dispatch(ctx context.Context, event Event) error
}

type Event struct {
    Type        EventType       // MEMBER_INVITED, USER_REGISTERED, PASSWORD_RESET...
    RecipientID string          // user_id du destinataire
    TenantID    string
    Locale      string          // "en", "fr" — pour templates localisés
    Payload     map[string]any  // données pour le template
}
```

### 4. Events notifiables (v1)

| EventType | Déclencheur | Canal v1 |
|---|---|---|
| `MEMBER_INVITED` | `InviteMemberUseCase` | Email (lien invitation) |
| `INVITE_ACCEPTED` | `AcceptInviteUseCase` | Email (confirmation Owner) |
| `EMAIL_VERIFICATION_REQUESTED` | `SignupOwnerUseCase` | Email (lien vérification) |
| `PASSWORD_RESET_REQUESTED` | `ForgotPasswordUseCase` | Email (lien reset) |
| `USER_LOGGED_IN_NEW_DEVICE` | `LoginUseCase` | Email (alerte sécurité) |
| `MEMBER_REVOKED` | `RevokeMemberUseCase` | Email (notification révocation) |
| `SUBSCRIPTION_EXPIRING_SOON` | `BillingScheduler` | Email (renouvellement J-7, J-1) |
| `SUBSCRIPTION_PAST_DUE` | `BillingWebhookConsumer` | Email + SMS (si activé) |

### 5. Templates Go `text/template`

```
platform/notification/templates/
├── en/
│   ├── member_invited.html
│   ├── member_invited.txt
│   ├── email_verification.html
│   ├── password_reset.html
│   └── ...
└── fr/
    ├── member_invited.html
    └── ...
```

Sélection par `event.Locale` (défaut `en`). Fallback vers `en` si la locale n'existe pas.

### 6. Struct `Config` du Module

```go
package notification

type Config struct {
    DB          *pgxpool.Pool
    Log         *slog.Logger

    // Canal email (obligatoire en v1)
    SMTPHost    string
    SMTPPort    int
    SMTPFrom    string
    SMTPUser    string
    SMTPPass    string

    // Optionnel — si nil, canal désactivé
    SMSProvider     SMSProvider     // interface: Send(ctx, to, body) error
    WebhookSender   WebhookSender   // interface: Send(ctx, url, secret, payload) error

    // Subscriber de l'outbox pour consommer les events
    Subscriber  events.Subscriber
}
```

### 7. Table `notification_logs`

```sql
CREATE TABLE notification_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID REFERENCES tenants(id),
    recipient_id UUID REFERENCES users(id),
    event_type   VARCHAR(100) NOT NULL,
    channel      VARCHAR(20)  NOT NULL CHECK (channel IN ('email','sms','whatsapp','in_app','push','webhook')),
    status       VARCHAR(20)  NOT NULL DEFAULT 'PENDING'
                    CHECK (status IN ('PENDING','SENT','FAILED','BOUNCED')),
    external_id  VARCHAR(255),  -- ID retourné par le provider (Resend message_id, etc.)
    payload      JSONB,
    sent_at      TIMESTAMPTZ,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notif_logs_tenant_created ON notification_logs(tenant_id, created_at DESC);
CREATE INDEX idx_notif_logs_status_pending ON notification_logs(status)
    WHERE status = 'PENDING';
```

### 8. Preferences de notification (optionnel par user)

```sql
CREATE TABLE user_notification_preferences (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    email_enabled   BOOLEAN NOT NULL DEFAULT TRUE,
    sms_enabled     BOOLEAN NOT NULL DEFAULT FALSE,  -- opt-in explicite
    phone           VARCHAR(30),                      -- pour SMS OTP
    whatsapp_number VARCHAR(30),                      -- peut différer du phone
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 9. SMSProvider — interface abstraite

```go
// platform/notification/sms_provider.go
package notification

type SMSProvider interface {
    Send(ctx context.Context, to, body string) error
}

// Implémentations dans infrastructure/ :
// - TwilioSMSProvider
// - OrangeSMSProvider (marché CEDEAO)
// - NullSMSProvider (tests)
```

---

## **Drawbacks**

1. **Dépendance outbox** : nécessite `kernel/events` (RFC-0015 v1.1) opérationnel avant la notification. Les deux doivent être déployés ensemble.
2. **Délai de notification** : l'email n'est pas envoyé instantanément mais après que le dispatcher outbox ait traité l'event (délai typique < 500ms avec polling SKIP LOCKED).
3. **Templates dans le binaire** : modifier un template = redéploiement. Une v2 pourrait stocker les templates en DB.

---

## **Alternatives**

### Envoi synchrone best-effort (RFC-0011 v1 — status quo)

- ✅ Simple, pas d'outbox
- ❌ Pas de garantie de livraison : si l'email échoue, l'event est perdu
- ❌ Un crash entre l'opération métier et l'envoi = notification silencieusement perdue
- **Verdict : rejeté** — at-least-once via outbox est requis pour B2B

### Service externe géré (Customer.io, Intercom, SendGrid)

- ✅ Analytics, templates visuels, délivrabilité gérée
- ❌ Dépendance externe forte, coût, GDPR (données de contact exposées)
- **Verdict : compatible** — le port `Dispatcher` peut être implémenté avec ces services

---

## **Plan d'adoption**

1. **P0** : RFC rédigée (ce document)
2. **P1-1** : implémenter `platform/notification/` avec email-first (SMTP / Resend) + consume outbox
3. **P1bis** : extension `platform/customer_notifications/` pour end-customers multi-canal (RFC-0036)

---

## **References**

- [RFC-0011 — Platform Notification Multi-Canal](0011-Platform-Notification-Multi-Canal.md) *(supersédé par ce RFC)*
- [RFC-0015 — Event Bus Outbox](0015-Event-Bus-Interne.md)
- [RFC-0036 — Platform customer_notifications](0036-Platform-Customer-Notifications.md) *(à créer — extension B2B2C)*
- [RFC-0004 — Platform Membership](0004-Platform-Membership-Lifecycle.md)
