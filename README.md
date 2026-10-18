# AgileFactory / kernel

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Architecture](https://img.shields.io/badge/Architecture-Clean%203%20niveaux-brightgreen)
![Multi-Tenant](https://img.shields.io/badge/Multi--Tenant-Shared%20Schema-blue)
![Release](https://img.shields.io/badge/Release-v0.1.0-blueviolet)
![License](https://img.shields.io/badge/License-Private-lightgrey)

> **`kernel`** est la bibliothèque Go socle de l'organisation **Agile Factory**.
> Elle fournit, sous forme de packages réutilisables, les briques transverses (auth, multi-tenancy, audit, événements transactionnels…) pour démarrer rapidement de nouveaux produits **SaaS B2B** et **B2B2C**.

**Repo :** [`AgileFactory/kernel`](https://github.com/AgileFactory/kernel) (privé)
**Path d'import :** `github.com/AgileFactory/kernel/...`
**Release courante :** `v0.1.0` (P0 — fondations B2B)

---

## Ce que c'est

`kernel` n'est **pas une application à forker**. C'est un **monorepo de packages Go importables** qu'un repo SaaS consommateur intègre via `go get` :

```bash
export GOPRIVATE=github.com/AgileFactory/*
git config --global \
    url."https://<USER>:<PAT>@github.com/AgileFactory/".insteadOf \
    "https://github.com/AgileFactory/"

go get github.com/AgileFactory/kernel@v0.1.0
```

Chaque produit SaaS garde son propre `main.go`, ses propres modules métier, et compose les packages `kernel` dont il a besoin. Le code de référence vit dans [`examples/saas-b2b/main.go`](examples/saas-b2b/main.go).

### Profils supportés

| Profil | Description | Surface HTTP |
|---|---|---|
| **SaaS B2B** | App pour utilisateurs internes d'une organisation cliente | `/api/v1/*` (JWT + RBAC) |
| **B2B2C** *(P1bis)* | Le tenant sert et facture ses propres clients finaux | + `/customer/v1/*` |
| **BaaS** *(P2+, différé)* | API publique pour développeurs externes | + `/public/v1/*` (API Keys) |

Les profils additionnels s'activent via flags côté repo consommateur : `B2B2C_ENABLED=true`, `BAAS_ENABLED=true` (futur). Un SaaS B2B pur garde les deux à `false` sans surcoût.

---

## Démarrage rapide (repo consommateur)

Le wiring complet (≈130 lignes) est dans [`examples/saas-b2b/main.go`](examples/saas-b2b/main.go). Forme abrégée :

```go
package main

import (
    "context"
    "net/http"

    deliveryhttp "github.com/AgileFactory/kernel/delivery/http"
    "github.com/AgileFactory/kernel/kernel/config"
    "github.com/AgileFactory/kernel/kernel/events"
    "github.com/AgileFactory/kernel/kernel/logger"
    "github.com/AgileFactory/kernel/kernel/migrator"
    "github.com/AgileFactory/kernel/kernel/server"
    "github.com/AgileFactory/kernel/platform/audit"
    auditinfra "github.com/AgileFactory/kernel/platform/audit/infrastructure"
    "github.com/AgileFactory/kernel/platform/auth"
    authinfra "github.com/AgileFactory/kernel/platform/auth/infrastructure"
    "github.com/AgileFactory/kernel/platform/membership"
    memberinfra "github.com/AgileFactory/kernel/platform/membership/infrastructure"
    "github.com/AgileFactory/kernel/platform/tenant"
    tenantinfra "github.com/AgileFactory/kernel/platform/tenant/infrastructure"
    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    cfg := config.Load()
    log, _, _ := logger.New(logger.Config{Level: cfg.LogLevel, Environment: cfg.Environment, Dir: cfg.LogDir})
    db, _ := pgxpool.New(context.Background(), cfg.DatabaseURL)
    defer db.Close()

    // 1. Migrations — ordonnées par dépendance, events d'abord pour l'outbox.
    _ = migrator.Run(cfg.DatabaseURL, []migrator.Source{
        {Name: "events",     FS: events.Migrations,     Path: "migrations"},
        {Name: "tenant",     FS: tenant.Migrations,     Path: "migrations"},
        {Name: "auth",       FS: auth.Migrations,       Path: "migrations"},
        {Name: "membership", FS: membership.Migrations, Path: "migrations"},
        {Name: "audit",      FS: audit.Migrations,      Path: "migrations"},
    })

    // 2. Events bus (outbox publisher + dispatcher).
    bus, _ := events.New(events.Config{Pool: db, Logger: log})

    // 3. Repos en dehors des modules : un seul instance partagée. Casse le
    //    cycle audit↔membership (audit a besoin du repo membership pour
    //    perm-check, membership a besoin de audit.EventLogger).
    auditRepo          := auditinfra.NewPostgresAuditRepository(db)
    tenantRepo         := tenantinfra.NewPostgresTenantRepository(db)
    tenantSettingsRepo := tenantinfra.NewPostgresTenantSettingsRepository(db)
    memberRepo         := memberinfra.NewPostgresMembershipRepository(db)
    userRepo           := authinfra.NewPostgresUserRepository(db)
    refreshTokenRepo   := authinfra.NewPostgresRefreshTokenRepository(db)
    emailVerifRepo     := authinfra.NewPostgresEmailVerificationTokenRepository(db)
    inviteTokenRepo    := memberinfra.NewPostgresInviteTokenRepository(db)

    // 4. Modules — composition explicite, sans framework de DI.
    auditMod  := audit.New(audit.Config{Repo: auditRepo, MembershipRepo: memberRepo, Logger: log})
    tenantMod := tenant.New(tenant.Config{
        Pool: db, Repo: tenantRepo, SettingsRepo: tenantSettingsRepo,
        MembershipRepo: memberRepo, AuditLogger: auditMod.EventLogger(), Logger: log,
    })
    memberMod := membership.New(membership.Config{
        Pool: db, Repo: memberRepo, InviteTokenRepo: inviteTokenRepo,
        TenantRepo: tenantRepo, UserLookup: userRepo,
        AuditLogger: auditMod.EventLogger(), Publisher: bus.Publisher(), Logger: log,
    })
    authMod := auth.New(auth.Config{
        Pool: db, UserRepo: userRepo,
        TenantRepo: tenantRepo, TenantSettingsRepo: tenantSettingsRepo,
        MembershipRepo: memberRepo, InviteTokenRepo: inviteTokenRepo,
        RefreshTokenRepo: refreshTokenRepo, EmailVerifRepo: emailVerifRepo,
        MembershipListUC: memberMod.ListUC,
        AuditLogger: auditMod.EventLogger(), Publisher: bus.Publisher(),
        JWTSecret: cfg.JWTSecret, TokenLifespan: cfg.TokenLifespan, Logger: log,
    })

    // 5. HTTP + dispatcher.
    mux := http.NewServeMux()
    auditMod.RegisterRoutes(mux)
    tenantMod.RegisterRoutes(mux)
    memberMod.RegisterRoutes(mux)
    authMod.RegisterRoutes(mux)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go bus.Run(ctx)

    server.New(cfg, deliveryhttp.ApplyGlobalMiddlewares(mux, log)).Start()
}
```

Surface HTTP livrée en v0.1.0 :

```
POST   /api/v1/auth/signup-owner       SignupOwner (atomic tenant+user+OWNER+email-verif)
POST   /api/v1/auth/accept-invite      Consume invite_token, créer user (si nouveau) + ACTIVE membership
POST   /api/v1/auth/verify-email       Consume verification token
POST   /api/v1/auth/login               Email + password → access + refresh tokens
POST   /api/v1/auth/refresh             Rolling refresh-token rotation
POST   /api/v1/auth/logout              Révoque le refresh-token présenté

GET    /api/v1/tenants/{id}             Membership required
GET    /api/v1/tenants/slug/{slug}      Membership required
PATCH  /api/v1/tenants/{id}             OWNER/ADMIN
DELETE /api/v1/tenants/{id}             OWNER (soft-delete)
GET    /api/v1/tenants/{id}/settings    Membership
PATCH  /api/v1/tenants/{id}/settings    OWNER/ADMIN
POST   /admin/v1/tenants                Platform super-admin (sales-assisted onboarding)

POST   /api/v1/memberships/invite       OWNER/ADMIN ; émet invite_token + event member.invited
POST   /api/v1/memberships/{id}/revoke  OWNER/ADMIN
GET    /api/v1/memberships/tenant/{id}  Liste paginée
GET    /api/v1/memberships/user/{id}    Liste paginée

GET    /api/v1/tenants/{id}/audit-logs  v1 offset, admin only
GET    /api/v2/tenants/{id}/audit-logs  v2 cursor (RFC-0006)
GET    /api/v1/audit-logs               Platform super-admin
GET    /api/v2/audit-logs               Platform super-admin (cursor)
```

---

## Structure du repo

Le code est **directement à la racine** (pas sous `internal/`) pour qu'il soit importable depuis n'importe quel repo consommateur.

```
kernel/
├── kernel/                     # Infra transverse importable
│   ├── auth/                   # context helpers + JWT context (RFC-0007)
│   ├── config/                 # Chargement env (godotenv)
│   ├── errors/                 # AppError + codes (ADR-0004)
│   ├── events/                 # Outbox publisher + dispatcher (RFC-0015 v1.1)
│   ├── logger/                 # slog fan-out (ADR-0005)
│   ├── migrator/               # golang-migrate + iofs (par-package via embed.FS)
│   ├── response/               # JSON HTTP helpers
│   ├── server/                 # HTTP server bootstrap + graceful shutdown
│   ├── storage/                # Port abstrait object-storage (interface Storage)
│   ├── txctx/                  # Transaction propagation via context (DBTX + ExecInTx)
│   └── validator/              # Email, UUID, etc.
├── platform/                   # Briques SaaS réutilisables
│   ├── audit/                  # ✅ Audit trail v1 + v2 cursor (RFC-0005 / ADR-0009)
│   ├── auth/                   # ✅ SignupOwner/AcceptInvite/Verify/Login/Refresh/Logout (RFC-0007 + 0018)
│   ├── membership/             # ✅ Invite via invite_tokens + revoke + list (RFC-0004 + 0018)
│   ├── tenant/                 # ✅ Tenant + tenant_settings + admin endpoints (RFC-0008 + 0018)
│   ├── storage/                # ✅ Port + backend S3 via minio-go (R2/MinIO/AWS/B2) — RFC-0017
│   ├── billing/                # ⏳ P1 (RFC-0020)
│   ├── notification/           # ⏳ P1 (RFC-0019)
│   ├── permission/             # ⏳ P1 (RFC-0010)
│   ├── user/                   # ⏳ P1 (RFC-0009)
│   ├── sso/                    # ⏳ P1 (RFC-0021 + RFC-0028 RS256/JWKS)
│   ├── end_customers/          # ⏳ P1bis B2B2C (RFC-0033)
│   ├── customer_auth/          # ⏳ P1bis B2B2C (RFC-0034)
│   ├── customer_billing/       # ⏳ P1bis B2B2C (RFC-0020 Merchant Mode)
│   ├── customer_notifications/ # ⏳ P1bis B2B2C (RFC-0036)
│   ├── white_label/            # ⏳ P1bis B2B2C (RFC-0035)
│   ├── apikey/                 # 🔒 Différé P2+ — README only (RFC-0022)
│   ├── webhooks_out/           # 🔒 Différé P2+ — README only (RFC-0023)
│   ├── usage_metering/         # 🔒 Différé P2+ — README only (RFC-0026)
│   └── devportal/              # 🔒 Différé P2+ — README only (RFC-0025)
├── delivery/http/              # Middlewares HTTP partagés (Auth, Tenant, Logger, Recovery)
├── examples/
│   └── saas-b2b/               # MVP B2B minimal — main.go de référence
├── docs/
│   ├── architecture/
│   │   ├── overview.md             # Vue d'ensemble (stack, layering, wiring)
│   │   ├── file-structure.md       # Détail du plan du repo
│   │   ├── technical-deepdive.md   # Invariants critiques (outbox, txctx, audit, AppError...)
│   │   ├── adrs/                   # Architecture Decision Records (0001-0010)
│   │   │   ├── rfcs/                   # Requests For Comments (0001-0026, 0028, 0033-0036, 0038)
│   │   └── diagrams/               # mermaid: layering + signup/invite/outbox flows + ERD
│   ├── api/
│   │   └── openapi.yaml            # Référence OpenAPI 3 — tous les endpoints v0.1.0
│   ├── development/
│   │   ├── conventions.md
│   │   └── integration-guide.md    # Tutoriel pas-à-pas pour un consommateur
│   └── operations/
│       └── runbook.md              # Procédures ops : deploy, migrations, rollback, outbox stuck...
├── scripts/                    # Lints + sync-wiki + utilitaires CI
├── .github/workflows/          # CI (build/vet/test/staticcheck/lints) + wiki-sync
├── go.mod                      # module github.com/AgileFactory/kernel
├── Dockerfile                  # multi-stage build de l'exemple saas-b2b
├── docker-compose.yml
├── Makefile
└── README.md
```

Conventions enforced par `scripts/lint.sh` en CI : pas d'`init()`, pas d'`os.Getenv` dans `platform/`, pas d'`internal/`, pas d'import vers `cmd/` ou `examples/` depuis `platform/`. Cf. [RFC-0038](docs/architecture/rfcs/0038-Conventions-Package-Reutilisable.md).

---

## Stack technique

| Couche | Choix |
|---|---|
| Langage | Go 1.25 |
| HTTP router | `net/http` stdlib (Go 1.22+ routing) — ADR-0001 |
| Base de données | PostgreSQL 16 via `jackc/pgx/v5` |
| Migrations | `golang-migrate/iofs` + `embed.FS` par-package (table de versions `schema_migrations_<feature>` séparée par package) |
| Auth (v0.1.0) | JWT HS256 stdlib + refresh tokens hashés SHA-256 + rolling rotation |
| Auth (P1) | RS256 + endpoint JWKS pour SSO/Enterprise — RFC-0028 |
| Événements | Outbox transactionnel PostgreSQL **at-least-once** (single-writer via `pg_try_advisory_lock`, backoff exponentiel, max 10 tentatives) — RFC-0015 v1.1 |
| Mot de passe | `bcrypt` (coût configurable) + minimum 12 caractères — RFC-0018 §8 |
| Logging | `log/slog` fan-out — ADR-0005 |
| Payments *(P1)* | Stripe (international) + Wave / Orange Money / Free Money (CEDEAO) via adapters — RFC-0020 |
| Observability *(P1)* | Prometheus + OpenTelemetry — RFC-0029 |

---

## Index RFC / ADR

### RFCs livrés ou implémentés

| RFC | Titre | Statut |
|---|---|---|
| [RFC-0002](docs/architecture/rfcs/0002-Create-Boilerplate-for-Agile-Factory.md) | Architecture & Stack Technique | PROPOSED |
| [RFC-0004](docs/architecture/rfcs/0004-Platform-Membership-Lifecycle.md) | Platform Membership Lifecycle | ACCEPTED |
| [RFC-0005](docs/architecture/rfcs/0005-Platform-Audit-Trail.md) | Platform Audit Trail | ACCEPTED + IMPL v2 cursor |
| [RFC-0006](docs/architecture/rfcs/0006-Strategie-Pagination-APIs.md) | Stratégie Pagination APIs | ACCEPTED + IMPL v2 |
| [RFC-0007](docs/architecture/rfcs/0007-Platform-Auth-Authentification.md) | Platform Auth *(sections onboarding supersédées par RFC-0018)* | ACCEPTED |
| [RFC-0008](docs/architecture/rfcs/0008-Platform-Tenant-Organisations.md) | Platform Tenant *(création publique supersédée par RFC-0018)* | ACCEPTED |
| [RFC-0015](docs/architecture/rfcs/0015-Event-Bus-Interne.md) | Event Bus + Outbox (v1.1) | PROPOSED + IMPL outbox P0-8 |
| [RFC-0017](docs/architecture/rfcs/0017-Platform-Object-Storage.md) | Platform Object Storage — Port + S3 backend | ACCEPTED + IMPL v0.1.0 partial (presign en v2) |
| **[RFC-0018](docs/architecture/rfcs/0018-Amendements-B2B-Onboarding.md)** | **Amendements B2B Onboarding** — signup-owner, accept-invite, refresh tokens | ACCEPTED + IMPL P0-10 |
| **[RFC-0038](docs/architecture/rfcs/0038-Conventions-Package-Reutilisable.md)** | **Conventions de Package Réutilisable** — Module/Config/Migrations | ACCEPTED + IMPL P0-0 |

### RFCs ACCEPTED — implémentation P1

| RFC | Titre | Implémentation |
|---|---|---|
| [RFC-0019](docs/architecture/rfcs/0019-Platform-Notification-B2B-Email-First.md) | Platform notification — B2B Email-First | P1-1 |
| [RFC-0020](docs/architecture/rfcs/0020-Platform-Billing-Hybride-Merchant-Mode.md) | Platform billing — Hybride Stripe + Mobile Money | P1-2 + P1bis-8 (Merchant Mode) |
| [RFC-0021](docs/architecture/rfcs/0021-Platform-SSO-Enterprise.md) | Platform SSO — OIDC + SAML enterprise | P1-7 |
| [RFC-0028](docs/architecture/rfcs/0028-JWT-RS256-JWKS.md) | JWT RS256 + JWKS Endpoint | P1-7 (avec SSO) |

### RFCs supersédées (historique préservé)

| RFC | Supersédé par |
|---|---|
| [RFC-0011](docs/architecture/rfcs/0011-Platform-Notification-Multi-Canal.md) | RFC-0019 + RFC-0036 |
| [RFC-0012](docs/architecture/rfcs/0012-Platform-Billing-Abonnements.md) | RFC-0020 |

### RFCs P1bis B2B2C — PROPOSED (à valider Aziz + Lucas)

| RFC | Titre | Statut |
|---|---|---|
| [RFC-0033](docs/architecture/rfcs/0033-Platform-End-Customers.md) | Platform end_customers — Clients Finaux | ACCEPTED |
| [RFC-0034](docs/architecture/rfcs/0034-Platform-Customer-Auth.md) | Platform customer_auth — Auth Clients Finaux | ACCEPTED |
| [RFC-0035](docs/architecture/rfcs/0035-Platform-White-Label.md) | Platform white_label — Branding Per-Tenant | ACCEPTED |
| [RFC-0036](docs/architecture/rfcs/0036-Platform-Customer-Notifications.md) | Platform customer_notifications — Email + SMS | ACCEPTED |

### RFCs BaaS — DRAFT DEFERRED (P2+)

[RFC-0022](docs/architecture/rfcs/0022-Platform-APIKey-PAT-Scopes.md) · [RFC-0023](docs/architecture/rfcs/0023-Platform-Webhooks-Sortants-Signes.md) · [RFC-0024](docs/architecture/rfcs/0024-Kernel-Rate-Limiting.md) · [RFC-0025](docs/architecture/rfcs/0025-Surface-Publique-BaaS-Versioning.md) · [RFC-0026](docs/architecture/rfcs/0026-Platform-Usage-Metering.md)

Squelettes réservés sous `platform/{apikey,webhooks_out,usage_metering,devportal}/` (README only).

### ADRs

| ADR | Titre | Statut |
|---|---|---|
| [ADR-0001](docs/architecture/adrs/0001-Go-Stdlib-HTTP-Router.md) | Go Stdlib HTTP Router | ACCEPTED |
| [ADR-0002](docs/architecture/adrs/0002-Clean-Architecture-Trois-Niveaux.md) | Clean Architecture 3 Niveaux | ACCEPTED |
| [ADR-0003](docs/architecture/adrs/0003-Multi-Tenancy-Shared-Schema-Memberships.md) | Multi-Tenancy Shared Schema | IMPLEMENTED |
| [ADR-0004](docs/architecture/adrs/0004-Gestion-Erreurs-AppError.md) | Gestion Erreurs AppError | IMPLEMENTED |
| [ADR-0005](docs/architecture/adrs/0005-Logging-Fan-Out-Slog.md) | Logging Fan-Out slog | IMPLEMENTED |
| [ADR-0006](docs/architecture/adrs/0006-Migrations-SQL-Golang-Migrate.md) | Migrations SQL golang-migrate | IMPLEMENTED |
| [ADR-0007](docs/architecture/adrs/0007-JWT-HS256-Stdlib.md) | JWT HS256 Stdlib *(→ RFC-0028 pour RS256+JWKS)* | IMPLEMENTED |
| [ADR-0008](docs/architecture/adrs/0008-Injection-Dependances-Manuelle.md) | DI manuelle (pas de wire/fx) | ACCEPTED |
| [ADR-0009](docs/architecture/adrs/0009-Audit-Best-Effort-Platform.md) | Audit Best-Effort *(DEPRECATED — outbox at-least-once en P1-3)* | IMPLEMENTED |
| [ADR-0010](docs/architecture/adrs/0010-Pagination-Uniforme-Data-Total.md) | Pagination uniforme data + total | ACCEPTED |

---

## Développement local

### Prérequis

- Go 1.25+
- Docker & Docker Compose

### Commandes

```bash
make dev            # docker compose up --build (postgres + backend)
make run            # go run l'exemple saas-b2b localement (migrations auto au boot)
make build          # binaire bin/api de l'exemple saas-b2b
make test           # go test -v ./...
make fmt            # go fmt ./...
make tidy           # go mod tidy
make db-shell       # psql sur le postgres docker
make db-reset       # DROP SCHEMA public CASCADE + recréation, next run re-applique
make help           # liste annotée
```

Pas de target `migrate` séparée : les migrations sont **embarquées** dans chaque package via `//go:embed` et appliquées par `migrator.Run` au démarrage du binaire.

### Variables d'environnement clés

| Variable | Défaut | Description |
|---|---|---|
| `APP_ENV` | `development` | `development` / `staging` / `production` |
| `DATABASE_URL` | — | `postgres://user:pass@host:5432/db` (requis hors `development`/`test`) |
| `JWT_SECRET` | — | Min 32 bytes (HMAC-SHA256). Requis hors `development`/`test` (refus de démarrer) ; en dev, un secret aléatoire par process est généré s'il est absent |
| `PORT` | `8080` | Port HTTP |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `TOKEN_LIFESPAN` | `1h` | Durée de validité des access tokens |

`REFRESH_TOKEN_TTL` (défaut 30 jours) et `BCRYPT_COST` (défaut `bcrypt.DefaultCost`) sont passés via `auth.Config` côté repo consommateur.

---

## CI

GitHub Actions ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) :

| Job | Vérifie |
|---|---|
| `build-test` | `go mod tidy` propre, `go vet`, `go build`, `go test -race -coverprofile` |
| `staticcheck` | `staticcheck v0.7.0` (compatible Go 1.25) |
| `lint-conventions` | [`scripts/lint.sh`](scripts/lint.sh) — RFC-0038 : no `init()`, no `os.Getenv` dans `platform/`, no `internal/`, no import `cmd/`/`examples/` depuis `platform/` |
| `lint-migrations` | [`scripts/lint-migrations.sh`](scripts/lint-migrations.sh) — naming `NNNN_<desc>.{up,down}.sql`, paires complètes, versions uniques par package |

Le workflow [`wiki-sync.yml`](.github/workflows/wiki-sync.yml) mirroite `docs/` → GitHub Wiki **sur chaque push à `main`** qui touche `docs/`, `scripts/sync-wiki.sh` ou le workflow lui-même. `workflow_dispatch` reste disponible pour rebuild manuel (force-resync après édition du script).

---

## Documentation

> Source de vérité dans `docs/`. Mirror lecture sur le [Wiki GitHub](https://github.com/AgileFactory/kernel/wiki) (auto-sync).

| Audience | Document | Quand le lire |
|---|---|---|
| **Nouveau consommateur** | [Integration Guide](docs/development/integration-guide.md) | Première intégration de `kernel` dans un repo SaaS — tutoriel pas-à-pas (30 min). |
| **Frontend / mobile / intégrateur API** | [OpenAPI 3](docs/api/openapi.yaml) | Référence exhaustive des endpoints v0.1.0 — schémas, codes d'erreur, exemples. |
| **Contributeur kernel** | [Technical deep-dive](docs/architecture/technical-deepdive.md) | Comprendre les invariants critiques (outbox, txctx, cycle audit↔membership, AppError…). |
| **Architecte** | [Architecture overview](docs/architecture/overview.md) + [diagrammes](docs/architecture/diagrams/) | Vue d'ensemble, séquences (signup, invite, outbox), ERD. |
| **SRE / oncall** | [Runbook](docs/operations/runbook.md) | Procédures d'exploitation : deploy, migrations zero-downtime, rollback, outbox bloqué, rotation JWT_SECRET, RGPD. |
| **Tous** | [RFCs](docs/architecture/rfcs/) + [ADRs](docs/architecture/adrs/) | Source de vérité des décisions architecturales — voir tableau plus haut. |

Chaque module livré porte son propre README détaillé :
[`platform/audit/`](platform/audit/README.md) · [`platform/auth/`](platform/auth/README.md) · [`platform/membership/`](platform/membership/README.md) · [`platform/tenant/`](platform/tenant/README.md) · [`platform/storage/`](platform/storage/README.md) · [`platform/notification/`](platform/notification/README.md)

---

## Roadmap

| Jalon | Contenu | Statut |
|---|---|---|
| **v0.1.0** | Auth + Tenant + Membership + Audit + Events outbox + CI (fin P0) | ✅ **LIVRÉ** |
| **v0.2.0** | Billing + Notification + Permission + User + SSO + Observability (fin P1) | ⏳ Prochain |
| **v0.3.0** | End-customers + Customer Auth + Merchant Mode + White-label (fin P1bis) | ⏳ |
| **v1.0.0** | Stabilisation des contrats publics après retour des premiers SaaS consommateurs | ⏳ |

Plan complet : [`docs/plan.txt`](docs/plan.txt) → `/home/zone01/.claude/plans/on-se-specialisera-que-tranquil-gray.md`.

---

## Licence

Propriétaire — usage réservé à l'organisation [Agile Factory](https://github.com/AgileFactory).
