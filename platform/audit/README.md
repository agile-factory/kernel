# `platform/audit/` — Audit trail

> ✅ **Implémenté** — RFC-0005 + ADR-0009, exposé en v1 (offset) et v2 (cursor).

Trace immuable des actions significatives (membership invited/accepted/revoked, user registered, tenant created, …). Deux surfaces :

1. **Producteur** : interface `EventLogger` que les autres modules appellent dans leurs use cases (fire-and-forget — ADR-0009).
2. **Consommateur** : endpoints HTTP de listing paginés, gardés par RBAC.

---

## Wiring

```go
auditRepo := auditinfra.NewPostgresAuditRepository(db)
auditMod  := audit.New(audit.Config{
    Repo:           auditRepo,
    MembershipRepo: memberRepo, // requis pour les perm-checks HTTP
    Logger:         log,
})

// Émis par les autres modules :
tenantMod := tenant.New(tenant.Config{AuditLogger: auditMod.EventLogger(), ...})
authMod   := auth.New(auth.Config{AuditLogger: auditMod.EventLogger(), ...})

// Surface HTTP :
auditMod.RegisterRoutes(mux)
```

Le repo est construit **hors** du module pour partager la même instance avec membership/tenant (casse le cycle audit↔membership).

---

## Config

| Champ | Type | Requis | Rôle |
|---|---|---|---|
| `Repo` | `domain.AuditRepository` | ✅ | Persistance (PostgreSQL via `auditinfra.NewPostgresAuditRepository`). |
| `MembershipRepo` | `memberdomain.MembershipRepository` | ✅ | Vérification OWNER/ADMIN sur les endpoints `GET /audit-logs`. |
| `Logger` | `*slog.Logger` | ✅ | Stack-trace des écritures best-effort échouées. |

---

## Endpoints

| Méthode | Route | Garde | Description |
|---|---|---|---|
| `GET` | `/api/v1/tenants/{tenantId}/audit-logs` | OWNER/ADMIN du tenant | Pagination offset (RFC-0006 v1) — retourne `data + total`. |
| `GET` | `/api/v2/tenants/{tenantId}/audit-logs` | OWNER/ADMIN du tenant | Pagination cursor (RFC-0006 v2) — retourne `data + next_cursor + has_more`. |
| `GET` | `/api/v1/audit-logs` | Platform super-admin | Vue cross-tenant. |
| `GET` | `/api/v2/audit-logs` | Platform super-admin | Vue cross-tenant cursor. |

Détails de pagination + DTOs : voir [`docs/api/openapi.yaml`](../../docs/api/openapi.yaml) et [ADR-0010](../../docs/architecture/adrs/0010-Pagination-Uniforme-Data-Total.md).

---

## Migrations

| Version | Fichier | Effet |
|---|---|---|
| 0001 | `create_audit_logs` | Table `audit_logs` + enum `audit_action` + index `(tenant_id, actor_id, action, occurred_at DESC)`. |
| 0002 | `add_audit_logs_cursor_indexes` | Index composite `(tenant_id, occurred_at DESC, id DESC)` pour la pagination v2 keyset. |

Embed FS exporté sous `audit.Migrations`. Source d'ordre dans le wiring : **après `events`, `tenant`, `auth`, `membership`** (audit a un FK soft sur `tenants` / `users` mais les références sont `ON DELETE SET NULL`).

---

## Sémantique d'écriture

`EventLogger.Execute` est **fire-and-forget** : la goroutine de save ne bloque jamais le request path. Cause :

- L'audit est annexe au business — perdre une trace ne doit pas faire échouer une opération métier.
- Le ctx est `context.WithoutCancel(parent)` + timeout 5 s → l'écriture survit à un client qui ferme la connexion.

Conséquence : **les callers ne doivent pas brancher sur l'erreur** (signature `Execute(ctx, params)` sans retour). Le RFC-0015 v1.1 fournira plus tard une version at-least-once via l'outbox (P1-3, ADR-0009 marquée DEPRECATED).

Voir [`docs/architecture/technical-deepdive.md`](../../docs/architecture/technical-deepdive.md#audit-best-effort) pour le détail.

---

## Références

- [RFC-0005](../../docs/architecture/rfcs/0005-Platform-Audit-Trail.md) — spec fonctionnelle.
- [ADR-0009](../../docs/architecture/adrs/0009-Audit-Best-Effort-Platform.md) — décision fire-and-forget.
- [ADR-0010](../../docs/architecture/adrs/0010-Pagination-Uniforme-Data-Total.md) — pagination uniforme.
- [RFC-0006](../../docs/architecture/rfcs/0006-Strategie-Pagination-APIs.md) — v1 offset / v2 cursor.
