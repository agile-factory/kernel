# _RFC #0018:_ Amendements B2B Onboarding — Auth + Tenant

**date:** `2026 May 12`

> **Supersède (sections impactées) :** [RFC-0007 — Platform Auth](0007-Platform-Auth-Authentification.md) · [RFC-0008 — Platform Tenant](0008-Platform-Tenant-Organisations.md)

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC amende les plateformes `auth` et `tenant` pour supprimer toute inscription et création de tenant **publique et non authentifiée**, au profit d'un flux **invite-only** adapté aux SaaS B2B.

Changements principaux :
- `POST /api/v1/auth/register` **supprimé** — remplacé par `POST /api/v1/auth/signup-owner` (atomique : tenant + user + ACTIVE OWNER membership + token de vérification email)
- `POST /api/v1/tenants` (public) **supprimé** — la création de tenant se fait uniquement via `signup-owner` (ou via `POST /admin/v1/tenants` réservé super-admin pour onboarding sales-assisted)
- `POST /api/v1/auth/accept-invite` **ajouté** — consume un `invite_token` et crée le user + ACTIVE membership avec le rôle pré-décidé
- `POST /api/v1/auth/verify-email` **ajouté** — validation du token de vérification envoyé à l'inscription
- `POST /api/v1/auth/refresh` **ajouté** — refresh token révocable (table `refresh_tokens`)
- `POST /api/v1/auth/logout` **ajouté** — révocation du refresh token
- Tranche RFC-0007 Open Question #1 (password policy) et #4 (RS256/JWKS)
- Tranche RFC-0008 Open Question #2 (qui peut créer un tenant)

---

## **Motivation**

### Problème 1 — Inscription publique incompatible B2B

`POST /api/v1/auth/register` est accessible sans authentification. N'importe qui peut créer un compte dans n'importe quel tenant existant. En B2B :

- Les workspaces (tenants) sont des espaces privés d'une organisation cliente.
- Un futur collaborateur n'arrive que sur **invitation** de l'Owner ou d'un Admin.
- Le seul flux de création de compte autorisé sans invitation est `signup-owner` (première personne qui crée le workspace).

### Problème 2 — Création publique de tenant

`POST /api/v1/tenants` est public (RFC-0008 Open Question #2 explicitement non résolue). Risques :

- Création de tenants orphelins sans Owner.
- Abus (spam, scraping de slugs).
- Incohérence : un tenant sans Owner ACTIVE ne peut pas être administré.

### Problème 3 — Absence de refresh token

RFC-0007 v1 : pas de refresh token. Un JWT expiré force une reconnexion complète. Pour un SaaS B2B, un access token court (~1h) + refresh token longue durée (30j, révocable) est le standard industry.

### Problème 4 — Password policy non tranchée

RFC-0007 Open Question #1 laissée ouverte. Tranchée ici : 12 caractères minimum + score zxcvbn ≥ 2.

---

## **Proposal**

### 1. Nouveaux use cases `platform/auth`

#### `SignupOwnerUseCase` (atomique — remplace `RegisterUseCase`)

```
1. Valider email + password (12 chars min, zxcvbn ≥ 2)
2. Valider et normaliser le slug du tenant
3. Vérifier unicité (email global + slug tenant)
4. BEGIN TRANSACTION
   4a. Créer le Tenant {id, name, slug, is_active: true}
   4b. Créer le User {id, email, hashed_password, email_verified: false}
   4c. Créer le Membership {user_id, tenant_id, role: ROLE_OWNER, status: ACTIVE}
   4d. Créer un email_verification_token (UUID v7, expires_at: +24h)
   4e. Insérer event_outbox {type: "auth.signup_owner", payload: {user_id, tenant_id, token}}
5. COMMIT
6. Retourner {user_id, tenant_id, access_token (1h), refresh_token (30j)}
```

**Atomicité garantie** : si l'une des étapes 4a–4e échoue, ROLLBACK total. Aucun état partiel en base.

#### `AcceptInviteUseCase`

```
1. Charger l'invite_token (UUID v7) — vérifier exists + non expiré + non consommé
2. Valider email + password (si nouvel utilisateur) OU vérifier que l'utilisateur existant correspond à l'email de l'invite
3. BEGIN TRANSACTION
   3a. Si user inconnu : créer User {email, hashed_password, email_verified: true (l'invite_token vaut vérification)}
   3b. Créer Membership {user_id, tenant_id, role: invite_token.role, status: ACTIVE}
   3c. Marquer invite_token.consumed_at = NOW()
   3d. Insérer event_outbox {type: "auth.invite_accepted", payload: {user_id, tenant_id}}
4. COMMIT
5. Retourner {user_id, access_token, refresh_token}
```

#### `VerifyEmailUseCase`

```
1. Charger l'email_verification_token — vérifier exists + non expiré + non consommé
2. BEGIN TRANSACTION
   2a. users.email_verified = true
   2b. email_verification_token.consumed_at = NOW()
3. COMMIT
```

#### `RefreshTokenUseCase`

```
1. Valider le refresh_token (lookup en DB, non révoqué, non expiré)
2. Rotation : révoquer le refresh_token courant, créer un nouveau
3. Émettre un nouvel access_token (1h)
4. Retourner {access_token, refresh_token}
```

Implémentation "rolling refresh" : chaque appel à `/refresh` émet un nouveau refresh_token et révoque l'ancien. Si un token compromis est utilisé, l'utilisateur légitime détecte un `401` au prochain refresh.

#### `LogoutUseCase`

```
1. Charger le refresh_token (depuis le cookie HTTP-only ou le body)
2. Révoquer (revoked_at = NOW())
3. Retourner 204 No Content
```

### 2. API HTTP mise à jour (`platform/auth`)

```
POST /api/v1/auth/signup-owner          # NOUVEAU — remplace /register
POST /api/v1/auth/accept-invite         # NOUVEAU
POST /api/v1/auth/verify-email          # NOUVEAU
POST /api/v1/auth/login                 # CONSERVÉ (inchangé)
POST /api/v1/auth/refresh               # NOUVEAU
POST /api/v1/auth/logout                # NOUVEAU

DELETE /api/v1/auth/register            # SUPPRIMÉ — 404 après migration
```

### 3. API HTTP mise à jour (`platform/tenant`)

```
# SUPPRIMÉ (public, non authentifié) :
DELETE  POST /api/v1/tenants

# CONSERVÉS :
GET   /api/v1/tenants/{id}              # authentifié (membership actif)

# AJOUTÉS :
PATCH  /api/v1/tenants/{id}            # ROLE_OWNER ou ROLE_ADMIN — met à jour name, settings
DELETE /api/v1/tenants/{id}            # ROLE_OWNER — soft-delete (is_active = false)

# SUPER-ADMIN seulement (platform role check) :
POST /admin/v1/tenants                  # onboarding sales-assisted
DELETE /admin/v1/tenants/{id}/hard      # GDPR hard-delete (future RFC-0027)
```

### 4. Table `email_verification_tokens`

```sql
CREATE TABLE email_verification_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       VARCHAR(128) NOT NULL UNIQUE,  -- UUID v7 URL-safe
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
    consumed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_verification_token ON email_verification_tokens(token)
    WHERE consumed_at IS NULL;
```

### 5. Table `refresh_tokens`

```sql
CREATE TABLE refresh_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash  VARCHAR(64) NOT NULL UNIQUE,  -- SHA-256 du token opaque
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '30 days',
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_token_hash ON refresh_tokens(token_hash)
    WHERE revoked_at IS NULL AND expires_at > NOW();
```

Le token renvoyé au client est le token brut (opaque, 256 bits random). Seul le hash SHA-256 est stocké en base.

### 6. Table `invite_tokens`

```sql
CREATE TABLE invite_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invited_by  UUID NOT NULL REFERENCES users(id),
    email       VARCHAR(255) NOT NULL,      -- email de l'invité
    role        VARCHAR(50)  NOT NULL,      -- rôle à assigner (ROLE_ADMIN, ROLE_AGENT, etc.)
    token       VARCHAR(128) NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '7 days',
    consumed_at TIMESTAMPTZ,
    revoked_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_invite_token ON invite_tokens(token)
    WHERE consumed_at IS NULL AND revoked_at IS NULL;
CREATE INDEX idx_invite_tenant_email ON invite_tokens(tenant_id, email)
    WHERE consumed_at IS NULL;
```

### 7. Migration `users` — ajout `email_verified`

```sql
ALTER TABLE users
    ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN status         VARCHAR(20) NOT NULL DEFAULT 'PENDING_EMAIL'
        CHECK (status IN ('PENDING_EMAIL', 'ACTIVE', 'SUSPENDED', 'DELETED'));

-- Les users existants sont rétroactivement ACTIVE (migration de données)
UPDATE users SET email_verified = TRUE, status = 'ACTIVE';
```

### 8. Password policy (tranche RFC-0007 OQ#1)

- **Longueur minimale** : 12 caractères
- **Score zxcvbn ≥ 2** (réduction des mots de passe trop prévisibles)
- **Pas de règle de complexité arbitraire** (majuscule/chiffre obligatoire — rejeté : patronne NIST SP 800-63B)
- Implémentation : `kernel/validator/password.go` — évaluation zxcvbn en Go (pas de dépendance externe : port Go de la bibliothèque)

### 9. Tranche RFC-0008 OQ#2 — Qui peut créer un tenant ?

**Décision** : personne via l'API publique. Deux chemins uniquement :

1. **`POST /api/v1/auth/signup-owner`** — premier Owner d'un nouveau workspace (self-serve)
2. **`POST /admin/v1/tenants`** — super-admin platform (onboarding sales-assisted, migration)

### 10. Tranche RFC-0007 OQ#4 — RS256 + JWKS

Différé dans **RFC-0028** (JWT RS256 + JWKS Endpoint). Résumé : HS256 est conservé pour les access tokens internes courts. RS256 est introduit pour les tokens SSO (RFC-0021) et les tokens JWKS-validés par des clients externes.

---

## **Drawbacks**

1. **Friction onboarding** : plus de self-registration directe. Un Owner doit d'abord créer son workspace via `signup-owner`, puis inviter ses collègues. Pour un SaaS B2B, c'est le flow attendu.
2. **Breaking change** : `POST /api/v1/auth/register` et `POST /api/v1/tenants` sont supprimés. Pas de clients en production → coût faible, fenêtre idéale.
3. **Invite_token expiré** : si l'invité ne clique pas dans 7 jours, l'invite_token doit être re-créé. L'UI doit permettre le re-send sans intervention manuelle.

---

## **Alternatives**

### Garder l'inscription publique avec email verification obligatoire

- ✅ Moins de friction initiale
- ❌ N'importe qui peut rejoindre un tenant existant (problème de sécurité B2B fondamental)
- ❌ Crée des tenants sans Owner potentiels si le flux `signup-owner` est contourné
- **Verdict : rejeté** — invite-only est la norme SaaS B2B

### Magic link comme seul mode d'invitation

- ✅ Pas de gestion de token custom
- ❌ Nécessite `platform/notification` opérationnel dès P0
- **Verdict : différé** — email avec lien d'acceptation suffit en v1 ; magic link possible en v1.1

---

## **Plan d'adoption**

1. Migration `users` : ajouter `email_verified` + `status` (UPDATE rétroactif)
2. Créer tables `email_verification_tokens`, `refresh_tokens`, `invite_tokens`
3. Implémenter `SignupOwnerUseCase`, `AcceptInviteUseCase`, `VerifyEmailUseCase`, `RefreshTokenUseCase`, `LogoutUseCase`
4. Mettre à jour `platform/auth/delivery/http/routes.go` : enregistrer les nouveaux endpoints, supprimer `/register`
5. Mettre à jour `platform/tenant/delivery/http/routes.go` : retirer `POST /api/v1/tenants`, ajouter PATCH/DELETE/admin
6. Tests : `POST /api/v1/auth/register` → 404 ; `POST /api/v1/auth/signup-owner` → 201 avec tenant + user + membership

---

## **References**

- [RFC-0007 — Platform Auth](0007-Platform-Auth-Authentification.md) *(sections impactées supersédées par ce RFC)*
- [RFC-0008 — Platform Tenant](0008-Platform-Tenant-Organisations.md) *(Open Question #2 résolue ici)*
- [RFC-0004 — Platform Membership](0004-Platform-Membership-Lifecycle.md)
- [RFC-0015 — Event Bus Outbox](0015-Event-Bus-Interne.md)
- [RFC-0028 — JWT RS256 + JWKS](0028-JWT-RS256-JWKS.md) *(à créer)*
- [ADR-0007 — JWT HS256 Stdlib](../adrs/0007-JWT-HS256-Stdlib.md)
- [NIST SP 800-63B — Digital Identity Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)
