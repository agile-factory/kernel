# `platform/auth/` — Authentification

> ✅ **Implémenté** — RFC-0007 (sections onboarding supersédées par RFC-0018) + ADR-0007. JWT HS256 stdlib + refresh tokens hashés SHA-256, rolling rotation.

Module d'authentification du profil **SaaS B2B**. Couvre l'onboarding (signup-owner + accept-invite), la vérification email, le login / refresh / logout. La surface JWT (RS256 + JWKS) pour SSO/Enterprise arrive en P1-7 via RFC-0028.

---

## Wiring

```go
userRepo         := authinfra.NewPostgresUserRepository(db)
refreshTokenRepo := authinfra.NewPostgresRefreshTokenRepository(db)
emailVerifRepo   := authinfra.NewPostgresEmailVerificationTokenRepository(db)

authMod := auth.New(auth.Config{
    Pool:               db,
    UserRepo:           userRepo,
    TenantRepo:         tenantRepo,
    TenantSettingsRepo: tenantSettingsRepo,
    MembershipRepo:     memberRepo,
    InviteTokenRepo:    inviteTokenRepo,
    RefreshTokenRepo:   refreshTokenRepo,
    EmailVerifRepo:     emailVerifRepo,
    MembershipListUC:   memberMod.ListUC,

    AuditLogger:        auditMod.EventLogger(),
    Publisher:          bus.Publisher(),  // optional — nil = events skipped
    JWTSecret:          cfg.JWTSecret,
    TokenLifespan:      cfg.TokenLifespan,
    RefreshTokenTTL:    30 * 24 * time.Hour, // optional, default = 30j
    BCryptCost:         0,                   // 0 = bcrypt.DefaultCost
    Logger:             log,
})
authMod.RegisterRoutes(mux)
```

`UserRepo`, `RefreshTokenRepo`, `EmailVerifRepo` sont construits hors du module pour permettre le partage avec d'autres modules (typiquement `membership.UserLookup` = `auth.UserRepo`).

---

## Config — champs critiques

| Champ | Type | Requis | Rôle |
|---|---|---|---|
| `Pool` | `*pgxpool.Pool` | ✅ | Transactions multi-tables (SignupOwner crée tenant+user+membership+token dans la même tx). |
| `JWTSecret` | `string` | ✅ | Clef HMAC-SHA256, **≥ 32 bytes**. Rotation : voir runbook. |
| `TokenLifespan` | `time.Duration` | ✅ | Durée de validité des access tokens (recommandé : 1h). |
| `RefreshTokenTTL` | `time.Duration` | ⏳ | Durée de validité des refresh tokens (défaut 30j). |
| `BCryptCost` | `int` | ⏳ | Coût bcrypt (défaut `bcrypt.DefaultCost = 10`). 12 recommandé en prod. |
| `Publisher` | `events.Publisher` | ⏳ | Si nil, les événements `auth.signup_owner` / `auth.invite_accepted` sont skippés (mode dégradé). |

---

## Endpoints

| Méthode | Route | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/auth/signup-owner` | — | Crée tenant + user + membership OWNER + token de vérification email dans une seule tx. Émet `auth.signup_owner`. |
| `POST` | `/api/v1/auth/accept-invite` | — | Consomme un `invite_token`, crée le user si nouveau (sinon exige le mot de passe actuel du compte), crée membership ACTIVE. Émet `auth.invite_accepted`. |
| `POST` | `/api/v1/auth/verify-email` | — | Consomme un email-verification token. Idempotent. |
| `POST` | `/api/v1/auth/login` | — | Email + password → access + refresh tokens. |
| `POST` | `/api/v1/auth/refresh` | — | Rolling rotation : révoque le refresh présenté, émet une nouvelle paire. |
| `POST` | `/api/v1/auth/logout` | — | Révoque le refresh-token fourni (idempotent). |

Détails complets (schémas, codes d'erreur AppError, exemples) : voir [`docs/api/openapi.yaml`](../../docs/api/openapi.yaml).

---

## Migrations

| Version | Fichier | Effet |
|---|---|---|
| 0001 | `create_users` | Table `users` (avec `tenant_id` legacy, supprimé en 0002). |
| 0002 | `drop_users_tenant_id` | Retire `users.tenant_id` (multi-tenancy via memberships maintenant). |
| 0003 | `add_email_verification_to_users` | Colonne `email_verified_at`. |
| 0004 | `create_email_verification_tokens` | Token opaque (UUID v7-like, 128 chars URL-safe), TTL 24h. |
| 0005 | `create_refresh_tokens` | `token_hash` SHA-256, `tenant_id` nullable, TTL 30j. |

Embed FS : `auth.Migrations`. Doit s'exécuter **après** `tenant` (FK potentiel sur tenants pour les refresh-tokens tenant-scoped).

---

## Sécurité — invariants

| Invariant | Implémentation |
|---|---|
| Mots de passe ≥ 12 caractères | Validation côté DTO (`SignupOwnerRequest.Validate`, `AcceptInviteRequest.Validate`). |
| Hash bcrypt (jamais stocké en clair) | `infrastructure.BcryptHasher`. Coût configurable. |
| Refresh tokens stockés en `SHA-256(token)` | Un dump DB ne permet pas de minter de nouveaux tokens. |
| Rolling rotation | Chaque `/refresh` révoque l'ancien token. Replay détecté par `UNIQUE(token_hash)` filtré sur `revoked_at IS NULL`. |
| Pas de Login pour user non-vérifié | Login renvoie `EMAIL_NOT_VERIFIED` tant que `email_verified_at IS NULL`. |
| JWT signé HS256 | ADR-0007. Migration RS256 + JWKS en P1-7 — RFC-0028. |

Détails : [`docs/architecture/technical-deepdive.md`](../../docs/architecture/technical-deepdive.md#auth-token-lifecycle).

---

## Références

- [RFC-0018](../../docs/architecture/rfcs/0018-Amendements-B2B-Onboarding.md) — onboarding B2B amendé (loi actuelle).
- [RFC-0007](../../docs/architecture/rfcs/0007-Platform-Auth-Authentification.md) — spec auth (login/refresh toujours valides).
- [RFC-0028](../../docs/architecture/rfcs/0028-JWT-RS256-JWKS.md) — migration RS256 + JWKS (P1-7).
- [ADR-0007](../../docs/architecture/adrs/0007-JWT-HS256-Stdlib.md) — choix HS256 stdlib.
