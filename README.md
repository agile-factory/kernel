# AgileFactory / kernel

![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)
![Architecture](https://img.shields.io/badge/Architecture-Clean%203%20niveaux-brightgreen)
![Multi-Tenant](https://img.shields.io/badge/Multi--Tenant-Shared%20Schema-blue)
![License](https://img.shields.io/badge/License-Private-lightgrey)

> **`kernel`** est la bibliothèque Go socle de l'organisation **Agile Factory**.  
> Elle fournit, sous forme de packages réutilisables, les briques transverses (auth, multi-tenancy, audit, billing, notifications…) pour démarrer rapidement de nouveaux produits **SaaS B2B** et **B2B2C**.

**Repo :** [`AgileFactory/kernel`](https://github.com/AgileFactory/kernel) (privé)  
**Path d'import :** `github.com/AgileFactory/kernel/...`

---

## Ce que c'est

`kernel` n'est **pas une application à forker**. C'est un **monorepo de packages Go importables** qu'un repo SaaS consommateur intègre via `go get` :

```bash
export GOPRIVATE=github.com/AgileFactory/*
go get github.com/AgileFactory/kernel@v0.1.0
```

Chaque produit SaaS garde son propre `main.go`, ses propres modules métier, et compose les packages `kernel` dont il a besoin.

### Profils supportés

| Profil | Description | Surface HTTP |
|---|---|---|
| **SaaS B2B** | App pour utilisateurs internes d'une organisation cliente | `/api/v1/*` (JWT + RBAC) |
| **B2B2C** | Le tenant sert et facture ses propres clients finaux | `/api/v1/*` + `/customer/v1/*` |
| **BaaS** *(différé P2+)* | API publique pour développeurs externes | `/public/v1/*` (API Keys) |

Les profils s'activent via flags côté repo consommateur : `B2B2C_ENABLED=true`, `BAAS_ENABLED=true` (futur).

---

## Démarrage rapide (repo consommateur)

```go
// fleet-manager/cmd/api/main.go
package main

import (
    "embed"
    "net/http"

    "github.com/AgileFactory/kernel/kernel/config"
    "github.com/AgileFactory/kernel/kernel/events"
    "github.com/AgileFactory/kernel/kernel/migrator"
    "github.com/AgileFactory/kernel/kernel/server"
    "github.com/AgileFactory/kernel/platform/audit"
    "github.com/AgileFactory/kernel/platform/auth"
    "github.com/AgileFactory/kernel/platform/membership"
    "github.com/AgileFactory/kernel/platform/tenant"

    "github.com/AgileFactory/fleet-manager/modules/fleet" // module métier local
)

//go:embed migrations/*.sql
var localMigrations embed.FS

func main() {
    cfg := config.Load()
    log := buildLogger(cfg)
    db  := mustConnectPostgres(cfg.DatabaseURL)

    migrator.Run(db, []embed.FS{
        auth.Migrations, tenant.Migrations,
        membership.Migrations, audit.Migrations,
        localMigrations,
    })

    publisher := events.NewPostgresOutboxPublisher(db, log)
    authMod   := auth.New(auth.Config{DB: db, Log: log, JWTSecret: cfg.JWTSecret, Publisher: publisher})
    tenantMod := tenant.New(tenant.Config{DB: db, Log: log, Publisher: publisher})
    memberMod := membership.New(membership.Config{DB: db, Log: log, Publisher: publisher})
    auditMod  := audit.New(audit.Config{DB: db, Log: log, Subscriber: publisher})
    fleetMod  := fleet.New(fleet.Config{DB: db, Log: log, JWTValidator: authMod.JWTValidator()})

    mux := http.NewServeMux()
    authMod.RegisterRoutes(mux)
    tenantMod.RegisterRoutes(mux)
    memberMod.RegisterRoutes(mux)
    auditMod.RegisterRoutes(mux)
    fleetMod.RegisterRoutes(mux)

    server.Run(cfg, mux, log)
}
```

---

## Structure du repo

```
kernel/
├── backend/
│   ├── cmd/api/                    # Entry point (dev local uniquement)
│   ├── internal/
│   │   ├── bootstrap/              # DI container (dev local)
│   │   ├── delivery/http/          # Middlewares globaux (WithJWT, WithTenant…)
│   │   ├── kernel/                 # Infra technique transverse
│   │   │   ├── auth/               # JWT HS256 + RS256 + JWKS (RFC-0028)
│   │   │   ├── config/             # Chargement env
│   │   │   ├── errors/             # AppError + codes (ADR-0004)
│   │   │   ├── events/             # Outbox publisher + dispatcher (RFC-0015)
│   │   │   ├── logger/             # slog fan-out (ADR-0005)
│   │   │   ├── migrator/           # golang-migrate helper (embed.FS)
│   │   │   ├── ratelimit/          # Port Limiter + implémentations (RFC-0024)
│   │   │   ├── response/           # JSON HTTP helpers
│   │   │   ├── server/             # HTTP server bootstrap
│   │   │   ├── txctx/              # Propagation transaction via context
│   │   │   └── validator/          # Email, UUID, password (zxcvbn)
│   │   └── platform/               # Briques SaaS réutilisables
│   │       ├── audit/              # ✅ IMPLÉMENTÉ (RFC-0005 + ADR-0009 amendée)
│   │       ├── auth/               # ✅ IMPLÉMENTÉ (RFC-0007 + RFC-0018)
│   │       ├── membership/         # ✅ IMPLÉMENTÉ (RFC-0004)
│   │       ├── tenant/             # ✅ IMPLÉMENTÉ (RFC-0008 + RFC-0018)
│   │       ├── billing/            # ⏳ P1 (RFC-0020)
│   │       ├── notification/       # ⏳ P1 (RFC-0019)
│   │       ├── permission/         # ⏳ P1 (RFC-0010)
│   │       ├── user/               # ⏳ P1 (RFC-0009)
│   │       ├── sso/                # ⏳ P1 (RFC-0021)
│   │       ├── end_customers/      # ⏳ P1bis B2B2C (RFC-0033)
│   │       ├── customer_auth/      # ⏳ P1bis B2B2C (RFC-0034)
│   │       ├── customer_billing/   # ⏳ P1bis B2B2C (RFC-0020 Merchant Mode)
│   │       ├── customer_notifications/ # ⏳ P1bis B2B2C (RFC-0036)
│   │       ├── white_label/        # ⏳ P1bis B2B2C (RFC-0035)
│   │       ├── apikey/             # 🔒 Différé P2+ (RFC-0022)
│   │       ├── webhooks_out/       # 🔒 Différé P2+ (RFC-0023)
│   │       ├── usage_metering/     # 🔒 Différé P2+ (RFC-0026)
│   │       └── devportal/          # 🔒 Différé P2+ (RFC-0025)
│   └── migrations/                 # Migrations SQL (en cours de migration vers embed.FS par package)
├── docs/
│   ├── architecture/
│   │   └── overview.md             # Vue d'ensemble — stack, layering, profils
│   ├── rfcs/                       # Décisions produit (voir index ci-dessous)
│   └── adrs/                       # Décisions techniques
├── Makefile
└── docker-compose.yml
```

---

## Stack technique

| Couche | Choix |
|---|---|
| Langage | Go 1.25+ |
| HTTP | `net/http` stdlib (routing Go 1.22+) |
| Base de données | PostgreSQL 16+ via `pgx/v5` |
| Migrations | `golang-migrate` + `embed.FS` par package |
| Auth | JWT HS256 (interne) + RS256 + JWKS (SSO/enterprise) |
| Événements | Outbox transactionnel PostgreSQL (at-least-once) |
| Logging | `log/slog` fan-out (stdout + app.log + error.log + server.log) |
| Payments | Stripe (international) + Wave/Orange Money/Free Money (CEDEAO) |
| Observability | Prometheus + OpenTelemetry (RFC-0016/0029) |

---

## Index RFC

### RFCs actives

| RFC | Titre | Statut |
|---|---|---|
| [RFC-0002](docs/rfcs/0002-Create-Boilerplate-for-Agile-Factory.md) | Architecture & Stack Technique | PROPOSED |
| [RFC-0004](docs/rfcs/0004-Platform-Membership-Lifecycle.md) | Platform Membership Lifecycle | ACCEPTED |
| [RFC-0005](docs/rfcs/0005-Platform-Audit-Trail.md) | Platform Audit Trail | ACCEPTED v2 |
| [RFC-0006](docs/rfcs/0006-Strategie-Pagination-APIs.md) | Stratégie Pagination APIs | ACCEPTED v2 |
| [RFC-0007](docs/rfcs/0007-Platform-Auth-Authentification.md) | Platform Auth *(sections onboarding → RFC-0018)* | ACCEPTED |
| [RFC-0008](docs/rfcs/0008-Platform-Tenant-Organisations.md) | Platform Tenant *(création publique → RFC-0018)* | ACCEPTED |
| [RFC-0013](docs/rfcs/0013-Format-Identifiants-UUID.md) | Format Identifiants UUID v7 | DRAFT |
| [RFC-0015](docs/rfcs/0015-Event-Bus-Interne.md) | Event Bus Outbox (v1.1) | PROPOSED |
| **[RFC-0018](docs/rfcs/0018-Amendements-B2B-Onboarding.md)** | **Amendements B2B Onboarding** — signup-owner, accept-invite, refresh tokens | DRAFT |
| **[RFC-0019](docs/rfcs/0019-Platform-Notification-B2B-Email-First.md)** | **Platform notification — B2B Email-First** | DRAFT |
| **[RFC-0020](docs/rfcs/0020-Platform-Billing-Hybride-Merchant-Mode.md)** | **Platform billing — Hybride Stripe + Mobile Money** | DRAFT |
| **[RFC-0021](docs/rfcs/0021-Platform-SSO-Enterprise.md)** | **Platform SSO — OIDC + SAML enterprise** | DRAFT |
| **[RFC-0028](docs/rfcs/0028-JWT-RS256-JWKS.md)** | **JWT RS256 + JWKS Endpoint** | DRAFT |
| **[RFC-0038](docs/rfcs/0038-Conventions-Package-Reutilisable.md)** | **Conventions de Package Réutilisable** — contrat Module/Config/Migrations | DRAFT |

### RFCs supersédées (historique préservé)

| RFC | Titre | Supersédé par |
|---|---|---|
| [RFC-0011](docs/rfcs/0011-Platform-Notification-Multi-Canal.md) | Platform notification multi-canal | RFC-0019 + RFC-0036 |
| [RFC-0012](docs/rfcs/0012-Platform-Billing-Abonnements.md) | Platform billing abonnements | RFC-0020 |

### RFCs BaaS — DRAFT DEFERRED (P2+)

| RFC | Titre |
|---|---|
| [RFC-0022](docs/rfcs/0022-Platform-APIKey-PAT-Scopes.md) | Platform apikey — API Keys & PAT scopés |
| [RFC-0023](docs/rfcs/0023-Platform-Webhooks-Sortants-Signes.md) | Platform webhooks_out — Webhooks sortants signés |
| [RFC-0024](docs/rfcs/0024-Kernel-Rate-Limiting.md) | Kernel ratelimit — Rate Limiting & Quotas |
| [RFC-0025](docs/rfcs/0025-Surface-Publique-BaaS-Versioning.md) | Surface Publique BaaS — Calendar Versioning |
| [RFC-0026](docs/rfcs/0026-Platform-Usage-Metering.md) | Platform usage_metering — Comptage d'usage |

### ADRs

| ADR | Titre | Statut |
|---|---|---|
| [ADR-0001](docs/adrs/0001-Go-Stdlib-HTTP-Router.md) | Go Stdlib HTTP Router | ACCEPTED |
| [ADR-0002](docs/adrs/0002-Clean-Architecture-Trois-Niveaux.md) | Clean Architecture 3 Niveaux | ACCEPTED |
| [ADR-0003](docs/adrs/0003-Multi-Tenancy-Shared-Schema-Memberships.md) | Multi-Tenancy Shared Schema | IMPLEMENTED |
| [ADR-0004](docs/adrs/0004-Gestion-Erreurs-AppError.md) | Gestion Erreurs AppError | IMPLEMENTED |
| [ADR-0005](docs/adrs/0005-Logging-Fan-Out-Slog.md) | Logging Fan-Out slog | IMPLEMENTED |
| [ADR-0007](docs/adrs/0007-JWT-HS256-Stdlib.md) | JWT HS256 Stdlib *(→ RFC-0028 pour RS256)* | IMPLEMENTED |
| [ADR-0009](docs/adrs/0009-Audit-Best-Effort-Platform.md) | Audit Best-Effort *(best-effort → DEPRECATED → outbox at-least-once P1-3)* | IMPLEMENTED |

---

## Développement local

### Prérequis

- Go 1.25+
- Docker & Docker Compose

### Lancer l'environnement

```bash
# Infrastructure (PostgreSQL)
make up

# Migrations
make migrate

# Serveur
make run

# Tests
make test
```

### Variables d'environnement clés

| Variable | Défaut | Description |
|---|---|---|
| `APP_ENV` | `development` | `development` / `staging` / `production` |
| `DATABASE_URL` | — | `postgres://user:pass@host:5432/db` |
| `JWT_SECRET` | — | Minimum 32 bytes |
| `PORT` | `8080` | Port HTTP |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `B2B2C_ENABLED` | `false` | Active `/customer/v1/*` + plateformes B2B2C |

---

## Roadmap

| Jalon | Contenu | Cible |
|---|---|---|
| **v0.1.0** | Auth + Tenant + Membership + Audit + Events (pattern Module autonome, migrations embarquées) | Fin P0 |
| **v0.2.0** | Billing + Notification + Permission + User + SSO + Observability | Fin P1 |
| **v0.3.0** | End-customers + Customer Auth + Merchant Mode + White-label | Fin P1bis |
| **v1.0.0** | Stabilisation contrats publics après retour premiers SaaS consommateurs | Post-P1bis |

---

## Licence

Propriétaire — usage réservé à l'organisation [Agile Factory](https://github.com/AgileFactory).
