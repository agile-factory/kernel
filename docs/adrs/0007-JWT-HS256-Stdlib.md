# _ADR #0007:_ Authentification JWT HS256 Sans Bibliothèque Tierce

**date:** `2026 Apr 30`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

L'authentification nécessite la génération et validation de JSON Web Tokens (JWT). L'écosystème Go propose plusieurs bibliothèques JWT (`golang-jwt/jwt`, `lestrrat-go/jwx`), mais leur utilisation introduit une dépendance externe pour une fonctionnalité implémentable avec la stdlib.

La philosophie du projet est de limiter les dépendances aux seuls cas où la stdlib est insuffisante. JWT HS256 repose sur `crypto/hmac`, `crypto/sha256`, et `encoding/base64` — tous disponibles en stdlib.

---

## **Decision**

Implémenter le service JWT **entièrement en stdlib** dans `platform/auth/infrastructure/jwt-service.go` :

- **Algorithme** : HS256 (HMAC-SHA256)
- **Header** : `{ "alg": "HS256", "typ": "JWT" }`
- **Payload** : `{ "sub": user_id, "role": role, "exp": unix_timestamp, "iat": unix_timestamp }`
- **Secret** : configuré via variable d'environnement `JWT_SECRET`
- **Durée** : configurable via `TOKEN_LIFESPAN` (défaut : 24h)

L'interface `TokenGenerator / TokenValidator` est définie dans le usecase (`PasswordVerifier`, `TokenGenerator`) pour respecter l'inversion de dépendance.

---

## **Alternatives Considered**

### `golang-jwt/jwt` (v5)
- ✅ Bibliothèque mature, gère RS256, ES256, JWKS
- ✅ Évite l'implémentation manuelle
- ❌ Dépendance externe pour un algo (HS256) trivial à implémenter
- ❌ Risque de breaking change ou changement de mainteneur
- **Verdict : différé** — à adopter si RS256 ou JWKS deviennent nécessaires (ex: intégration OAuth2)

### RS256 avec clé publique/privée
- ✅ Plus sécurisé (clé privée jamais partagée)
- ✅ Permet la vérification sans secret partagé (microservices)
- ❌ Complexité de gestion des clés pour la v1
- ❌ Requiert une bibliothèque ou un crypto plus lourd
- **Verdict : différé** — pour architecture microservices ou OAuth2

---

## **Consequences**

### **Pros**
- Zéro dépendance JWT externe
- Implémentation transparente et auditable (~50 lignes)
- Interface abstraite : remplacement par une lib tierce = modification de `infrastructure/` uniquement
- Conforme à la philosophie stdlib du projet

### **Cons**
- Pas de support natif pour RS256, ES256, JWKS (imposerait une bibliothèque)
- L'implémentation maison doit être auditée lors de revues de sécurité
- Pas de gestion du `refresh_token` dans la v1 (à implémenter ultérieurement)
- En cas de fuite du `JWT_SECRET`, tous les tokens actifs sont compromis (contre-mesure : rotation du secret + invalidation)
