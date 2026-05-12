# _RFC #0011:_ Platform `notification` — Notifications Multi-Canal

**date:** `2026 May 01`

> ❌ **Supersédé par :**
> - **[RFC-0019 — Platform Notification B2B Email-First](0019-Platform-Notification-B2B-Email-First.md)** (2026-05-12) : redefinit les canaux pour les **memberships** (email-first, outbox at-least-once, SMS/WhatsApp optionnels pour OTP uniquement).
> - **RFC-0036 — Platform customer_notifications** (à créer, P1bis) : canal multi-canal d'office pour les **end-customers B2B2C** (Email + SMS + WhatsApp + Push).
>
> Ce RFC reste valide pour l'historique et les patterns d'architecture (Dispatcher interface, templates Go, notification_logs). Les sections de priorisation de canaux et la stratégie de delivery sont remplacées.

---

## **Status:** `DRAFT` — **SUPERSEDED BY RFC-0019 + RFC-0036**

---

## **Summary**

Ce RFC définit la plateforme `notification`, brique SaaS transversale qui envoie des **notifications aux utilisateurs via plusieurs canaux** (email, SMS, in-app) dans le socle Agile Factory.

La plateforme est découplée des plateformes métier via un **modèle d'événements** : les use cases émettent des événements (`UserInvited`, `MembershipAccepted`), la plateforme `notification` les consomme et choisit le canal approprié.

---

## **Motivation**

Plusieurs fonctionnalités du socle nécessitent des notifications :
- Invitation par email (RFC-0004 : `InviteMemberUseCase` — actuellement sans email)
- Vérification d'email à l'inscription (RFC-0007 : `RegisterUseCase`)
- Réinitialisation de mot de passe
- Alertes de sécurité (connexion depuis un nouveau device)
- Alertes métier par module (facture en retard, contrat expirant)

Sans plateforme centralisée, chaque use case intègre directement l'envoi d'email/SMS, créant des couplages forts et des configurations dupliquées.

---

## **Proposal**

### 1. Architecture orientée événements

```
Use Case → NotificationEvent → NotificationDispatcher → Canal (email/SMS/in-app)
```

Le dispatcher est injecté dans les use cases comme port :

```go
// platform/notification/domain/dispatcher.go
type Dispatcher interface {
    Dispatch(ctx context.Context, event Event) error
}

type Event struct {
    Type       EventType       // MEMBER_INVITED, USER_REGISTERED...
    RecipientID string
    TenantID   string
    Payload    map[string]any  // données pour le template
}
```

### 2. Canaux supportés

| Canal | Cas d'usage | Dépendance externe |
|---|---|---|
| **Email** | Invitation, vérification, reset password | SMTP / Resend / Mailgun |
| **SMS** | OTP, alertes critiques | Twilio / Orange SMS API |
| **WhatsApp** | Notifications riches, OTP, rappels | WhatsApp Business API (Meta) / Twilio WhatsApp |
| **In-App** | Notifications temps réel dans l'UI | WebSocket / SSE |
| **Push** | Mobile (iOS/Android) | Firebase FCM |

La v1 implémente **email uniquement** via SMTP standard (configurable via `SMTP_HOST`, `SMTP_PORT`, `SMTP_FROM`).

### 3. Templates de notifications

Les templates sont des fichiers Go `text/template` stockés dans `platform/notification/templates/` :

```
templates/
├── member_invited.html
├── member_invited.txt
├── user_registered.html
└── password_reset.html
```

Chaque template reçoit le `Payload` de l'événement.

### 4. Modèle de données

```sql
CREATE TABLE notification_logs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID REFERENCES tenants(id),
    recipient_id UUID REFERENCES users(id),
    event_type   VARCHAR(100) NOT NULL,
    channel      VARCHAR(20)  NOT NULL, -- 'email', 'sms', 'whatsapp', 'in_app', 'push'
    status       VARCHAR(20)  NOT NULL DEFAULT 'SENT', -- SENT | FAILED | PENDING
    payload      JSONB,
    sent_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    error        TEXT         -- en cas d'échec
);
```

### 5. Delivery garantie : best-effort v1, outbox v2

En v1 : envoi synchrone best-effort (comme `platform/audit`). Un échec d'envoi est loggué mais ne bloque pas l'opération métier.

En v2 : **outbox pattern** — l'événement est d'abord persisté dans une table `notification_outbox`, puis traité par un worker asynchrone avec retry.

---

## **Drawbacks**

### Dépendance à un service d'envoi externe
En v1 avec SMTP, la livraison dépend d'un serveur mail externe (spam, rebonds, délais). Un service transactionnel (Resend, Postmark) est recommandé pour la production.

### Templates dans le code
Les templates HTML sont dans le binaire Go. Modifier un template nécessite un redéploiement. Une v2 pourrait stocker les templates en DB (éditables via UI).

### Pas de déduplication en v1
Un bug applicatif pourrait envoyer deux emails pour le même événement. L'outbox pattern (v2) résout ce problème avec l'idempotency.

---

## **Alternatives**

### Couplage direct dans les use cases
```go
uc.emailService.Send(ctx, "invitation", user.Email, payload)
```
- ✅ Simple
- ❌ Couplage fort entre use case et canal
- ❌ Impossible de changer de canal sans modifier le use case
- **Verdict : rejeté**

### Service externe (SendGrid, Customer.io)
- ✅ Gestion avancée des templates, analytics, deliverability
- ❌ Dépendance externe forte, coût supplémentaire
- **Verdict : compatible** — le port `Dispatcher` peut être implémenté avec SendGrid sans changer les use cases

### Event bus dédié (RFC-0015)
- ✅ Découplement total, scaling indépendant
- ❌ Infrastructure supplémentaire (NATS, Kafka)
- **Verdict : recommandé à terme** — la plateforme `notification` sera le premier consommateur de l'event bus

---

## **Open Questions**

1. Faut-il un endpoint `GET /notifications` pour l'in-app (notifications non lues) dès la v1 ?
2. Orange SMS API ou Twilio pour le marché sénégalais ?
3. Gestion des préférences de notification (un utilisateur peut désactiver les SMS) ?
4. **WhatsApp** : Meta WhatsApp Business API (via Cloud API directe) ou Twilio WhatsApp sandbox ? La Cloud API directe évite une intermédiaire mais nécessite un numéro dédié vérifié par Meta.

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0004 : Platform Membership (invite sans email actuellement)
- RFC-0007 : Platform Auth (vérification email, reset password)
- RFC-0015 : Event Bus Interne (consommateur principal des notifications)
