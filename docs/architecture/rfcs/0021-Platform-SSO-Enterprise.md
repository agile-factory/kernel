# _RFC #0021:_ Platform `sso` — SSO Enterprise (OIDC + SAML)

**date:** `2026 May 12`

---

## **Status:** `ACCEPTED` *(implémentation prévue P1-7, cf. plan §6)*

---

## **Summary**

Ce RFC définit la plateforme `sso`, brique P1 qui permet aux tenants enterprise de **déléguer l'authentification de leurs membres** à leur Identity Provider (IdP).

Protocoles : **OIDC v1** (Okta, Auth0, Google Workspace, Azure AD) + **SAML 2.0** (différé v2).

Fonctionnalités : configuration SSO par tenant, **JIT provisioning** (création automatique User + Membership au premier login SSO), claim mapping, mode strict (bloc email/password), tokens RS256 (couplage RFC-0028).

---

## **Motivation**

Les tenants enterprise requièrent une intégration SSO pour :
- **Sécurité** : révocation centralisée (désactiver dans Okta = désactiver tous accès)
- **Compliance** : SOC2/ISO27001 exige souvent le SSO
- **UX** : pas de nouveau mot de passe pour les employés

---

## **Proposal**

### 1. Modèle de domaine

```go
type SSOConfig struct {
    ID              string
    TenantID        string
    Protocol        SSOProtocol  // OIDC | SAML
    IssuerURL       string
    ClientID        string
    ClientSecret    string       // chiffré AES-256-GCM en DB
    Scopes          []string     // ["openid","email","profile"]
    SAMLMetadataXML string       // SAML uniquement
    ClaimMappings   ClaimMappings
    StrictMode      bool         // bloque login email/password pour ce tenant
    IsActive        bool
}

type ClaimMappings struct {
    EmailClaim string
    NameClaim  string
    RoleClaim  string
    RoleMap    map[string]string // "okta_admin" → "ROLE_ADMIN"
}
```

### 2. Use Cases

**`InitiateSSOLoginUseCase`** : génère state token PKCE + redirect URL vers l'IdP.

**`HandleCallbackUseCase`** :
```
1. Valider state (anti-CSRF)
2. Échanger code → id_token (OIDC)
3. Vérifier signature id_token via jwks_uri du issuer
4. JIT Provisioning :
   - User inexistant → créer User + Membership ACTIVE (rôle via ClaimMappings)
   - User existant ACTIVE → OK
   - Membership révoqué → ErrSSOAccessRevoked
5. Émettre access_token RS256 (1h) + refresh_token (30j)
6. event_outbox → auth.sso_login
```

**`ConfigureSSOUseCase`** (ROLE_OWNER) : valide l'issuer URL, chiffre le client_secret, persiste.

### 3. API HTTP

```
GET  /api/v1/sso/{tenant_slug}/initiate       # public — redirect vers IdP
GET  /api/v1/sso/callback?code=...&state=...  # public — reçoit le code

POST  /api/v1/tenants/{id}/sso/config         # ROLE_OWNER
GET   /api/v1/tenants/{id}/sso/config
PATCH /api/v1/tenants/{id}/sso/config
DELETE /api/v1/tenants/{id}/sso/config
```

### 4. Table `sso_configs`

```sql
CREATE TABLE sso_configs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL UNIQUE REFERENCES tenants(id) ON DELETE CASCADE,
    protocol          VARCHAR(10) NOT NULL CHECK (protocol IN ('OIDC','SAML')),
    issuer_url        VARCHAR(500),
    client_id         VARCHAR(255),
    client_secret_enc TEXT,          -- AES-256-GCM
    scopes            TEXT[] NOT NULL DEFAULT ARRAY['openid','email','profile'],
    saml_metadata_xml TEXT,
    email_claim       VARCHAR(100) NOT NULL DEFAULT 'email',
    name_claim        VARCHAR(100) NOT NULL DEFAULT 'name',
    role_claim        VARCHAR(100),
    role_map          JSONB,
    strict_mode       BOOLEAN NOT NULL DEFAULT FALSE,
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sso_state_tokens (
    state         VARCHAR(128) PRIMARY KEY,
    tenant_id     UUID NOT NULL REFERENCES tenants(id),
    code_verifier VARCHAR(128),
    expires_at    TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '5 minutes'
);
```

### 5. JIT Provisioning — règle de rôle

Priorité : `ClaimMappings.RoleMap[claims[RoleClaim]]` → sinon `ROLE_VIEWER`.

Exemples :
- Okta groups → `{"okta_admins": "ROLE_ADMIN"}`
- Azure AD roles → `{"platform.admin": "ROLE_ADMIN"}`

### 6. Mode strict SSO

`strict_mode = true` → `LoginUseCase` retourne `ErrSSORequired` pour les membres du tenant (sauf ROLE_OWNER — anti-lockout). Activable uniquement après `TestSSOConfigUseCase` validé.

### 7. Couplage RS256 (RFC-0028)

Tokens post-SSO émis en RS256 pour permettre validation JWKS chez les clients enterprise.

---

## **Drawbacks**

1. **SAML différé** : la structure `SSOConfig` prévoit `saml_metadata_xml` mais l'implémentation est v2.
2. **Anti-lockout** : strict_mode mal configuré peut bloquer l'Owner. Mitigation : TestSSO obligatoire avant activation.

---

## **Alternatives**

### Déléguer à Auth0 en tant que SSO central

Rejeté — dépendance externe, coût MAU, données identité exposées à un tiers.

---

## **References**

- [RFC-0028 — JWT RS256 + JWKS](0028-JWT-RS256-JWKS.md)
- [RFC-0018 — Amendements B2B Onboarding](0018-Amendements-B2B-Onboarding.md)
- [OpenID Connect Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html)
- [go-oidc — coreos](https://github.com/coreos/go-oidc)
