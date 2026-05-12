# _RFC #0025:_ Surface Publique BaaS — Versioning Calendar + OpenAPI

**date:** `2026 May 12`

> ⚠️ **DEFERRED — Implémentation P2+.** Ce RFC fixe la vision pour la surface `/public/v1/*` (BaaS). La structure du router est documentée dès maintenant pour permettre son ajout sans refactor.

> **Étend :** [RFC-0014 — Versioning API HTTP](0014-Versioning-API-HTTP.md)

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC distingue les trois surfaces HTTP du boilerplate et définit les règles de versioning pour la surface publique BaaS :

| Surface | Consommateurs | Auth | Versioning |
|---|---|---|---|
| `/api/v1/*` | Frontends internes (memberships) | JWT bearer | Évolution continue (pas de versioning strict) |
| `/customer/v1/*` | Apps end-customers (B2B2C) | JWT scopé tenant | Calendar versioning — apps mobiles long-term |
| `/public/v1/*` | Développeurs externes (BaaS) | API Keys | Calendar versioning `X-API-Version: YYYY-MM-DD` |
| `/admin/v1/*` | Super-admin plateforme | JWT + platform role | Évolution continue |

---

## **Motivation**

RFC-0014 définissait le versioning par `/api/v1/` uniquement. Avec les profils B2B2C (end-customers mobiles) et BaaS (développeurs tiers), deux nouvelles surfaces émergent avec des contraintes de compatibilité différentes :

- **B2B2C** : les apps mobiles end-customers sont installées longtemps → API `/customer/v1/*` doit rester stable (calendar versioning).
- **BaaS** : les développeurs tiers intègrent contre une version précise → calendar versioning à la Stripe (`2026-05-12`), dépréciation 12 mois minimum.

---

## **Proposal**

### 1. Surface `/api/v1/*` — Évolution continue

Consommateurs : frontends SPA/mobile gérés par l'organisation du SaaS. Ils se déploient en même temps que le backend.

- Pas de header de version requis.
- Breaking changes accompagnés d'un déploiement coordonné frontend + backend.
- Pas de période de dépréciation formelle.

### 2. Surface `/customer/v1/*` — Calendar versioning light

Consommateurs : apps mobiles end-customers (installées de manière autonome, mises à jour lentes).

- Header optionnel `X-API-Version: YYYY-MM-DD`
- Si absent : version courante (latest stable)
- Dépréciation : 6 mois minimum de cohabitation des versions
- Stratégie : router vers le handler de la version demandée ou vers latest si absent

### 3. Surface `/public/v1/*` — Calendar versioning strict (BaaS, P2+)

Consommateurs : développeurs externes qui intègrent la plateforme dans leurs propres produits.

**Header obligatoire** : `X-API-Version: YYYY-MM-DD`

Si absent → `400 Bad Request` avec body :
```json
{
  "error": "api_version_required",
  "message": "Header X-API-Version is required. Use the latest: 2026-05-12",
  "latest_version": "2026-05-12",
  "docs": "https://docs.agile-factory.io/api/versions"
}
```

Règles de dépréciation :
- Une version dépréciée est annoncée 12 mois avant sa suppression.
- Le header `Sunset: <date>` est ajouté aux réponses pour les versions en dépréciation.
- Alerte email aux tenants utilisant une version dépréciée (via `notification` platform).

### 4. Structure du router — terrain préparé P0

```go
// backend/internal/delivery/http/router.go
// Structure documentée pour permettre l'ajout du mux BaaS sans refactor

func Build(deps Dependencies, cfg Config) http.Handler {
    // Mux 1 — SaaS B2B + B2B2C admin (toujours actif)
    apiMux := http.NewServeMux()
    deps.Auth.RegisterRoutes(apiMux)
    deps.Tenant.RegisterRoutes(apiMux)
    // ...

    root := http.NewServeMux()
    root.Handle("/api/", middleware.WithJWT(deps.Auth.JWTValidator())(apiMux))
    root.Handle("/admin/", middleware.WithPlatformAdmin()(apiMux))

    // Mux 2 — B2B2C end-customers (activé via B2B2C_ENABLED=true)
    if cfg.B2B2CEnabled {
        customerMux := http.NewServeMux()
        // deps.CustomerAuth.RegisterRoutes(customerMux)
        root.Handle("/customer/", middleware.WithCustomerJWT()(customerMux))
    }

    // Mux 3 — BaaS public (TERRAIN PRÉPARÉ — activé via BAAS_ENABLED=true)
    // if cfg.BaaSEnabled {
    //     publicMux := http.NewServeMux()
    //     deps.APIKey.RegisterRoutes(publicMux)
    //     root.Handle("/public/", middleware.WithAPIKey()(
    //         middleware.WithRateLimit(planLimiter, ExtractByAPIKey)(publicMux)))
    // }

    root.Handle("/.well-known/jwks.json", deps.Auth.JWKSHandler())
    root.Handle("/health", healthHandler())
    root.Handle("/metrics", metricsHandler())

    return root
}
```

### 5. OpenAPI 3.1 — génération automatique (P2+)

La surface `/public/v1/*` sera documentée via OpenAPI 3.1 auto-généré depuis les handlers (reflection + annotations godoc). Endpoint de découverte :

```
GET /public/v1/openapi.json        → spec JSON
GET /public/v1/openapi.yaml        → spec YAML
GET /public/v1/docs                → UI Swagger (embedded)
```

---

## **Références**

- [RFC-0014 — Versioning API HTTP](0014-Versioning-API-HTTP.md) *(étendu par ce RFC)*
- [RFC-0022 — Platform API Keys](0022-Platform-APIKey-PAT-Scopes.md)
- [RFC-0024 — Kernel Rate Limiting](0024-Kernel-Rate-Limiting.md)
- [Stripe API Versioning](https://stripe.com/docs/api/versioning)
