# _RFC #0034:_ Platform `customer_auth` — Authentification des Clients Finaux

**date:** `2026 Jun 28`

---

## **Status:** `ACCEPTED`

> **Review Aziz** : séparation staff/customer claire. `password_hash VARCHAR(60)` → **`VARCHAR(72)`** (marge pour Argon2id en P2). Colonnes `tenant_settings` à ajouter dans `customer_auth/migrations/` (pas `auth/migrations/`, ADR-0006). Le `tenant_id` dans le payload JWT customer suffit pour le middleware — pas besoin de lookup supplémentaire.
> **Review Lucas (AMENDMENTS OBLIGATOIRES)** :
> 1. **`CustomerJWTService` dédié** — ne pas modifier `jwt-service.go` staff. Struct `customerJWTPayload { Sub, TenantID, Aud, Exp, Iat }` dans `customer_auth/infrastructure/`.
> 2. **Rate-limit OTP SMS OBLIGATOIRE** — `CustomerForgotPasswordUseCase` doit vérifier qu'aucun `customer_reset_tokens` non expiré n'a été créé dans les **60 dernières secondes** pour ce `end_customer_id` → retourner `errors.TooManyRequests` sinon. Évite les abus de coût SMS.
> 3. **Résistance à l'énumération d'email** — si l'email n'existe pas dans le tenant, `CustomerLoginUseCase` et `CustomerForgotPasswordUseCase` retournent `ErrInvalidCredentials` (même code, même délai) — jamais `ErrNotFound`. Aligner sur le pattern de `platform/auth/usecase/login.go`.
> 4. **`IsLocked()` clock injectable** — `IsLocked()` utilise `time.Now()` directement → non-déterministe en test. Ajouter `clock func() time.Time` dans `CustomerCredential` ou passer en paramètre : `IsLocked(now time.Time) bool`.

---

## **Summary**

Ce RFC définit le module `platform/customer_auth` : authentification dédiée aux **end-customers** (RFC-0033), distincte de l'auth staff (`platform/auth`).

Les end-customers accèdent à un **portail customer-facing** (app mobile ou web) exposé via `/customer/v1/`. Ils ne doivent jamais obtenir de token valable sur les endpoints `/api/v1/` (réservés aux membres du workspace).

La séparation est garantie par :
- Tables distinctes (`customer_credentials`, `customer_refresh_tokens`)
- Claim JWT `aud: "customer"` vs `aud: "staff"`
- Middleware de route indépendant (`customer_auth.Required`)

---

## **Motivation**

### Problème 1 — Tokens JWT cross-contamination

Si staff et end-customers partagent le même émetteur JWT sans discrimination d'audience, un token end-customer peut théoriquement être présenté sur un endpoint staff (et vice-versa). La séparation par `aud` bloque cette classe d'attaque.

### Problème 2 — Cycle de vie différent

| Aspect | Staff (`platform/auth`) | End-customer (`customer_auth`) |
|--------|------------------------|-------------------------------|
| Création | Invite-only (RFC-0018) | Self-registration ou import tenant |
| MFA | Futur (TOTP) | SMS OTP (CEDEAO-first) |
| Forgot password | Email lien | SMS code OTP ou email |
| Durée access token | 1h | Configurable par tenant (15min–24h) |
| Refresh token TTL | 30j | Configurable par tenant (7j–90j) |

### Problème 3 — Tenant peut désactiver le portail customer

Un SaaS B2B pur n'active pas le portail customer. Le module doit être entièrement conditionné à `tenant_settings.b2b2c_enabled = true`.

---

## **Proposal**

### 1. Modèle domaine

```go
// platform/customer_auth/domain/credential.go
package domain

import "time"

type CustomerCredential struct {
    ID             string
    EndCustomerID  string
    TenantID       string
    PasswordHash   string    // bcrypt, coût configurable
    LastLoginAt    *time.Time
    FailedAttempts int
    LockedUntil    *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// IsLocked retourne true si le compte est verrouillé.
// now est injecté pour la testabilité (pas de time.Now() dans le domaine).
func (c *CustomerCredential) IsLocked(now time.Time) bool {
    return c.LockedUntil != nil && c.LockedUntil.After(now)
}
```

```go
// platform/customer_auth/domain/customer_token.go
package domain

import "time"

// CustomerRefreshToken — même pattern que staff (RFC-0018 §5), tables séparées.
type CustomerRefreshToken struct {
    ID            string
    EndCustomerID string
    TenantID      string
    TokenHash     string    // SHA-256 du token opaque
    ExpiresAt     time.Time
    RevokedAt     *time.Time
    CreatedAt     time.Time
}
```

### 2. Port Repository

```go
// platform/customer_auth/domain/repository.go
package domain

import "context"

type CredentialRepository interface {
    Create(ctx context.Context, cred *CustomerCredential) error
    FindByEndCustomerID(ctx context.Context, endCustomerID string) (*CustomerCredential, error)
    Update(ctx context.Context, cred *CustomerCredential) error
}

type CustomerRefreshTokenRepository interface {
    Create(ctx context.Context, t *CustomerRefreshToken) error
    FindByHash(ctx context.Context, tokenHash string) (*CustomerRefreshToken, error)
    Revoke(ctx context.Context, id string) error
    RevokeAllForEndCustomer(ctx context.Context, endCustomerID string) error
}
```

### 3. JWT Customer — `CustomerJWTService` dédié

**Ne pas modifier `platform/auth/infrastructure/jwt-service.go`** (staff tokens). Créer un service JWT indépendant dans ce module :

```go
// platform/customer_auth/infrastructure/customer_jwt_service.go
package infrastructure

// customerJWTPayload — claims distincts du staff JWT (ajout aud + tenant_id).
type customerJWTPayload struct {
    Sub      string `json:"sub"`       // end_customer_id
    TenantID string `json:"tenant_id"` // tenant scope
    Aud      string `json:"aud"`       // toujours "customer"
    Exp      int64  `json:"exp"`
    Iat      int64  `json:"iat"`
}

// CustomerJWTService émet et valide les tokens customer.
// Même algorithme (HS256 + même secret applicatif) mais audience séparée.
type CustomerJWTService struct {
    secretKey     []byte
    tokenLifespan time.Duration
}
```

Le middleware `customer_auth.Required` valide `aud == "customer"` et injecte `end_customer_id` + `tenant_id` dans le contexte. Tout token staff présenté sur un endpoint `/customer/v1/` sera rejeté car son `aud` est absent ou `!= "customer"`.

Le token customer a le profil de claims :

```json
{
  "sub":       "<end_customer_id>",
  "tenant_id": "<tenant_id>",
  "aud":       "customer",
  "iat":       1234567890,
  "exp":       1234571490
}
```

Le middleware `customer_auth.Required` valide `aud == "customer"`. Le middleware staff `auth.Required` valide `aud == "staff"` (ou absence du claim dans les tokens v0.1.0 — compatibility shim).

La durée du token est configurée via `tenant_settings.customer_token_ttl` (défaut : `1h`).

### 4. Use cases

#### `CustomerRegisterUseCase`

```
Pré-condition : tenant.b2b2c_enabled == true
1. Vérifier qu'un EndCustomer avec cet email existe dans le tenant
   (le tenant importe/crée ses clients via RFC-0033 — pas de self-registration libre)
   Alternative : mode "open registration" si tenant_settings.customer_open_registration == true
2. Vérifier qu'il n'a pas encore de credential
3. Valider password (même policy que staff : 12 chars min)
4. BEGIN TRANSACTION
   4a. INSERT customer_credentials {end_customer_id, password_hash}
   4b. INSERT event_outbox {type: "customer_auth.registered"}
5. COMMIT
6. Retourner {access_token (customer aud), refresh_token}
```

#### `CustomerLoginUseCase`

```
1. Trouver EndCustomer par email dans le tenant
2. Trouver CustomerCredential
3. Vérifier IsLocked() — 429 si verrouillé
4. bcrypt.CompareHashAndPassword
5. Si échec : incrémenter failed_attempts, verrouiller si >= 5 (locked_until = +15min)
6. Si succès : réinitialiser failed_attempts, mettre à jour last_login_at
7. Émettre access_token + refresh_token
```

#### `CustomerRefreshUseCase`

Même pattern rolling-refresh que staff (RFC-0018 §RefreshTokenUseCase).

#### `CustomerLogoutUseCase`

Révoquer le refresh token présenté.

#### `CustomerForgotPasswordUseCase`

```
1. Trouver EndCustomer par email dans le tenant
2. Si phone disponible ET tenant.customer_otp_channel == 'SMS' → émettre SMS OTP (RFC-0036)
3. Sinon → émettre email lien (RFC-0036)
4. INSERT customer_reset_tokens {end_customer_id, token_hash, expires_at: +1h}
5. event_outbox {type: "customer_auth.password_reset_requested"}
```

#### `CustomerResetPasswordUseCase`

```
1. Valider token (existe, non expiré, non consommé)
2. Valider nouveau password
3. BEGIN TRANSACTION
   3a. Mettre à jour password_hash
   3b. Marquer token consumed
   3c. RevokeAllForEndCustomer (logout de toutes les sessions)
4. COMMIT
```

### 5. Schéma SQL

```sql
-- platform/customer_auth/migrations/0001_create_customer_credentials.up.sql

CREATE TABLE customer_credentials (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    end_customer_id  UUID NOT NULL UNIQUE REFERENCES end_customers(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    password_hash    VARCHAR(72) NOT NULL,  -- bcrypt=60, marge pour Argon2id en P2
    last_login_at    TIMESTAMPTZ,
    failed_attempts  INTEGER NOT NULL DEFAULT 0,
    locked_until     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE customer_refresh_tokens (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    end_customer_id  UUID NOT NULL REFERENCES end_customers(id) ON DELETE CASCADE,
    tenant_id        UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash       VARCHAR(64) NOT NULL UNIQUE,
    expires_at       TIMESTAMPTZ NOT NULL,
    revoked_at       TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customer_refresh_token_hash
    ON customer_refresh_tokens(token_hash)
    WHERE revoked_at IS NULL AND expires_at > NOW();

CREATE TABLE customer_reset_tokens (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    end_customer_id  UUID NOT NULL REFERENCES end_customers(id) ON DELETE CASCADE,
    token_hash       VARCHAR(64) NOT NULL UNIQUE,
    expires_at       TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    consumed_at      TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customer_reset_token_hash
    ON customer_reset_tokens(token_hash)
    WHERE consumed_at IS NULL AND expires_at > NOW();
```

```sql
-- tenant_settings : ajout de colonnes customer_auth
ALTER TABLE tenant_settings
    ADD COLUMN customer_open_registration BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN customer_token_ttl         INTERVAL NOT NULL DEFAULT '1 hour',
    ADD COLUMN customer_refresh_ttl       INTERVAL NOT NULL DEFAULT '30 days',
    ADD COLUMN customer_otp_channel       VARCHAR(10) NOT NULL DEFAULT 'EMAIL'
                   CHECK (customer_otp_channel IN ('EMAIL','SMS'));
```

### 6. API HTTP

```
# Portail end-customer (pas de JWT requis pour login/register)
POST /customer/v1/auth/register             CustomerRegister
POST /customer/v1/auth/login                CustomerLogin
POST /customer/v1/auth/refresh              CustomerRefresh
POST /customer/v1/auth/logout               CustomerLogout (customer JWT requis)
POST /customer/v1/auth/forgot-password      CustomerForgotPassword
POST /customer/v1/auth/reset-password       CustomerResetPassword
```

Middleware `customer_auth.Required` protège tous les endpoints `/customer/v1/` sauf auth.

### 7. Middleware `customer_auth.Required`

```go
// delivery/http/middleware/customer_auth.go
func CustomerRequired(jwtSecret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Valider Bearer token, vérifier aud == "customer"
            // Injecter end_customer_id + tenant_id dans le contexte
        })
    }
}
```

---

## **Drawbacks**

1. **Double stack auth** : deux tables refresh, deux middlewares, deux ensembles de use cases. Risque de divergence si la politique de sécurité évolue pour l'un et pas l'autre. Mitigation : `kernel/auth` centralise l'émission JWT — les deux modules l'utilisent.
2. **OTP SMS** (forgot password) : dépend de RFC-0036 et d'un provider SMS. En l'absence du provider configuré, fallback email obligatoire.
3. **brute-force lockout** : 5 tentatives / 15min est conservatif. À configurer par tenant via `tenant_settings.customer_lockout_attempts` (v2).

---

## **Alternatives**

### Un seul module auth avec `user_type: STAFF | CUSTOMER`

- ✅ Code partagé maximal
- ❌ Impossible de garantir l'isolation d'audience JWT sans logique conditionnelle complexe
- ❌ Tables staff et customer mélangées — risque d'escalade de privilèges
- **Verdict : rejeté**

### Déléguer l'auth customer à un IdP tiers (Auth0, Keycloak)

- ✅ Plus robuste pour MFA, social login
- ❌ Dépendance externe — contredit la politique stdlib-first
- ❌ Complexité d'intégration pour les SaaS CEDEAO (latence, coût)
- **Verdict : différé P2+** — port `ExternalCustomerIDP` peut être ajouté en façade ultérieurement

---

## **Plan d'adoption**

1. RFC-0033 accepté (prérequis — `end_customers` table doit exister)
2. RFC-0036 accepté en parallèle (nécessaire pour OTP SMS/email forgot-password)
3. Review Aziz + Lucas
4. Implémenter : migrations → domain → infra → usecase → middleware → delivery → tests
5. Mettre à jour `examples/saas-b2b/main.go` : conditionnel `B2B2C_ENABLED`

---

## **References**

- [RFC-0033 — Platform end_customers](0033-Platform-End-Customers.md) *(prérequis)*
- [RFC-0018 — Amendements B2B Onboarding](0018-Amendements-B2B-Onboarding.md) *(pattern refresh token, réutilisé)*
- [RFC-0036 — Platform customer_notifications](0036-Platform-Customer-Notifications.md) *(OTP + email reset)*
- [ADR-0007 — JWT HS256 Stdlib](../adrs/0007-JWT-HS256-Stdlib.md)
- [RFC-0028 — JWT RS256 + JWKS](0028-JWT-RS256-JWKS.md) *(customer tokens vers RS256 en P2)*
