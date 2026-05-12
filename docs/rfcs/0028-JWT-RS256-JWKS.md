# _RFC #0028:_ JWT RS256 + JWKS Endpoint

**date:** `2026 May 12`

> **Amende :** [ADR-0007 — JWT HS256 Stdlib](../adrs/0007-JWT-HS256-Stdlib.md)

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC complète ADR-0007 en introduisant **RS256** (RSA-SHA256) et un **endpoint JWKS** (`GET /.well-known/jwks.json`) pour les cas d'usage qui dépassent le HS256 :

- **Tokens SSO** (RFC-0021) : le token doit être validable par le client enterprise sans secret partagé
- **Tokens BaaS** (futur) : les API Keys scopes → tokens RS256 validés localement par les consommateurs
- **Rotation de clés** : current + previous actives simultanément pendant 24h (transition sans interruption)

**HS256 reste le défaut** pour les access tokens internes courts (1h). RS256 est émis uniquement pour les tokens SSO et les tokens destinés à être validés par des tiers.

---

## **Motivation**

### Limites de HS256 pour B2B Enterprise

HS256 est un HMAC symétrique : le même secret sert à signer et à vérifier. Pour valider un token, un tiers doit connaître le secret — ce qui implique de le partager, créant un risque de sécurité.

Scénarios où HS256 est insuffisant :
1. **SSO enterprise (RFC-0021)** : un tenant active OIDC. Ses applications internes doivent valider les tokens émis par la plateforme sans dépendre d'un appel réseau à chaque requête (validation locale via clé publique).
2. **Fédération inter-services** : les microservices du SaaS consommateur valident les tokens sans passer par le kernel.
3. **BaaS futur** : les clients API Keys valident localement les tokens avant de les passer à leurs backends.

RS256 (asymétrique) permet de **publier la clé publique** via JWKS sans exposer la clé privée.

---

## **Proposal**

### 1. Deux services JWT coexistants

```go
// kernel/auth/hs256_service.go
// Tokens internes courts (access tokens memberships, ~1h)
type HS256Service struct {
    secret []byte
}

// kernel/auth/rs256_service.go
// Tokens SSO + JWKS (access tokens RS256, ~1h ; validation externe)
type RS256Service struct {
    currentKey  *rsa.PrivateKey
    previousKey *rsa.PrivateKey  // nil si pas encore de rotation
    currentKID  string           // UUID v7 — change à chaque rotation
    previousKID string
}
```

### 2. Structure JWKS

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "01HZQ8K3VW9X2YM5N6P4Q7R8S",
      "n": "<base64url encoded modulus>",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "use": "sig",
      "alg": "RS256",
      "kid": "01HZQ7J2UV8W1XL4M5O3P6Q7R",
      "n": "<base64url encoded modulus (previous)>",
      "e": "AQAB"
    }
  ]
}
```

Deux clés publiées simultanément pendant 24h après une rotation (current + previous). Les validateurs qui cachent les clés publiques continuent de valider les tokens anciens pendant la fenêtre de transition.

### 3. Endpoint JWKS

```
GET /.well-known/jwks.json
→ 200 application/json { "keys": [...] }
Cache-Control: public, max-age=3600
```

Non authentifié, public. Mis en cache côté client (1h recommandé). Le `kid` dans le header JWT permet au validateur de sélectionner la bonne clé parmi les deux publiées.

### 4. Rotation des clés

```go
// kernel/auth/rs256_service.go

// RotateKey génère une nouvelle paire RSA-2048, promeut current → previous.
// À appeler selon le schedule de rotation (recommandé : 90 jours).
func (s *RS256Service) RotateKey() error {
    newKey, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return err
    }
    s.previousKey = s.currentKey
    s.previousKID = s.currentKID
    s.currentKey  = newKey
    s.currentKID  = newUUIDv7()
    return nil
}
```

La clé RSA-2048 est générée au démarrage si aucune n'est présente en DB/fichier. Pour la production, les clés sont persistées dans la table `jwt_signing_keys` (ou injectées via secret manager).

### 5. Table `jwt_signing_keys`

```sql
CREATE TABLE jwt_signing_keys (
    kid         VARCHAR(128) PRIMARY KEY,
    algorithm   VARCHAR(10) NOT NULL DEFAULT 'RS256',
    private_pem TEXT NOT NULL,    -- clé privée PEM chiffrée (AES-256-GCM)
    public_pem  TEXT NOT NULL,    -- clé publique PEM (non chiffrée)
    status      VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
                    CHECK (status IN ('ACTIVE','PREVIOUS','REVOKED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at  TIMESTAMPTZ
);
```

### 6. Sélection HS256 vs RS256

La sélection se fait à l'émission du token selon le contexte :

| Contexte | Algorithme | Raison |
|---|---|---|
| Login email/password (membership) | **HS256** | Token interne, validation uniquement par le kernel |
| Login SSO (OIDC callback) | **RS256** | Token validable localement par le client enterprise |
| Refresh token (tous cas) | Opaque (SHA-256 hash) | Pas de JWT, token stocké en DB |
| Future API Key (BaaS) | **RS256** | Validable par les backends des développeurs |

### 7. Header JWT — `kid`

Tous les tokens RS256 incluent le `kid` dans le header :

```json
{ "alg": "RS256", "typ": "JWT", "kid": "01HZQ8K3VW9X2YM5N6P4Q7R8S" }
```

Le validateur externe utilise le `kid` pour récupérer la bonne clé publique depuis le JWKS sans recharger l'intégralité du set à chaque requête.

---

## **Drawbacks**

1. **RSA-2048 plus lent que HS256** : sign ~1ms vs ~0.1ms. Acceptable pour les tokens émis à la connexion (pas à chaque requête).
2. **Gestion des clés** : les clés privées RSA doivent être persistées de manière sécurisée (secret manager ou DB chiffrée). En dev local, génération à la volée au démarrage.
3. **JWKS cache côté client** : si un client cache le JWKS longtemps et qu'une clé est révoquée d'urgence, les tokens anciens restent valides jusqu'à expiration. Mitigation : TTL des tokens courts (1h max).

---

## **Alternatives**

### EdDSA (Ed25519) à la place de RSA

- ✅ Plus rapide, clé plus courte, sécurité équivalente
- ❌ Support moins universel dans les bibliothèques tierces (certains IdPs ne supportent pas Ed25519 JWKS)
- **Verdict : différé** — introduire après stabilisation de l'écosystème

### Un seul algorithme RS256 pour tous les tokens

- ✅ Simplicité
- ❌ Overhead CPU pour les millions de tokens internes courts
- **Verdict : rejeté** — HS256 reste optimal pour les tokens internes

---

## **References**

- [ADR-0007 — JWT HS256 Stdlib](../adrs/0007-JWT-HS256-Stdlib.md) *(amendé par ce RFC)*
- [RFC-0021 — Platform SSO](0021-Platform-SSO-Enterprise.md)
- [RFC JSON Web Key (RFC 7517)](https://datatracker.ietf.org/doc/html/rfc7517)
- [RFC JSON Web Algorithms (RFC 7518)](https://datatracker.ietf.org/doc/html/rfc7518)
