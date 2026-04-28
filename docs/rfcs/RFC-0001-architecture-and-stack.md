# _RFC #0001:_ Architecture & Stack Technique du Socle SaaS

**date:** `2026 Apr 28`

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC définit les fondations techniques et architecturales d'un **socle backend Go réutilisable** destiné à être la base de plusieurs SaaS (immobilier, Social network, shorter workflow product, etc.).

La proposition repose sur trois piliers :

1. **Go en stack 100% standard library** (pas de framework HTTP), avec uniquement quatre dépendances externes ciblées (`pgx`, `bcrypt`).
2. **Clean Architecture en trois niveaux** : `kernel/` (technique pur) → `platform/` (briques SaaS réutilisables) → `modules/` (métier spécifique au SaaS).
3. **Multi-tenancy natif dès le jour 1** via un `tenant_id` propagé par `context.Context`.

L'objectif : disposer d'une fondation pérenne, découplée et scalable, permettant de démarrer un nouveau SaaS en quelques jours sans réécrire les briques transverses (auth, billing, multi-tenancy, notifications).

---

## **Motivation**

### Le problème

Construire plusieurs SaaS successifs sans socle commun mène inévitablement à :

- **Réécriture systématique** des briques transverses (auth, multi-tenancy, billing, notifications) à chaque projet
- **Incohérences entre produits** (deux systèmes d'auth différents, deux logiques de tenant, deux formats d'erreur)
- **Coût de maintenance multiplié** par le nombre de SaaS
- **Time-to-market dégradé** : 2-3 mois perdus en plomberie avant chaque nouveau produit
- **Dette technique exponentielle** : chaque correctif doit être propagé à N projets

### Pourquoi maintenant

Le coût de poser une fondation propre **dès le premier SaaS** est de l'ordre de 3 à 4 semaines. Le coût de la réintroduire **rétroactivement** dans 2-3 SaaS existants se chiffre en **mois-homme**, voire en réécriture complète. L'expérience de la communauté (Stripe, Linear, Shopify) montre que cette décision est l'une des plus structurantes du cycle de vie d'un produit B2B.

### Pourquoi Go

Go est choisi pour ses caractéristiques alignées avec les besoins long-terme d'un socle SaaS :

- **Stabilité du langage** : la compatibilité Go 1.x est garantie depuis 2012, aucun changement cassant majeur
- **Stdlib exceptionnelle** (`net/http`, `database/sql`, `context`, `log/slog`, `crypto/*`) qui couvre 80% des besoins backend
- **Performances natives** sans tuning : 10-50k req/s typique sur une seule instance
- **Compilation statique** : déploiement en un seul binaire, pas de runtime à gérer
- **Concurrence native** via goroutines, idéale pour un backend SaaS multi-tenant
- **Écosystème mature** pour le SaaS (drivers DB, SDK Stripe, AWS, Postgres)

### Pourquoi pas un framework

Les frameworks Go (Gin, Echo, Fiber) présentent des inconvénients critiques pour un socle pluri-annuel :

- **Couplage fort** : `*gin.Context` ou équivalent s'infiltre dans toutes les couches, brisant la Clean Architecture
- **Lock-in** : changer de framework = réécrire toute la couche delivery
- **Dépendance à un mainteneur tiers** : risque d'abandon ou de changements cassants
- **Bénéfice marginal** : depuis Go 1.22, `net/http` gère nativement le routing pattern-based (`GET /users/{id}`)
- **Fiber spécifiquement** : utilise `fasthttp`, incompatible avec l'écosystème HTTP standard Go

### Pourquoi Clean Architecture

Pour un socle qui doit héberger plusieurs SaaS très différents :

- **Séparation domaine / technique** : le métier d'un SaaS immobilier ne dépend pas de Postgres ou Stripe
- **Testabilité** : la logique métier se teste sans DB ni HTTP
- **Substituabilité** : on peut passer de REST à gRPC, de Postgres à un autre SGBD, sans toucher au cœur
- **Réutilisabilité** : le `platform/` reste identique entre SaaS, seul le `modules/` change

---

## **Proposal**

### 1. Stack technique

#### 1.1 Standard library

L'intégralité du code applicatif s'appuie sur la stdlib Go :

| Domaine | Package |
|---|---|
| Routing HTTP | `net/http` (Go 1.22+ patterns) |
| Serveur HTTP | `net/http` |
| Logs structurés | `log/slog` |
| Contexte / cancellation | `context` |
| JSON | `encoding/json` |
| Crypto / hash | `crypto/*` |
| Tests | `testing` |
| Concurrence | `sync`, channels |
| Configuration | `os.Getenv`, `flag` |

#### 1.2 Dépendances externes (4 only)

Seules les fonctionnalités impossibles ou déraisonnables à implémenter from-scratch justifient une dépendance :

```text
golang.org/x/crypto             → bcrypt, argon2 (extension officielle stdlib)
```

Ces dépendances **ne s'infiltrent pas dans le domaine** : elles sont confinées dans `infrastructure/`.

#### 1.3 Dépendances additionnelles (selon besoin futur)

Ajoutées **uniquement quand le besoin est avéré** :

```text
github.com/redis/go-redis/v9    → si cache Redis
github.com/stripe/stripe-go     → si billing Stripe
github.com/aws/aws-sdk-go-v2    → si stockage S3
```

### 2. Architecture en trois niveaux

```text
┌─────────────────────────────────────────┐
│  MODULES (métier SaaS-spécifique)       │  ← change par produit
│  ex: property/, booking/, invoice/      │
├─────────────────────────────────────────┤
│  PLATFORM (briques SaaS réutilisables)  │  ← stable entre SaaS
│  auth/, tenant/, user/, billing/, …     │
├─────────────────────────────────────────┤
│  KERNEL (technique pur)                 │  ← rarement modifié
│  config/, logger/, errors/, response/   │
└─────────────────────────────────────────┘
```

**Règle de dépendance** : chaque niveau ne peut dépendre **que** du niveau inférieur. Jamais l'inverse.

### 3. Structure de dossiers

```text
.
├── cmd/
│   └── api/
│       └── main.go                  ← point d'entrée
│
├── internal/
│   ├── kernel/                      ← niveau 1 : technique pur
│   │   ├── config/
│   │   ├── logger/
│   │   ├── errors/
│   │   ├── response/
│   │   ├── validator/
│   │   └── server/                  ← HTTP server + graceful shutdown
│   │
│   ├── platform/                    ← niveau 2 : socle SaaS
│   │   ├── auth/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── infrastructure/
│   │   │   └── delivery/
│   │   ├── tenant/
│   │   ├── user/
│   │   ├── permission/
│   │   ├── notification/
│   │   ├── audit/
│   │   └── billing/
│   │
│   ├── modules/                     ← niveau 3 : métier
│   │   └── (vide au départ, rempli par chaque SaaS)
│   │
│   ├── delivery/                    ← entrées globales
│   │   └── http/
│   │       ├── middleware/
│   │       │   ├── auth.go
│   │       │   ├── tenant.go
│   │       │   ├── recovery.go
│   │       │   ├── logger.go
│   │       │   └── ratelimit.go
│   │       ├── router.go
│   │       └── server.go
│   │
│   └── bootstrap/                   ← wiring DI
│       ├── container.go
│       └── routes.go
│
├── pkg/                             ← code public réutilisable hors module
├── migrations/                      ← SQL versionné
├── docs/
│   └── rfcs/                        ← ce document et les suivants
├── scripts/
├── docker-compose.yml
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```

#### 3.1 Anatomie d'un module

Chaque feature dans `platform/` ou `modules/` suit la même structure interne :

```text
auth/
├── domain/
│   ├── user.go            ← entity
│   ├── email.go           ← value object
│   ├── repository.go      ← interface (consommée par usecase)
│   └── errors.go
├── usecase/
│   ├── register.go
│   ├── login.go
│   └── refresh.go
├── infrastructure/
│   ├── postgres_repo.go   ← implémentation
│   ├── jwt_service.go
│   └── bcrypt_hasher.go
└── delivery/
    └── http/
        ├── handler.go
        ├── dto.go
        └── routes.go
```

Cette homogénéité permet de **copier-coller un module entier** d'un SaaS à l'autre.

### 4. Inversion de dépendance idiomatique Go

Les interfaces sont définies **côté consommateur**, pas côté implémentation :

```go
// internal/platform/auth/usecase/login.go
package usecase

// Interface définie ICI, dans le package qui l'utilise
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
}

type LoginUseCase struct {
    repo UserRepository  // ← dépend d'une abstraction locale
}
```

L'implémentation `PostgresUserRepository` n'a pas à savoir que cette interface existe : Go satisfait l'interface implicitement (duck typing).

### 5. Multi-tenancy natif

Stratégie : **shared database, shared schema, tenant_id partout**.

#### 5.1 Modèle de données

Toute table métier porte :

```sql
tenant_id UUID NOT NULL REFERENCES tenants(id)
CREATE INDEX idx_<table>_tenant ON <table>(tenant_id, created_at);
```

#### 5.2 Propagation par contexte

Un middleware extrait le `tenant_id` du JWT et l'injecte dans `context.Context` :

```go
func TenantMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        claims := auth.ClaimsFromContext(r.Context())
        ctx := tenant.WithID(r.Context(), claims.TenantID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

Tous les repositories filtrent automatiquement :

```go
func (r *Repo) FindAll(ctx context.Context) ([]*Property, error) {
    tid := tenant.MustFromContext(ctx)
    return r.db.Query(ctx, `SELECT * FROM properties WHERE tenant_id = $1`, tid)
}
```

> **Note** : la stratégie multi-tenant fait l'objet d'un RFC dédié (RFC-0002) qui détaille les arbitrages, garde-fous et alternatives.

### 6. Observabilité dès le départ

Trois piliers, configurés par le `kernel/` :

- **Logs structurés JSON** via `log/slog` avec `request_id`, `tenant_id`, `user_id`
- **Métriques** exposées en `/metrics` (format Prometheus, lib `expvar` stdlib + helpers maison)
- **Traces** : interface `Tracer` abstraite dans `kernel/`, implémentation OpenTelemetry ajoutée si besoin

### 7. Gestion d'erreurs typées

Un type `AppError` centralisé porte un **code stable** consommable côté client :

```go
type AppError struct {
    Code     string  // "USER_NOT_FOUND", "TENANT_QUOTA_EXCEEDED"
    Message  string
    HTTPCode int
    Cause    error
}
```

Les codes sont déclarés dans `kernel/errors/codes.go` et **versionnés** : ils font partie du contrat API.

### 8. Tests à trois niveaux

| Niveau | Cible | Outils |
|---|---|---|
| Unit | `domain/` + `usecase/` (mocks) | `testing` stdlib |
| Integration | `infrastructure/` (vraie DB) | `testing` + `testcontainers-go` |
| End-to-end | API HTTP complète | `testing` + `httptest` |

Cible : **>80% de couverture sur `domain/` et `usecase/`**, pragmatique sur le reste.

### 9. Déploiement

- **Docker multi-stage** : binaire statique, image finale `scratch` ou `distroless` (~15 MB)
- **Migrations** versionnées via `golang-migrate`, exécutées au démarrage en dev, manuellement en prod
- **Configuration** par variables d'environnement uniquement (12-factor app)
- **Graceful shutdown** : `SIGTERM` → drain des connexions HTTP → fermeture DB

### 10. Plan de mise en œuvre

| Phase | Contenu | Durée estimée |
|---|---|---|
| 1 | Bootstrap projet (structure, Docker, Makefile, main.go minimal) | 2 jours |
| 2 | `kernel/` complet (config, logger, errors, response, server) | 3-4 jours |
| 3 | `platform/auth` complet (register, login, refresh, JWT) | 1 semaine |
| 4 | `platform/tenant` + middleware + multi-tenancy bout-en-bout | 1 semaine |
| 5 | `platform/user`, `platform/permission` (RBAC basique) | 1 semaine |
| 6 | Tests d'intégration + CI/CD | 3-4 jours |
| 7 | Documentation + premier module exemple | 3 jours |

**Total : ~4-5 semaines** pour un socle prêt à recevoir son premier SaaS métier.

---

## **Drawbacks**

### Coût de mise en place initial plus élevé

Choisir la stdlib + Clean Architecture impose d'écrire soi-même des helpers (middleware chain, response writers, validation simple) qu'un framework fournirait gratuitement. Coût additionnel estimé : **+1 semaine** sur la phase de bootstrap par rapport à un démarrage Gin/Echo.

### Courbe d'apprentissage pour développeurs juniors

Un dev habitué à Express, NestJS ou Gin peut se sentir désorienté par l'absence de "magie" : pas de décorateurs, pas d'auto-binding, tout est explicite. L'onboarding nécessite une session de mentorat dédiée à la philosophie Go et à la Clean Architecture.

### Verbosité accrue dans la couche delivery

Sans framework, le code de validation/binding HTTP est plus verbeux :

```go
// Sans framework
var dto CreateUserDTO
if err := json.NewDecoder(r.Body).Decode(&dto); err != nil { ... }
if err := dto.Validate(); err != nil { ... }
```

vs

```go
// Avec Gin
var dto CreateUserDTO
if err := c.ShouldBindJSON(&dto); err != nil { ... }
```

Ce coût est mitigé par des helpers maison dans `kernel/response/` et `kernel/validator/`.

### Multi-tenancy "shared schema" : isolation logique uniquement

Un bug applicatif (oubli de filtre `tenant_id`) pourrait théoriquement exposer des données cross-tenant. Mitigations prévues :

- Repository wrappers qui forcent le filtre
- Tests d'isolation systématiques entre deux tenants
- Postgres Row-Level Security activable en defense-in-depth

### Pas d'isolation physique des tenants

Les clients exigeant une isolation forte (santé, défense, conformité spécifique) ne pourront pas être servis sans évolution de la stratégie tenant. Ce cas est documenté comme **hors scope** de la v1.

### Maintenance des helpers maison

Les utilitaires écrits soi-même (middleware, validation, errors) doivent être maintenus par l'équipe. Coût marginal mais réel sur le long terme.

### Risque de sur-architecture pour un MVP

Pour un SaaS très simple (<5 entités, <3 use cases), la structure en 3 niveaux peut paraître excessive. Ce coût est assumé : le socle vise la **scalabilité**, pas la simplicité initiale.

### Pas de communauté framework derrière

Pas de réponse Stack Overflow toute prête pour "comment faire X avec [framework]". Compensé par la qualité de la documentation officielle Go et la stabilité de la stdlib.

---

## **Alternatives**

### Alternative A : Framework full-stack (Gin / Echo)

**Description** : utiliser Gin ou Echo comme framework HTTP principal, avec leurs helpers de binding, validation et middleware.

✅ **Avantages**
- Démarrage plus rapide (gain ~1 semaine)
- Communauté large, beaucoup d'exemples
- Helpers intégrés réduisent le code boilerplate

❌ **Inconvénients**
- `*gin.Context` ou équivalent s'infiltre dans toutes les couches → brise la Clean Architecture
- Lock-in framework : changement = réécriture massive
- Risque de breaking changes ou abandon (ex: cas Buffalo, Revel)
- Bénéfice marginal depuis Go 1.22 (routing stdlib désormais suffisant)

**Verdict** : **rejetée**. Le gain initial ne compense pas le coût de découplage et le risque long-terme sur un socle pluri-annuel.

### Alternative B : Stdlib + chi router

**Description** : `net/http` + le router `go-chi/chi` (lib légère, ~1500 lignes, 100% compatible stdlib).

✅ **Avantages**
- Sub-routers élégants pour `/api/v1`, `/api/v2`
- Middleware chain plus ergonomique
- 100% compatible stdlib, signatures `func(w, r)` identiques
- Aucun lock-in (remplaçable par stdlib en quelques heures)

❌ **Inconvénients**
- Une dépendance supplémentaire pour un gain marginal depuis Go 1.22
- Une lib externe à monitorer (CVE, mises à jour)

**Verdict** : **différée**. Si le routing stdlib s'avère trop limitant à l'usage (notamment pour les sub-routers et la composition de middlewares), `chi` sera réintroduit. Le coût de migration est faible (signatures identiques).

### Alternative C : Architecture en flat packages

**Description** : abandonner les 3 niveaux (kernel/platform/modules) au profit d'une organisation **par feature** plate :

```text
internal/
├── auth/
├── user/
├── tenant/
├── property/
└── booking/
```

✅ **Avantages**
- Plus simple à appréhender
- Idiomatique Go pour petits projets
- Moins de "couches conceptuelles"

❌ **Inconvénients**
- Pas de distinction claire entre code réutilisable (auth, tenant) et code métier-spécifique (property)
- Difficile d'extraire le socle pour démarrer un nouveau SaaS
- Risque de couplages entre features sans frontière explicite

**Verdict** : **rejetée**. L'objectif explicite du socle est la **réutilisation entre SaaS**, ce qui exige une frontière nette entre `platform/` et `modules/`.

### Alternative D : Multi-tenancy par database isolée

**Description** : une base Postgres par tenant, plutôt qu'un `tenant_id` partagé.

✅ **Avantages**
- Isolation physique totale (sécurité, conformité)
- Migrations indépendantes par tenant
- Backup/restore par client

❌ **Inconvénients**
- Coût opérationnel élevé : N bases à monitorer, sauvegarder, migrer
- Connection pool explose au-delà de 50 tenants
- Onboarding nouveau tenant = provisioning DB
- Surdimensionné pour la phase actuelle

**Verdict** : **différée**. Reconsidérée si un client exige cette isolation (santé, finance) ou si le nombre de tenants justifie le sharding.

### Alternative E : Status quo (pas de socle, projet par projet)

**Description** : démarrer chaque SaaS from-scratch en copiant-collant ad-hoc depuis les précédents.

✅ **Avantages**
- Aucun investissement initial
- Liberté totale par projet

❌ **Inconvénients**
- Réécriture systématique des briques transverses (3-6 semaines/projet)
- Divergence inévitable entre produits
- Maintenance × N projets
- Time-to-market dégradé pour chaque nouveau SaaS

**Verdict** : **rejetée**. Incompatible avec l'objectif explicite de réutilisation.

---

## **Open Questions**

Questions explicitement ouvertes pour discussion d'équipe :

1. **DI** : container manuel ou `google/wire` (génération de code) ? À trancher après implémentation du premier module pour évaluer la complexité réelle.

2. **Event bus interne** : Postgres `LISTEN/NOTIFY` (zéro infra) ou NATS (plus scalable) ? À trancher dans un RFC dédié dès qu'un cas d'usage cross-module émerge.

3. **Versioning API** : `/api/v1` dans l'URL ou via header `Accept-Version` ? Recommandation initiale : URL (plus explicite, plus debuggable).

4. **Format ID** : UUID v4 ou v7 (sortable) ? Recommandation initiale : v7 pour le tri chronologique natif et la performance d'index.

---

## **References**

- [Go Standard Library Documentation](https://pkg.go.dev/std)
- [Go 1.22 Release Notes - Routing Enhancements](https://go.dev/blog/routing-enhancements)
- Robert C. Martin, *Clean Architecture*, 2017
- [Stripe Engineering - Designing APIs for humans](https://stripe.com/blog)
- [The Twelve-Factor App](https://12factor.net/)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
