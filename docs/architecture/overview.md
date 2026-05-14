# Architecture — `AgileFactory/kernel`

> **`kernel`** est une bibliothèque Go que les repos SaaS de l'organisation **Agile Factory** importent pour bootstrapper rapidement leurs produits. Ce document décrit le socle technique, la disposition des packages, et la façon dont les modules s'assemblent dans un binaire consommateur.

**Pas une application à forker.** Le repo est un monorepo de **packages importables** :
```
github.com/AgileFactory/kernel/kernel/*    // infra transverse
github.com/AgileFactory/kernel/platform/*  // briques SaaS réutilisables
github.com/AgileFactory/kernel/delivery/*  // helpers HTTP partagés
```

Chaque produit SaaS garde son propre `main.go` + ses modules métier locaux, et compose les packages `kernel` dont il a besoin via `go get`.

---

## 1. Stack technique

| Couche | Technologie | Rôle |
|---|---|---|
| **Langage** | Go 1.25 | Type-safe, stdlib HTTP, `embed.FS`, generics |
| **HTTP router** | `net/http` (stdlib) | Zéro lock-in, audit-friendly (ADR-0001) |
| **Données** | PostgreSQL 16 + `jackc/pgx/v5` | Driver natif Postgres, `pgxpool` |
| **Migrations** | `golang-migrate/v4` + `iofs` | SQL embarqué par-package, version-tracking séparé (RFC-0038) |
| **Logging** | `log/slog` (stdlib) | Logging structuré, fan-out (ADR-0005) |
| **Auth** | JWT HS256 (stdlib `crypto/*`) | Tokens internes ; RS256 + JWKS prévus en P1 (RFC-0028) |
| **Mots de passe** | `bcrypt` | Coût configurable par module |
| **Distribution** | Module Go privé (GOPRIVATE) | `go get github.com/AgileFactory/kernel@vX.Y.Z` |

Aucun framework HTTP tiers : la stack est volontairement légère pour faciliter les audits enterprise et limiter la surface de dépendances.

---

## 2. Architecture en 3 couches (Clean Architecture)

Chaque `platform/<feature>` suit le même découpage (ADR-0002) :

```
platform/<feature>/
├── <feature>.go                # API publique : Module + Config + New + RegisterRoutes
├── domain/                     # entities, value objects, repository INTERFACES
├── usecase/                    # use cases (1 fichier par action, tests inclus)
├── infrastructure/             # implémentations concrètes (postgres, services)
├── delivery/http/              # handlers HTTP, DTOs, RegisterRoutes
└── migrations/                 # *.sql embarqués via //go:embed
```

**Règle de dépendance** : `delivery → usecase → domain`. `infrastructure` implémente les interfaces de `domain`. Les couches supérieures ne connaissent jamais d'implémentation concrète — uniquement des interfaces (testabilité + swap d'impl).

### Pattern Module autonome (RFC-0038)

Toute plateforme expose une racine homogène :

```go
type Config struct {
    Repo        Repository         // injecté par le consommateur
    AuditLogger audit.EventLogger  // interface inter-package
    Logger      *slog.Logger
    // ...options
}

type Module struct { /* use cases exposés */ }

func New(cfg Config) *Module               { /* ... */ }
func (m *Module) RegisterRoutes(mux *http.ServeMux) { /* ... */ }

//go:embed migrations/*.sql
var Migrations embed.FS
```

Aucun `init()`, aucune variable globale mutable, aucun singleton : tout passe par la `Config`.

---

## 3. Plan du repo

```
kernel/                                 # repo "AgileFactory/kernel"
├── go.mod                              # module github.com/AgileFactory/kernel
├── kernel/                             # infra transverse importable
│   ├── auth/                           # context, JWT helpers
│   ├── config/                         # chargement env (godotenv)
│   ├── errors/                         # AppError + codes (ADR-0004)
│   ├── logger/                         # slog fan-out (ADR-0005)
│   ├── migrator/                       # composeur embed.FS → golang-migrate
│   ├── response/                       # JSON HTTP helper
│   ├── server/                         # HTTP server bootstrap
│   └── validator/                      # email, uuid, etc.
├── platform/                           # briques SaaS réutilisables
│   ├── audit/                          # audit trail (RFC-0005)
│   ├── auth/                           # authentification (RFC-0007 → RFC-0018)
│   ├── membership/                     # cycle de vie membership (RFC-0004)
│   └── tenant/                         # organisations tenants (RFC-0008)
├── delivery/http/                      # middlewares HTTP partagés
│   └── middleware/                     # Auth, TenantContext, Logger, Recovery
├── examples/                           # apps consommatrices exemples
│   └── saas-b2b/                       # MVP B2B minimal (≈100 lignes de main.go)
└── docs/                               # RFCs, ADRs, architecture
```

Le code n'est **pas** sous `internal/` : tout est importable depuis un repo consommateur externe. Le path d'import débute toujours par `github.com/AgileFactory/kernel/...`.

---

## 4. Multi-tenancy (ADR-0003)

**Modèle : shared schema + memberships.**

- Une seule base PostgreSQL, un seul schéma `public`.
- `tenants(id, slug, name, is_active, …)` : la table des organisations clientes.
- `users(id, email, password_hash, …)` : table globale, sans `tenant_id`.
- `memberships(user_id, tenant_id, role, status, …)` : la relation many-to-many avec rôle (`OWNER/ADMIN/AGENT/VIEWER`) et statut (`PENDING/ACTIVE/REVOKED`).
- Le `tenant_id` est propagé via `context.Context` (middleware `TenantContext`).
- Toute requête métier filtre par `tenant_id` au niveau du repository.

**Defense-in-depth prévu (P2)** : activation optionnelle de **Row-Level Security PostgreSQL** par tenant (critique en B2B2C, cf. plan §10).

---

## 5. Audit trail (RFC-0005, en cours d'amendement)

- Table `audit_logs(id, tenant_id, actor_id, action, target_type, target_id, metadata, created_at)`.
- Chaque action sensible (création tenant, login, invitation, révocation) émet un `audit.LogEventParams` via l'interface `audit.EventLogger`.
- **Mode actuel : best-effort fire-and-forget** (ADR-0009). Le retour de `Execute` est asynchrone et ne fait pas échouer la transaction métier.
- **Évolution P0-8** : remplacer par un **outbox transactionnel** (RFC-0015 v1.1) — l'audit log est inséré dans la même transaction que l'opération métier, puis dispatché en arrière-plan (at-least-once, requis SOC2/ISO27001).

Les consumers (tenant, membership, auth) dépendent de l'**interface** `audit.EventLogger`, pas du type concret — ce qui permet de swapper l'implémentation sans toucher aux use cases.

---

## 6. Migrations embarquées (RFC-0038)

Chaque package porte ses propres migrations sous `platform/<feature>/migrations/`, exposées par un `embed.FS` au niveau du package. Le consommateur compose dans l'ordre des dépendances :

```go
migrator.Run(cfg.DatabaseURL, []migrator.Source{
    {Name: "tenant",     FS: tenant.Migrations,     Path: "migrations"},
    {Name: "auth",       FS: auth.Migrations,       Path: "migrations"},
    {Name: "membership", FS: membership.Migrations, Path: "migrations"},
    {Name: "audit",      FS: audit.Migrations,      Path: "migrations"},
})
```

Chaque source a sa propre table de versions `schema_migrations_<name>` (via le query param `x-migrations-table` du driver postgres), ce qui permet à chaque package de versionner ses migrations indépendamment.

---

## 7. Profils consommateurs

Le boilerplate sert deux profils prioritaires et un futur (plan §2) :

| Profil | Priorité | Surface HTTP | Activation |
|---|---|---|---|
| **SaaS B2B** | P0/P1 | `/api/v1/*` (memberships, JWT) | défaut |
| **B2B2C** | P1bis | + `/customer/v1/*` (end-customers d'un tenant) | `B2B2C_ENABLED=true` |
| **BaaS** *(différé)* | P2+ | + `/public/v1/*` (API Keys, calendar versioning) | `BAAS_ENABLED=true` (futur) |

Un SaaS B2B pur garde les deux flags à `false` sans surcoût : les plateformes additionnelles ne sont pas chargées dans le `main.go`.

---

## 8. Auth, RBAC, sécurité

| Mécanisme | Détail |
|---|---|
| **Tokens** | JWT HS256 actuellement (RFC-0007). RS256 + JWKS prévus en P1 (RFC-0028) pour intégration SSO. |
| **Refresh tokens** | Prévu en P0-10 (table `refresh_tokens` révocable). |
| **Hashing** | `bcrypt` avec coût par défaut (configurable par module). |
| **Multi-tenant** | `tenant_id` extrait du JWT ou du Host header (custom domain B2B2C), propagé via `context.Context`. |
| **Rôles** | `ROLE_OWNER` (non-révocable), `ROLE_ADMIN`, `ROLE_AGENT`, `ROLE_VIEWER` — vérifiés au niveau use case. |
| **RBAC granulaire** | `platform/permission/` (RFC-0010), scopes alignés API Keys, à implémenter en P1. |
| **SSO Enterprise** | `platform/sso/` OIDC + SAML, P1 (RFC-0021). |

### Protection des API (en place ou prévu)

- **CORS** : middleware HTTP standard, origines configurables côté consommateur.
- **Rate limiting** : `kernel/ratelimit/` prévu en P1 (RFC-0024), réutilisable pour `/customer/v1/auth/*` et la future surface BaaS.
- **HTTPS** : gestion à l'extérieur du binaire (reverse proxy / load-balancer / Cloudflare).
- **SQL injection** : impossible avec `pgx` (requêtes paramétrées systématiques).
- **Logs** : aucune donnée personnelle identifiable, propagation de `X-Request-ID` (UUID v7) — RFC-0029.

---

## 9. Wiring type — exemple `examples/saas-b2b/main.go`

```go
// Repos en dehors des modules : un seul instance partagée entre plateformes
// (audit handler a besoin du repo membership pour la perm-check HTTP, tandis
// que membership a besoin de audit.EventLogger).
auditRepo  := auditinfra.NewPostgresAuditRepository(db)
tenantRepo := tenantinfra.NewPostgresTenantRepository(db)
memberRepo := memberinfra.NewPostgresMembershipRepository(db)
userRepo   := authinfra.NewPostgresUserRepository(db)

// Modules — composition explicite, sans framework de DI
auditMod := audit.New(audit.Config{
    Repo: auditRepo, MembershipRepo: memberRepo, Logger: log,
})
tenantMod := tenant.New(tenant.Config{
    Repo: tenantRepo, AuditLogger: auditMod.EventLogger(), Logger: log,
})
memberMod := membership.New(membership.Config{
    Repo: memberRepo, TenantRepo: tenantRepo,
    AuditLogger: auditMod.EventLogger(), Logger: log,
})
authMod := auth.New(auth.Config{
    UserRepo: userRepo, TenantRepo: tenantRepo, MembershipRepo: memberRepo,
    MembershipListUC: memberMod.ListUC,
    AuditLogger: auditMod.EventLogger(),
    JWTSecret: cfg.JWTSecret, TokenLifespan: cfg.TokenLifespan,
    Logger: log,
})

mux := http.NewServeMux()
auditMod.RegisterRoutes(mux)
tenantMod.RegisterRoutes(mux)
memberMod.RegisterRoutes(mux)
authMod.RegisterRoutes(mux)
```

Le consommateur lit ce wiring **en clair** dans son `main.go` : pas de magie, pas de `wire`/`fx`. C'est le coût accepté pour un audit facile et un débuggage direct (ADR-0008).

---

## 10. Évolution prévue (plan de spécialisation)

Le plan complet est dans [`/docs/plan.txt`](../plan.txt) (référence vers `/home/zone01/.claude/plans/on-se-specialisera-que-tranquil-gray.md`). Synthèse :

- **P0 (~4 sem)** — fondations : restructuration packaging (DONE), pattern Module autonome (DONE), migrations embarquées (DONE), RFCs 0018-0028, outbox `kernel/events`, refactor auth/tenant/membership pour signup-owner/accept-invite, tag `v0.1.0`.
- **P1 (~5 sem)** — plateformes core : `billing`, `notification`, `permission`, `user`, `sso` + JWKS, observability (OTel + Prometheus), tag `v0.2.0`.
- **P1bis (~5 sem)** — B2B2C : `end_customers`, `customer_auth`, `customer_billing` (Merchant Mode Stripe Connect + Wave/Orange/Free Marchand), `customer_notifications`, `white_label` (custom domains + branding), mux `/customer/v1/*`, RLS PG obligatoire, tag `v0.3.0`.
- **P2+** — BaaS différé : `apikey`, `webhooks_out`, `usage_metering`, `devportal`, SDK generation, migration UUID v4→v7.

Stratégie hybride payment : Stripe (international) + Wave / Orange Money / Free Money (CEDEAO) via adapters `PaymentProcessor` interchangeables (RFC-0020). Notifications multi-canal par profil : email-first pour memberships B2B, multi-canal large (Email + SMS + WhatsApp + Push) pour end-customers B2B2C (RFC-0019 + RFC-0036).

---

## 11. Références

### ADRs (Architecture Decision Records)
- [ADR-0001](./adrs/0001-Go-Stdlib-HTTP-Router.md) — Go Stdlib HTTP Router
- [ADR-0002](./adrs/0002-Clean-Architecture-Trois-Niveaux.md) — Clean Architecture 3 niveaux
- [ADR-0003](./adrs/0003-Multi-Tenancy-Shared-Schema-Memberships.md) — Multi-tenancy shared schema
- [ADR-0004](./adrs/0004-Gestion-Erreurs-AppError.md) — Gestion d'erreurs `AppError`
- [ADR-0005](./adrs/0005-Logging-Fan-Out-Slog.md) — Logging fan-out slog
- [ADR-0006](./adrs/0006-Migrations-SQL-Golang-Migrate.md) — Migrations SQL
- [ADR-0007](./adrs/0007-JWT-HS256-Stdlib.md) — JWT HS256 stdlib (→ RFC-0028 RS256 successor)
- [ADR-0008](./adrs/0008-Injection-Dependances-Manuelle.md) — DI manuelle
- [ADR-0009](./adrs/0009-Audit-Best-Effort-Platform.md) — Audit best-effort (deprecated par RFC-0015 v1.1)
- [ADR-0010](./adrs/0010-Pagination-Uniforme-Data-Total.md) — Pagination uniforme

### RFCs fondamentaux
- [RFC-0002](./rfcs/0002-Create-Boilerplate-for-Agile-Factory.md) — Création du boilerplate
- [RFC-0004](./rfcs/0004-Platform-Membership-Lifecycle.md) — Lifecycle membership
- [RFC-0005](./rfcs/0005-Platform-Audit-Trail.md) — Audit trail
- [RFC-0007](./rfcs/0007-Platform-Auth-Authentification.md) — Auth (amendé par RFC-0018)
- [RFC-0008](./rfcs/0008-Platform-Tenant-Organisations.md) — Tenant organisations
- [RFC-0015](./rfcs/0015-Event-Bus-Interne.md) — Event bus / outbox
- [RFC-0038](./rfcs/0038-Conventions-Package-Reutilisable.md) — Conventions de package
