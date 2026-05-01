# _RFC #0007:_ Platform `auth` — Authentification et Cycle de Vie des Utilisateurs

**date:** `2026 May 01`

---

## **Status:** `ACCEPTED`

---

## **Summary**

Ce RFC définit la plateforme `auth`, brique SaaS fondamentale qui gère l'identité des utilisateurs dans le socle Agile Factory.

La plateforme couvre :
1. **Inscription** (`register`) : création de compte + auto-membership dans un tenant
2. **Connexion** (`login`) : vérification des credentials + émission d'un JWT HS256

Elle est conçue pour être **stateless** (pas de session serveur), **multi-tenant dès le départ** (un utilisateur s'inscrit dans un tenant spécifique), et **extensible** vers OAuth2/magic links sans changement de domaine.

---

## **Motivation**

L'authentification est la brique la plus critique d'un SaaS. Elle doit :
- Être implémentée une seule fois dans le socle et réutilisée par tous les produits
- Ne pas fuiter d'informations sensibles (pas d'énumération d'utilisateurs)
- Supporter le modèle multi-tenant natif (un utilisateur = plusieurs tenants)
- Utiliser des standards éprouvés (bcrypt, JWT)

Sans cette plateforme commune, chaque SaaS réimplémente son propre système d'auth avec des niveaux de sécurité variables.

---

## **Proposal**

### 1. Modèle de domaine

```go
type User struct {
    ID             string
    Email          Email     // value object validé
    HashedPassword string
    Role           UserRole  // ROLE_VIEWER | ROLE_ADMIN (rôle global)
    CreatedAt      time.Time
    UpdatedAt      time.Time
}
```

Le `Role` sur `User` est un **rôle plateforme global** (distinct du rôle par tenant dans `memberships`).

`Email` est un value object avec validation RFC 5322.

### 2. Use Cases

#### `RegisterUseCase`

```
1. Valider le format email
2. Vérifier que le tenant existe et est actif
3. Vérifier l'unicité de l'email
4. Hasher le mot de passe (bcrypt, cost configurable)
5. Persister l'utilisateur
6. Créer un auto-membership ACTIVE dans le tenant
7. Émettre un log d'audit USER_REGISTERED
```

L'auto-membership garantit que l'utilisateur peut immédiatement accéder au tenant sans invitation séparée.

#### `LoginUseCase`

```
1. Valider le format email
2. Trouver l'utilisateur par email
3. Vérifier le mot de passe (bcrypt.CompareHashAndPassword)
4. Générer un JWT HS256 (sub: user_id, role, exp)
5. Émettre un log d'audit USER_LOGGED_IN
```

En cas d'erreur (email inexistant ou mot de passe incorrect), **toujours retourner `ErrInvalidCredentials`** — jamais "email non trouvé" ou "mot de passe incorrect" séparément (anti-énumération).

### 3. Interfaces (inversion de dépendance)

```go
// Définies dans le usecase, implémentées dans infrastructure
type PasswordHasher interface {
    Hash(password string) (string, error)
}
type PasswordVerifier interface {
    Verify(hashedPassword, password string) error
}
type TokenGenerator interface {
    GenerateToken(user *domain.User) (string, error)
}
```

Implémentations :
- `BcryptHasher` (infrastructure/bcrypt-hasher.go)
- `JWTService` (infrastructure/jwt-service.go)

### 4. API HTTP

```
POST /api/v1/auth/register
Body: { "id": "uuid", "tenantId": "uuid", "email": "...", "password": "..." }
→ 201 Created { user, memberships }

POST /api/v1/auth/login
Body: { "email": "...", "password": "..." }
→ 200 OK { token, user, memberships }
```

La réponse inclut `memberships` (liste des tenants de l'utilisateur) pour permettre au client de choisir le contexte tenant.

### 5. Sécurité

| Risque | Mitigation |
|---|---|
| Énumération d'utilisateurs | Toujours retourner `ErrInvalidCredentials` (email + mdp) |
| Brute force | Rate limiting (middleware, v1.1) |
| Fuite de hash | `HashedPassword` jamais sérialisé dans les DTOs |
| Secret JWT | Variable d'environnement `JWT_SECRET`, rotation prévue |
| Mots de passe faibles | Validation minimale : longueur ≥ 8 (extensible) |

---

## **Drawbacks**

### Pas de refresh token en v1
Un JWT expiré force une reconnexion complète. Le refresh token est différé à la v1.1 (nécessite un token store ou blacklist).

### Pas d'OAuth2 / SSO en v1
Inscription uniquement par email/password. OAuth2 (Google, GitHub) est différé.

### Pas de vérification d'email
Un utilisateur peut s'inscrire avec n'importe quel email sans vérification. Différé à v1.1 (nécessite la plateforme `notification`).

### Rôle global `User.Role` vs rôle tenant `Membership.Role`
Deux concepts de rôle coexistent. Le rôle global est pour l'accès plateforme (futur super-admin), le rôle tenant est pour les permissions au sein d'une organisation. Cela peut être confusant pour les nouveaux développeurs.

---

## **Alternatives**

### Session-based auth (cookies + store serveur)
- ✅ Plus simple à révoquer (supprimer la session)
- ❌ Stateful → problème de scaling horizontal
- ❌ CSRF à gérer
- **Verdict : rejeté** — JWT stateless aligné avec la philosophie 12-factor

### OAuth2 / OIDC dès le départ
- ✅ Standard industry, SSO possible
- ❌ Complexité élevée (authorization code flow, refresh, PKCE)
- **Verdict : différé** — Phase 3

### Magic link (connexion par email sans mot de passe)
- ✅ UX simplifiée, plus sécurisé que les mots de passe faibles
- ❌ Nécessite la plateforme `notification` (envoi d'email)
- **Verdict : différé** — Phase 3

---

## **Open Questions**

1. Longueur minimale du mot de passe : 8 ou 12 caractères ? Complexité requise ?
2. Durée du JWT : 24h, 1h avec refresh ? À trancher avec la v1.1.
3. Blacklist des tokens révoqués : Redis (performance) ou table SQL (cohérence) ?
4. Stratégie de rotation du `JWT_SECRET` sans interruption de service ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0003 : Multi-Tenancy Architecture
- RFC-0004 : Platform Membership (auto-membership à l'inscription)
- RFC-0005 : Platform Audit (audit des événements auth)
- ADR-0007 : JWT HS256 Stdlib
- [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- [The Twelve-Factor App — Config](https://12factor.net/config)
