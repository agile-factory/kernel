# `platform/tenant/` — Tenants (organisations)

> ✅ **Implémenté** — RFC-0008 (création publique supersédée par RFC-0018 §9) + RFC-0018 §3.

Modélise l'unité d'isolation multi-tenant (`tenants`) et ses préférences (`tenant_settings` 1:1). Couvre :

- Lecture / mise à jour / suppression d'un tenant.
- Gestion des `tenant_settings` (branding, locale, currency, data residency, B2B2C flags).
- Création super-admin via `POST /admin/v1/tenants` (sales-assisted onboarding RFC-0018 §9).

La création publique d'un tenant se fait via `auth.SignupOwner` (qui orchestre tenant+user+membership+token en une seule tx) — voir [`platform/auth/README.md`](../auth/README.md).

---

## Wiring

```go
tenantRepo         := tenantinfra.NewPostgresTenantRepository(db)
tenantSettingsRepo := tenantinfra.NewPostgresTenantSettingsRepository(db)

tenantMod := tenant.New(tenant.Config{
    Pool:           db,
    Repo:           tenantRepo,
    SettingsRepo:   tenantSettingsRepo,
    MembershipRepo: memberRepo,                  // perm-checks HTTP
    AuditLogger:    auditMod.EventLogger(),
    Logger:         log,
})
tenantMod.RegisterRoutes(mux)
```

---

## Config

| Champ | Type | Requis | Rôle |
|---|---|---|---|
| `Pool` | `*pgxpool.Pool` | ✅ | Création tx (tenant + settings + audit). |
| `Repo` | `Repository` | ✅ | CRUD tenants. |
| `SettingsRepo` | `SettingsRepository` | ✅ | CRUD tenant_settings. |
| `MembershipRepo` | `memberdomain.MembershipRepository` | ✅ | OWNER/ADMIN gating sur PATCH/DELETE/settings. |
| `AuditLogger` | `audit.EventLogger` | ✅ | Trace `TENANT_CREATED`. |
| `Logger` | `*slog.Logger` | ✅ | — |

---

## Endpoints

| Méthode | Route | Garde | Description |
|---|---|---|---|
| `GET` | `/api/v1/tenants/{id}` | Membership du tenant | Détails. |
| `GET` | `/api/v1/tenants/slug/{slug}` | Membership du tenant | Détails par slug (lookup public-domain ou onboarding). |
| `PATCH` | `/api/v1/tenants/{id}` | OWNER/ADMIN | Update `name` uniquement (slug immuable). |
| `DELETE` | `/api/v1/tenants/{id}` | OWNER | Soft-delete (`is_active = false`). |
| `GET` | `/api/v1/tenants/{id}/settings` | Membership | Lit `tenant_settings`. |
| `PATCH` | `/api/v1/tenants/{id}/settings` | OWNER/ADMIN | Sparse update (pointeurs nullable). |
| `POST` | `/admin/v1/tenants` | Platform super-admin | Sales-assisted onboarding (RFC-0018 §9). |

Détails : [`docs/api/openapi.yaml`](../../docs/api/openapi.yaml).

---

## Migrations

| Version | Fichier | Effet |
|---|---|---|
| 0001 | `create_tenants` | Table + `UNIQUE(name)`, `UNIQUE(slug)`. |
| 0002 | `create_tenant_settings` | 1:1 avec `tenants` (PK = `tenant_id`). Branding + locale + currency + `data_region` + `payment_region` + B2B2C flags. |

Embed FS : `tenant.Migrations`. Ordre : **avant** `auth` + `membership` (FKs).

### Tenant settings — champs métier

| Champ | Type | Défaut | Notes |
|---|---|---|---|
| `locale` | varchar(10) | `en-US` | i18n. |
| `currency` | char(3) | `USD` | ISO 4217. |
| `data_region` | varchar(8) | `eu` | `eu` / `us` / `apac` — pinning GDPR (RFC-0027). |
| `payment_region` | varchar(20) | `INTERNATIONAL` | `INTERNATIONAL` (Stripe) ou `CEDEAO` (Stripe + Wave + Orange/Free Money) — RFC-0020. |
| `merchant_mode` | bool | `false` | Active la facturation des clients finaux (P1bis B2B2C). |
| `b2b2c_enabled` | bool | `false` | Surface `/customer/v1/*` (P1bis). |
| `custom_domain` | varchar(253) | `null` | RFC 1035 max. UNIQUE PARTIAL — Host-header routing B2B2C. |

---

## Références

- [RFC-0008](../../docs/architecture/rfcs/0008-Platform-Tenant-Organisations.md) — Tenant Organisations.
- [RFC-0018](../../docs/architecture/rfcs/0018-Amendements-B2B-Onboarding.md) §3 + §9 — création publique + admin.
- [ADR-0003](../../docs/architecture/adrs/0003-Multi-Tenancy-Shared-Schema-Memberships.md) — shared-schema multi-tenancy.
