# `platform/notification/` — Notification B2B email-first

> ✅ **Implémenté (v0.1.0 partial)** — RFC-0019. Surface : handlers outbox + EmailSender + templates html/text/subject par locale. Aucune route HTTP pour l'instant.

Module de notification du profil B2B. Souscrit aux événements `auth.signup_owner`, `member.invited`, `auth.invite_accepted` (stub) sur le bus outbox, rend les templates par locale, et délègue l'envoi à un `EmailSender` (interface — implémentations Mailpit en dev, Postmark/SES en prod hors kernel).

---

## Wiring

```go
import (
    "github.com/AgileFactory/kernel/platform/notification"
    notifinfra "github.com/AgileFactory/kernel/platform/notification/infrastructure"
)

notifLogRepo := notifinfra.NewPostgresNotificationLogRepository(db)

notifMod, err := notification.New(notification.Config{
    LogRepo:       notifLogRepo,
    EmailSender:   mySender, // implements notification.EmailSender
    Logger:        log,
    From:          "noreply@acme.com",
    BaseURL:       "https://app.acme.com",
    DefaultLocale: "fr",                // optional; default "en"
})
if err != nil { log.Error("notification init", "error", err); os.Exit(1) }

notifMod.Subscribe(bus)         // wires handlers BEFORE bus.Run
go bus.Run(ctx)
```

⚠️ **Order matters** : `Subscribe` doit être appelé **avant** `bus.Run` sinon le premier batch d'événements pendants n'a pas de handler enregistré et part en quarantaine.

---

## Config

| Champ | Type | Requis | Rôle |
|---|---|---|---|
| `LogRepo` | `notifdomain.NotificationLogRepository` | ✅ | Persistance des tentatives (audit + replay). |
| `EmailSender` | `EmailSender` | ✅ | Driver d'envoi. Interface `Send(ctx, EmailMessage) (externalID, error)`. |
| `Logger` | `*slog.Logger` | ✅ | — |
| `From` | `string` | ✅ | Header `From:` des emails sortants. |
| `BaseURL` | `string` | ✅ | URL publique de l'app — sert à composer `VerifyURL` et `AcceptURL`. |
| `DefaultLocale` | `string` | ⏳ | `"en"` par défaut. |

`New()` parse les templates en eager → toute erreur de template surfaces au boot, pas sur le premier event.

---

## Événements consommés

| Outbox event | Handler | Template | Action |
|---|---|---|---|
| `auth.signup_owner` | `handleSignupOwner` | `email_verification` | Envoie le mail de vérification d'email. |
| `member.invited` | `handleMemberInvited` | `member_invited` | Envoie l'invitation avec lien `AcceptURL`. |
| `auth.invite_accepted` | `handleInviteAcceptedStub` | — | No-op (le template owner-notification n'est pas livré en v0.1.0 ; renvoie nil pour marquer l'événement publié et éviter la quarantaine). |

---

## Templates

Embed FS sous `templates/{locale}/{name}.{html,txt,subject}.tmpl`. Locales livrées :

- `en` : `email_verification`, `member_invited`.
- `fr` : `email_verification`, `member_invited`.

Trois fichiers par template (subject / text / html). Le renderer applique fallback automatique sur la locale par défaut si la locale demandée n'existe pas.

---

## Migrations

| Version | Fichier | Effet |
|---|---|---|
| 0001 | `create_notification_logs` | Table `notification_logs` (status : `PENDING`, `SENT`, `FAILED` ; metadata JSONB ; external_id). |

Embed FS : `notification.Migrations`. Pas d'ordre fort — autonome.

---

## Dispatch direct (hors outbox)

Pour les flows hors transaction (cron, test) :

```go
err := notifMod.Dispatch(ctx, notification.Event{
    Type:   notification.EventEmailVerificationRequested,
    Email:  "alice@acme.com",
    Locale: "fr",
    Payload: map[string]any{
        "TenantName": "Acme",
        "VerifyURL":  "https://app.acme.com/verify-email?token=...",
    },
})
```

---

## Surface HTTP

`RegisterRoutes` est un **no-op** en v0.1.0 — le module est consommateur d'événements, pas producteur d'API. Les vues admin sur `notification_logs` (replay manuel, dashboard) sont en backlog.

---

## Références

- [RFC-0019](../../docs/architecture/rfcs/0019-Platform-Notification-B2B-Email-First.md) — spec B2B email-first.
- [RFC-0015](../../docs/architecture/rfcs/0015-Event-Bus-Interne.md) — outbox.
