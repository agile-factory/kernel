# `platform/membership/` — Cycle de vie des memberships

> ✅ **Implémenté** — RFC-0004 + RFC-0018 §6 (invitation token-based).

Modélise la relation many-to-many user ↔ tenant avec un rôle (`ROLE_ADMIN`, `ROLE_AGENT`, `ROLE_VIEWER`, `ROLE_OWNER`) et un statut (`PENDING`, `ACTIVE`, `REVOKED`). Gère le cycle invite → accept → revoke.

> **Évolution RFC-0018 §6** : l'invitation crée un **`invite_token`** (durée 7j), pas un membership `PENDING`. L'acceptation (`POST /auth/accept-invite`) consomme le token et insère un membership `ACTIVE`.

---

## Wiring

```go
memberRepo      := memberinfra.NewPostgresMembershipRepository(db)
inviteTokenRepo := memberinfra.NewPostgresInviteTokenRepository(db)

memberMod := membership.New(membership.Config{
    Pool:            db,
    Repo:            memberRepo,
    InviteTokenRepo: inviteTokenRepo,
    TenantRepo:      tenantRepo,
    UserLookup:      userRepo,                  // typiquement auth.UserRepo
    AuditLogger:     auditMod.EventLogger(),
    Publisher:       bus.Publisher(),           // nil = event 'member.invited' skipped
    Logger:          log,
})
memberMod.RegisterRoutes(mux)
```

Le `UserLookup` est l'interface minimale `{ FindByEmail(ctx, email) }` consommée pour distinguer invite vers user existant vs. invite vers email inconnu. Typiquement on passe le `*authinfra.PostgresUserRepository` qui l'implémente déjà.

---

## Config

| Champ | Type | Requis | Rôle |
|---|---|---|---|
| `Pool` | `*pgxpool.Pool` | ✅ | Transaction (invite = insert token + audit + event dans une seule tx). |
| `Repo` | `domain.MembershipRepository` | ✅ | CRUD memberships. |
| `InviteTokenRepo` | `domain.InviteTokenRepository` | ✅ | CRUD invite_tokens. |
| `TenantRepo` | `tenantdomain.TenantRepository` | ✅ | Vérifier que le tenant existe + est actif avant d'inviter. |
| `UserLookup` | `usecase.UserLookup` | ✅ | Détection email connu vs. nouveau. |
| `AuditLogger` | `audit.EventLogger` | ✅ | Trace `MEMBER_INVITED`, `MEMBER_REVOKED`. |
| `Publisher` | `events.Publisher` | ⏳ | Si nil, `member.invited` skipped. |
| `Logger` | `*slog.Logger` | ✅ | — |

---

## Endpoints

| Méthode | Route | Garde | Description |
|---|---|---|---|
| `POST` | `/api/v1/memberships/invite` | OWNER/ADMIN du tenant | Crée un `invite_token` (durée 7j). Émet `member.invited`. Réponse contient le **raw token** (à transmettre au mail ou copier à la main). |
| `POST` | `/api/v1/memberships/{id}/revoke` | OWNER/ADMIN du tenant | Passe membership ACTIVE → REVOKED. Idempotent. |
| `GET` | `/api/v1/memberships/tenant/{tenantId}` | Membership du tenant | Liste paginée (offset). |
| `GET` | `/api/v1/memberships/user/{userId}` | Self ou platform admin | Liste paginée. |

Détails : [`docs/api/openapi.yaml`](../../docs/api/openapi.yaml).

---

## Migrations

| Version | Fichier | Effet |
|---|---|---|
| 0001 | `create_memberships` | Table + enums `member_role` + `member_status` + `UNIQUE(user_id, tenant_id)`. |
| 0002 | `add_owner_role` | Ajoute `ROLE_OWNER` à l'enum `member_role`. |
| 0003 | `create_invite_tokens` | Table tokens (RFC-0018 §6) — TTL 7j, partial unique index sur `(token) WHERE consumed_at IS NULL AND revoked_at IS NULL`. |
| 0004 | `partial_unique_active_memberships` | `UNIQUE(user_id, tenant_id) WHERE status != 'REVOKED'` — autorise réinvite après révocation. |

Embed FS : `membership.Migrations`. Ordre : **après** `tenant` + `auth` (FKs).

---

## Événements outbox

Émis lors d'un `Invite` réussi (dans la même tx que l'insertion de `invite_tokens` row + audit) :

```json
{
  "event_type": "member.invited",
  "payload": {
    "email": "alice@acme.com",
    "token": "<raw>",
    "role": "ROLE_AGENT",
    "tenant_name": "Acme",
    "inviter_name": "bob@acme.com",
    "expires_at": "2026-05-22T10:00:00Z"
  },
  "meta": {
    "tenant_id": "...",
    "occurred_at": "2026-05-15T10:00:00Z"
  }
}
```

Consommé par `platform/notification` (template `member_invited`, voir [`platform/notification/README.md`](../notification/README.md)).

---

## Références

- [RFC-0004](../../docs/architecture/rfcs/0004-Platform-Membership-Lifecycle.md) — cycle de vie.
- [RFC-0018](../../docs/architecture/rfcs/0018-Amendements-B2B-Onboarding.md) §6 — amendement token-based.
