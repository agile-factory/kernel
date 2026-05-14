# Plan du repo `AgileFactory/kernel`

Référence : [overview.md](overview.md) — architecture d'ensemble. [RFC-0038](rfcs/0038-Conventions-Package-Reutilisable.md) — conventions de package.

`kernel` est une **bibliothèque Go** : tout est importable depuis un repo consommateur via `github.com/AgileFactory/kernel/...`. Aucun code sous `internal/`.

```
kernel/                                 # module github.com/AgileFactory/kernel
├── .github/                            # GitHub workflows, issue templates, PR template
├── assets/                             # diagrammes statiques (SVG)
│   └── img/
│       ├── usecase.svg
│       └── wireflow.svg
├── bin/                                # binaires de build (gitignored)
├── delivery/                           # helpers HTTP partagés (importables)
│   └── http/
│       ├── router.go                   # ApplyGlobalMiddlewares, WithAuth, WithTenant
│       └── middleware/
│           ├── auth.go                 # WithJWT — vérifie le token et injecte user_id en context
│           ├── tenant.go               # WithTenant — résout le tenant (JWT/path/Host)
│           ├── logger.go               # log structuré par requête (slog)
│           └── recovery.go             # capture panic + 500
├── docs/
│   ├── architecture/
│   │   ├── overview.md                 # vue d'ensemble (stack, profils, wiring)
│   │   ├── file-structure.md           # ce fichier
│   │   ├── lokatik.drawio              # diagramme legacy (TODO: à remplacer par C4)
│   │   ├── adrs/                       # Architecture Decision Records (0001 → 0010)
│   │   │   ├── 0001-Go-Stdlib-HTTP-Router.md
│   │   │   ├── 0002-Clean-Architecture-Trois-Niveaux.md
│   │   │   ├── 0003-Multi-Tenancy-Shared-Schema-Memberships.md
│   │   │   ├── 0004-Gestion-Erreurs-AppError.md
│   │   │   ├── 0005-Logging-Fan-Out-Slog.md
│   │   │   ├── 0006-Migrations-SQL-Golang-Migrate.md
│   │   │   ├── 0007-JWT-HS256-Stdlib.md
│   │   │   ├── 0008-Injection-Dependances-Manuelle.md
│   │   │   ├── 0009-Audit-Best-Effort-Platform.md
│   │   │   ├── 0010-Pagination-Uniforme-Data-Total.md
│   │   │   └── template.md
│   │   └── rfcs/                       # Requests For Comments (0001 → 0028 + 0038)
│   │       ├── 0001-Adopt-Flutter-for-Front-End.md
│   │       ├── 0002-Create-Boilerplate-for-Agile-Factory.md
│   │       ├── 0003-Description-For-Multi-Tenant-Use-Case
│   │       ├── 0004-Platform-Membership-Lifecycle.md
│   │       ├── 0005-Platform-Audit-Trail.md
│   │       ├── 0006-Strategie-Pagination-APIs.md
│   │       ├── 0007-Platform-Auth-Authentification.md
│   │       ├── 0008-Platform-Tenant-Organisations.md
│   │       ├── 0009-Platform-User-Profils.md
│   │       ├── 0010-Platform-Permission-RBAC.md
│   │       ├── 0011-Platform-Notification-Multi-Canal.md
│   │       ├── 0012-Platform-Billing-Abonnements.md
│   │       ├── 0013-Format-Identifiants-UUID.md
│   │       ├── 0014-Versioning-API-HTTP.md
│   │       ├── 0015-Event-Bus-Interne.md
│   │       ├── 0016-Observabilite-Metriques-Traces.md
│   │       ├── 0017-Platform-Object-Storage.md
│   │       ├── 0018-Amendements-B2B-Onboarding.md
│   │       ├── 0019-Platform-Notification-B2B-Email-First.md
│   │       ├── 0020-Platform-Billing-Hybride-Merchant-Mode.md
│   │       ├── 0021-Platform-SSO-Enterprise.md
│   │       ├── 0022-Platform-APIKey-PAT-Scopes.md
│   │       ├── 0023-Platform-Webhooks-Sortants-Signes.md
│   │       ├── 0024-Kernel-Rate-Limiting.md
│   │       ├── 0025-Surface-Publique-BaaS-Versioning.md
│   │       ├── 0026-Platform-Usage-Metering.md
│   │       ├── 0028-JWT-RS256-JWKS.md
│   │       ├── 0038-Conventions-Package-Reutilisable.md
│   │       └── template.md
│   ├── development/
│   │   └── conventions.md              # naming, layout, lint rules
│   └── plan.txt                        # pointeur vers le plan de spécialisation
├── examples/                           # apps consommatrices exemples
│   └── saas-b2b/
│       └── main.go                     # MVP B2B (≈130 lignes — composition explicite des Modules)
├── kernel/                             # infra transverse, importable
│   ├── auth/                           # context helpers (UserIDFromContext, IsPlatformAdmin)
│   │   ├── context.go
│   │   └── context_test.go
│   ├── config/                         # chargement env via godotenv
│   │   └── config.go
│   ├── errors/                         # AppError + codes + sentinel + stack (ADR-0004)
│   │   ├── codes.go
│   │   ├── errors.go
│   │   ├── errors_test.go
│   │   ├── sentinel.go
│   │   ├── stack.go
│   │   └── wrap.go
│   ├── logger/                         # slog fan-out (file + stdout) — ADR-0005
│   │   └── logger.go
│   ├── migrator/                       # composeur embed.FS → golang-migrate iofs
│   │   └── migrator.go
│   ├── response/                       # response.JSON / response.Error helpers HTTP
│   │   └── response.go
│   ├── server/                         # http.Server bootstrap + graceful shutdown
│   │   └── server.go
│   └── validator/                      # email/uuid validators
│       └── validator.go
├── platform/                           # briques SaaS réutilisables (importables)
│   ├── audit/                          # audit trail (RFC-0005)
│   │   ├── audit.go                    # Module + Config + EventLogger interface + Migrations FS
│   │   ├── domain/
│   │   │   ├── audit_log.go
│   │   │   ├── audit_test.go
│   │   │   ├── cursor.go
│   │   │   ├── errors.go
│   │   │   └── repository.go
│   │   ├── usecase/
│   │   │   ├── log_event.go            # fire-and-forget (ADR-0009, → outbox P0-8)
│   │   │   ├── list_audit_logs.go      # v1 offset + v2 cursor (RFC-0006)
│   │   │   ├── params.go
│   │   │   └── audit_test.go
│   │   ├── infrastructure/
│   │   │   ├── postgres-repo.go
│   │   │   └── query.go
│   │   ├── delivery/http/
│   │   │   ├── handler.go              # v1 offset
│   │   │   ├── handler_v2.go           # v2 cursor
│   │   │   ├── routes.go               # RegisterRoutes(mux, h, hv2)
│   │   │   ├── dto.go
│   │   │   └── dto_v2.go
│   │   └── migrations/
│   │       ├── 0001_create_audit_logs.{up,down}.sql
│   │       └── 0002_add_audit_logs_cursor_indexes.{up,down}.sql
│   ├── auth/                           # authentification (RFC-0007 → RFC-0018)
│   │   ├── auth.go                     # Module + Config + Migrations FS + JWTService()
│   │   ├── domain/
│   │   │   ├── email.go
│   │   │   ├── email_test.go
│   │   │   ├── errors.go
│   │   │   ├── repository.go
│   │   │   └── user.go
│   │   ├── usecase/
│   │   │   ├── login.go
│   │   │   ├── register.go             # à splitter en signup_owner + accept_invite (P0-10)
│   │   │   ├── refresh.go              # à implémenter (P0-10)
│   │   │   └── auth_test.go
│   │   ├── infrastructure/
│   │   │   ├── bcrypt-hasher.go
│   │   │   ├── jwt-service.go          # HS256 (→ RS256 + JWKS en P1 via RFC-0028)
│   │   │   ├── postgres-repo.go
│   │   │   └── query.go
│   │   ├── delivery/http/
│   │   │   ├── handler.go
│   │   │   ├── routes.go
│   │   │   └── dto.go
│   │   └── migrations/
│   │       ├── 0001_create_users.{up,down}.sql
│   │       └── 0002_drop_users_tenant_id.{up,down}.sql
│   ├── membership/                     # cycle de vie membership (RFC-0004)
│   │   ├── membership.go               # Module + Config + Repository alias + Migrations FS
│   │   ├── domain/
│   │   │   ├── errors.go
│   │   │   ├── membership.go
│   │   │   ├── membership_test.go
│   │   │   └── repository.go
│   │   ├── usecase/
│   │   │   ├── invite.go
│   │   │   ├── accept.go
│   │   │   ├── revoke.go
│   │   │   └── list_members.go
│   │   ├── infrastructure/
│   │   │   ├── postgres-repo.go
│   │   │   └── query.go
│   │   ├── delivery/http/
│   │   │   ├── handler.go
│   │   │   ├── routes.go
│   │   │   └── dto.go
│   │   └── migrations/
│   │       ├── 0001_create_memberships.{up,down}.sql
│   │       └── 0002_add_owner_role.{up,down}.sql
│   └── tenant/                         # organisations tenants (RFC-0008)
│       ├── tenant.go                   # Module + Config + Repository alias + Migrations FS
│       ├── domain/
│       │   ├── errors.go
│       │   ├── repository.go
│       │   └── tenant.go
│       ├── usecase/
│       │   ├── create_tenant.go
│       │   └── find_tenant.go
│       ├── infrastructure/
│       │   ├── postgres-repo.go
│       │   └── query.go
│       ├── delivery/http/
│       │   ├── handler.go
│       │   ├── routes.go
│       │   └── dto.go
│       └── migrations/
│           └── 0001_create_tenants.{up,down}.sql
├── scripts/                            # utilitaires dev / CI
│   └── create-issues.sh
├── .gitignore
├── docker-compose.yml                  # postgres + backend (exemple saas-b2b)
├── Dockerfile                          # multi-stage build de l'exemple saas-b2b
├── go.mod
├── go.sum
├── LICENSE
├── Makefile                            # run / build / test / docker-up / db-shell / db-reset
└── README.md                           # quick-start consommateur
```

## Conventions

- **`internal/` interdit** — tout doit être importable depuis un repo consommateur.
- **Aucun `init()` global** dans `platform/` ou `kernel/`. Toute initialisation passe par `New(Config)` (RFC-0038).
- **Migrations embarquées** par package : `platform/<feature>/migrations/<NNN>_<desc>.{up,down}.sql` + `//go:embed`. Composées par le consommateur via `migrator.Run([]migrator.Source{...})`.
- **Dépendances inter-packages via interfaces** : ex. `audit.EventLogger` consommé par tenant/membership/auth.
- **Repos hors modules** : construits dans `main.go`, passés via `Config{Repo: ...}` à chaque Module. Ce choix casse le cycle de construction audit↔membership (cf. wiring dans `examples/saas-b2b/main.go`).

## Plateformes prévues (squelettes à créer ultérieurement)

Référencées dans le plan de spécialisation, **non encore implémentées** :

| Path prévu | Priorité | RFC | Rôle |
|---|---|---|---|
| `platform/notification/` | P1 | RFC-0019 | Notifications email-first, consume outbox |
| `platform/billing/` | P1 | RFC-0020 | Stripe + Wave + Orange Money + Free Money + Merchant Mode |
| `platform/permission/` | P1 | RFC-0010 | RBAC granulaire |
| `platform/user/` | P1 | RFC-0009 | Profil utilisateur (displayName, avatar, locale…) |
| `platform/sso/` | P1 | RFC-0021 | OIDC + SAML enterprise |
| `platform/storage/` | P2 | RFC-0017 | Object storage S3-compatible |
| `platform/end_customers/` | P1bis | RFC-0033 | Clients finaux du tenant (B2B2C) |
| `platform/customer_auth/` | P1bis | RFC-0034 | Auth scopée tenant pour end-customers |
| `platform/customer_billing/` | P1bis | RFC-0020 (étendu) | Merchant Mode (Stripe Connect, Wave/Orange Marchand) |
| `platform/customer_notifications/` | P1bis | RFC-0036 | Notifications multi-canal end-customers |
| `platform/white_label/` | P1bis | RFC-0035 | Custom domains + branding par tenant |
| `platform/apikey/` | P2+ (différé) | RFC-0022 | API Keys / PAT scopés (BaaS) |
| `platform/webhooks_out/` | P2+ (différé) | RFC-0023 | Webhooks sortants signés HMAC |
| `platform/usage_metering/` | P2+ (différé) | RFC-0026 | Comptage d'usage |
| `platform/devportal/` | P2+ (différé) | RFC-0025 | Documentation OpenAPI + sandbox |

`kernel/events/` (outbox publisher + dispatcher — RFC-0015 v1.1) et `kernel/txctx/` sont à implémenter en P0-8, en préalable aux platforms qui en dépendent.
