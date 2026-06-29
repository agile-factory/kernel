# _ADR #0018:_ Platform `customer_auth` — B2B2C End-Customer Authentication (RFC-0034)

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0034 définit le module `platform/customer_auth` : authentification email + password pour les end-customers dans les portails B2B2C per-tenant. Implémenté sur `feat/p1-v0.2.0` en 2026-06-28 (commit `72bc678`).

---

## **Decision**

### Architecture

Structure standard `platform/` du kernel :
- `domain/` — `CustomerCredential`, `CustomerRefreshToken`, `CustomerResetToken`, ports de repository, erreurs sentinel
- `infrastructure/` — `BcryptHasher`, `CustomerJWTService`, `PostgresCredentialRepository`, `PostgresRefreshTokenRepository`, `PostgresResetTokenRepository`
- `usecase/` — `Register`, `Login`, `Refresh`, `Logout`, `ForgotPassword`, `ResetPassword`
- `delivery/http/` — `CustomerAuthHandler` (6 endpoints), `CustomerRequired` middleware
- `customer_auth.go` — `Module` / `Config` + `RegisterRoutes` + `CustomerRequired`
- `migrations/` — `0001_create_customer_credentials.up/down.sql`, `0002_tenant_settings_customer_auth.up/down.sql`

### CustomerJWTService dédié (Lucas amendment)

Un `CustomerJWTService` distinct du `JWTService` staff est créé. Les tokens customer portent `aud:"customer"` et `tenant_id`. La validation vérifie `aud == "customer"` — un token staff présenté à un middleware customer est rejeté, et vice-versa. Cela prévient la cross-contamination de tokens entre les rôles.

```go
type customerJWTPayload struct {
    Sub      string `json:"sub"`       // end_customer_id
    TenantID string `json:"tenant_id"` // tenant scope
    Aud      string `json:"aud"`       // always "customer"
    Exp      int64  `json:"exp"`
    Iat      int64  `json:"iat"`
}
```

### Email enumeration resistance (Lucas amendment)

Le use case `Login` retourne toujours `ErrInvalidCredentials` en cas d'erreur, que l'email soit inconnu ou que le mot de passe soit incorrect. Jamais `ErrNotFound`. Cela empêche l'énumération d'adresses email via les codes d'erreur HTTP.

```go
cred, err := uc.credRepo.FindByEmail(ctx, in.Email, in.TenantID)
if err != nil {
    return CustomerLoginOutput{}, domain.ErrInvalidCredentials // not ErrCredentialNotFound
}
```

De même, `ForgotPassword` retourne `nil` silencieusement si l'email est inconnu.

### IsLocked() avec clock injectable (Lucas amendment)

```go
func (c *CustomerCredential) IsLocked(now time.Time) bool {
    return c.LockedUntil != nil && c.LockedUntil.After(now)
}
```

`time.Now()` n'est jamais appelé dans le domaine. Le `now` est injecté par le use case. Cela rend les tests déterministes sans mocking du package `time`.

### OTP 60s rate-limit (Lucas amendment)

`ForgotPassword` vérifie `HasRecentToken(ctx, endCustomerID, since)` avant de créer un nouveau token de reset. Si un token non-consommé existe dans les 60 dernières secondes, `ErrOTPRateLimit` est retourné. Cela prévient le spam de codes OTP par SMS.

```go
since := now.Add(-60 * time.Second)
has, err := uc.resetRepo.HasRecentToken(ctx, cred.EndCustomerID, since)
if has { return domain.ErrOTPRateLimit }
```

### Rolling refresh token

Le `Refresh` use case implémente un rolling-refresh atomique : l'ancien token est révoqué et un nouveau est émis dans la même transaction. Le token raw est généré via `crypto/rand` (32 bytes → base64url), stocké en base64url chez le client. Seul le SHA-256 hex (64 chars) est stocké en DB.

### PasswordHasher interface combinée

```go
type PasswordHasher interface {
    Hasher          // Hash(password string) (string, error)
    PasswordVerifier // Verify(password, hash string) error
}
```

La `Config` accepte un seul `PasswordHasher` qui implémente les deux aspects. `BcryptHasher` (bcrypt cost=10) est l'implémentation de production.

### Lookup email via JOIN SQL

Le `PostgresCredentialRepository.FindByEmail` fait un JOIN avec `end_customers` :

```sql
SELECT cc.* FROM customer_credentials cc
JOIN end_customers ec ON ec.id = cc.end_customer_id
WHERE lower(ec.email) = lower($1) AND cc.tenant_id = $2
```

Ce join cross-module au niveau SQL (pas au niveau domaine) est acceptable pour un schéma partagé multi-tenant. La comparaison `lower()` assure la case-insensitivity.

---

## **Consequences**

- **Positif :** Les tokens customer sont structurellement incompatibles avec les tokens staff — pas de privilege escalation par token replay.
- **Positif :** Email enumeration résistance conforme à OWASP Authentication Cheat Sheet.
- **Positif :** `IsLocked(now time.Time)` permet des tests déterministes de lockout sans dépendance à `time.Now()`.
- **Positif :** OTP rate-limit côté serveur évite le spam SMS sans infrastructure Redis.
- **Négatif :** Le JOIN SQL avec `end_customers` crée un couplage infrastructure entre `customer_auth` et `end_customers`. Ce couplage est intentionnel pour les portails B2B2C et est documenté ici.
