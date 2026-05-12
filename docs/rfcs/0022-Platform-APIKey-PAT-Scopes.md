# _RFC #0022:_ Platform `apikey` — API Keys & PAT Scopés

**date:** `2026 May 12`

> ⚠️ **DEFERRED — Implémentation P2+.** Ce RFC fixe la vision pour le profil BaaS futur. Le package `platform/apikey/` existe en squelette (README uniquement). Implémentation déclenchée quand `BAAS_ENABLED=true` est activé.

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC définit le système d'**API Keys** et de **Personal Access Tokens (PAT)** scopés pour la surface publique BaaS (`/public/v1/*`). Les API Keys permettent aux développeurs de tenants externes d'intégrer les services du boilerplate dans leurs propres applications.

---

## **Motivation**

Le profil BaaS (Backend-as-a-Service) expose une API publique consommée par des machines (intégrations, CI/CD, scripts) et des développeurs externes. Ces consommateurs ne peuvent pas utiliser les JWT bearer classiques (flux OAuth2 interactif). Ils ont besoin de clés statiques à longue durée de vie, scopées finement et révocables.

---

## **Proposal**

### 1. Modèle

```go
type APIKey struct {
    ID          string
    TenantID    string
    CreatedByID string     // user_id du membre qui a créé la clé
    Name        string     // label humain ("CI/CD", "Zapier integration")
    Prefix      string     // "pk_live_" ou "pk_test_" (visible dans l'UI)
    Hash        string     // bcrypt du token complet (jamais stocké en clair)
    Scopes      []string   // ["read:invoices", "write:webhooks", "read:customers"]
    Environment string     // "live" | "test"
    ExpiresAt   *time.Time // nil = pas d'expiration
    LastUsedAt  *time.Time
    RevokedAt   *time.Time
    CreatedAt   time.Time
}
```

Format du token retourné une seule fois à la création : `pk_live_<32-bytes-random-base58>`. Seul le hash bcrypt (cost 10) est persisté.

### 2. Scopes

Les scopes sont alignés sur le RBAC granulaire (RFC-0010) :

```
read:invoices       write:invoices
read:customers      write:customers
read:webhooks       write:webhooks
read:audit          admin:*           # scope super-puissant, requiert confirmation
```

### 3. API HTTP

```
# Gestion des clés (authentifié JWT — ROLE_OWNER/ADMIN du tenant)
POST   /api/v1/tenants/{id}/api-keys
GET    /api/v1/tenants/{id}/api-keys
DELETE /api/v1/tenants/{id}/api-keys/{key_id}  # révocation

# Surface publique BaaS (authentifié via API Key)
Authorization: Bearer pk_live_xxxx
GET /public/v1/...
```

### 4. Middleware `WithAPIKey`

```go
// delivery/http/middleware/api_key.go
// Terrain préparé en P0 — implémenté en P2+ avec BAAS_ENABLED=true
func WithAPIKey(apiKeyRepo APIKeyRepository) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if !strings.HasPrefix(token, "pk_") {
                next.ServeHTTP(w, r)
                return
            }
            key, err := apiKeyRepo.FindByToken(r.Context(), token)
            // ... validate, inject tenant_id + scopes in context
            next.ServeHTTP(w, r)
        })
    }
}
```

### 5. Table `api_keys`

```sql
CREATE TABLE api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    created_by    UUID NOT NULL REFERENCES users(id),
    name          VARCHAR(100) NOT NULL,
    prefix        VARCHAR(10)  NOT NULL,  -- 'pk_live_' | 'pk_test_'
    hash          VARCHAR(255) NOT NULL UNIQUE,
    scopes        TEXT[] NOT NULL DEFAULT '{}',
    environment   VARCHAR(10)  NOT NULL DEFAULT 'live',
    expires_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_api_keys_hash ON api_keys(hash) WHERE revoked_at IS NULL;
CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id) WHERE revoked_at IS NULL;
```

---

## **Drawbacks**

- bcrypt lookup à chaque requête API Key → overhead CPU. Mitigation P2+ : cache Redis LRU (hash → key_id + scopes, TTL 5 min).
- Pas de rotation automatique : les clés sans expiration sont à longue durée de vie. Mitigation : UI rappelle la durée depuis la création, alerte si > 90 jours.

---

## **References**

- [RFC-0010 — Platform Permission RBAC](0010-Platform-Permission-RBAC.md)
- [RFC-0025 — Surface Publique BaaS](0025-Surface-Publique-BaaS-Versioning.md)
- [RFC-0024 — Kernel Rate Limiting](0024-Kernel-Rate-Limiting.md)
