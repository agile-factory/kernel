# _RFC #0038:_ Conventions de Package Réutilisable

**date:** `2026 May 12`

---

## **Status:** `ACCEPTED + IMPLEMENTED (P0-0a/b/c)`

---

## **Summary**

Ce RFC définit les conventions auxquelles tous les packages `platform/<feature>` du repo [`AgileFactory/kernel`](https://github.com/AgileFactory/kernel) doivent se conformer pour être consommables, indépendants et stables comme bibliothèque Go.

`kernel` n'est pas une application : c'est une bibliothèque que les repos SaaS de l'organisation importent. Pour que cette importation reste fiable d'un produit à l'autre, chaque package suit un **contrat strict** : struct `Module`, struct `Config`, migrations embarquées via `embed.FS`, helper `RegisterRoutes(mux)`, aucune `init()` globale, dépendances inter-packages via interfaces uniquement.

---

## **Motivation**

L'organisation Agile Factory développe plusieurs produits SaaS sur le même socle. Sans conventions partagées :

- Chaque package évolue avec sa propre signature, ses propres patterns de DI, ses propres migrations.
- Le SaaS consommateur écrit du code de glue spécifique pour chaque package → friction, divergence, copier-coller fragile.
- Les breaking changes deviennent imprévisibles : une mise à jour de `kernel` casse les SaaS en aval sans signalement clair.
- Tester en isolation devient impossible si les packages dépendent de singletons ou d'`init()` globaux.

Avec un contrat unique :
- Le SaaS consommateur sait exactement comment instancier et brancher chaque package.
- Les breaking changes sont détectés à la compilation (changement de signature `Config`).
- Les tests unitaires fonctionnent avec des mocks via la `Config`.
- L'onboarding d'un nouveau dev sur n'importe quel package suit le même mental model.

---

## **Proposal**

### 1. Structure de dossier d'un package `platform/<feature>`

```
platform/<feature>/
├── <feature>.go             # API publique exportée (Module + New)
├── config.go                # struct Config (dépendances + options)
├── doc.go                   # godoc du package
├── domain/                  # entities, value objects, repository interfaces
│   ├── entity.go
│   ├── repository.go        # interface (jamais d'impl)
│   └── errors.go
├── usecase/                 # use cases (1 fichier par action)
│   ├── action_one.go
│   └── action_one_test.go   # tests unitaires (mocks via Config)
├── infrastructure/          # implémentations concrètes (postgres, http client, etc.)
│   ├── postgres_repo.go
│   └── queries.go           # constantes SQL
├── delivery/http/           # handlers HTTP, DTOs, routes
│   ├── handler.go
│   ├── dto.go
│   └── routes.go            # RegisterRoutes(mux)
└── migrations/              # .sql versionnés (format <NNN>_<desc>.{up,down}.sql)
    ├── 0001_create_table.up.sql
    ├── 0001_create_table.down.sql
    └── ...
```

### 2. Fichier `<feature>.go` — API publique

```go
// Package auth fournit l'authentification et le cycle de vie des utilisateurs.
//
// Voir RFC-0007 + RFC-0018 pour la spécification.
package auth

import (
    "embed"
    "net/http"

    // ... imports internes
)

// Migrations contient les fichiers SQL .up/.down.sql à appliquer.
// Le SaaS consommateur les compose avec ses propres migrations via kernel/migrator.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Module est la racine du package auth. Construit via New(Config).
type Module struct {
    cfg     Config
    repo    domain.UserRepository
    hasher  domain.PasswordHasher
    jwt     domain.JWTService
    UC      UseCases // exposé pour usage avancé / tests
}

// UseCases regroupe les use cases exposés.
type UseCases struct {
    SignupOwner   *usecase.SignupOwner
    AcceptInvite  *usecase.AcceptInvite
    VerifyEmail   *usecase.VerifyEmail
    Login         *usecase.Login
    RefreshToken  *usecase.RefreshToken
    Logout        *usecase.Logout
}

// New construit le Module avec ses dépendances injectées.
// Aucune ressource externe n'est ouverte ici (DB pool, HTTP client, etc.
// sont fournis par le consommateur via Config).
func New(cfg Config) *Module {
    cfg.applyDefaults()
    repo := infrastructure.NewPostgresUserRepository(cfg.DB)
    // ... construit la chaîne de dépendances
    return &Module{
        cfg:    cfg,
        repo:   repo,
        // ...
    }
}

// RegisterRoutes enregistre les endpoints HTTP du package sur le mux fourni.
// Le consommateur peut envelopper avec ses propres middlewares avant/après.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
    mux.HandleFunc("POST /api/v1/auth/signup-owner", m.handleSignupOwner)
    mux.HandleFunc("POST /api/v1/auth/accept-invite", m.handleAcceptInvite)
    mux.HandleFunc("POST /api/v1/auth/verify-email", m.handleVerifyEmail)
    mux.HandleFunc("POST /api/v1/auth/login", m.handleLogin)
    mux.HandleFunc("POST /api/v1/auth/refresh", m.handleRefresh)
    mux.HandleFunc("POST /api/v1/auth/logout", m.handleLogout)
}

// JWTValidator expose un validator JWT réutilisable par les autres
// packages / modules du SaaS consommateur (via middleware WithJWT).
func (m *Module) JWTValidator() middleware.JWTValidator {
    return m.jwt
}
```

### 3. Fichier `config.go` — struct Config

```go
package auth

import (
    "log/slog"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/AgileFactory/kernel/kernel/events"
    "github.com/AgileFactory/kernel/platform/notification"
)

// Config contient toutes les dépendances et options du Module auth.
// Le consommateur instancie cette struct dans son main.go puis appelle New(cfg).
type Config struct {
    // Dépendances obligatoires (panic si nil)
    DB        *pgxpool.Pool
    Log       *slog.Logger
    JWTSecret []byte

    // Dépendances optionnelles (interfaces — peuvent être mockées en test)
    Publisher       events.Publisher        // pour émettre user.registered, etc.
    NotifDispatcher notification.Dispatcher // pour envoyer emails (signup, invite, reset)

    // Options avec défauts (applyDefaults remplit si zéro)
    AccessTokenTTL  time.Duration // défaut 1h
    RefreshTokenTTL time.Duration // défaut 30 jours
    BcryptCost      int           // défaut 12
}

func (c *Config) applyDefaults() {
    if c.AccessTokenTTL == 0 {
        c.AccessTokenTTL = time.Hour
    }
    if c.RefreshTokenTTL == 0 {
        c.RefreshTokenTTL = 30 * 24 * time.Hour
    }
    if c.BcryptCost == 0 {
        c.BcryptCost = 12
    }
}

func (c *Config) validate() error {
    if c.DB == nil {
        return errors.New("auth.Config: DB is required")
    }
    if c.Log == nil {
        return errors.New("auth.Config: Log is required")
    }
    if len(c.JWTSecret) < 32 {
        return errors.New("auth.Config: JWTSecret must be at least 32 bytes")
    }
    return nil
}
```

### 4. Migrations embarquées

Chaque package porte ses propres migrations dans `migrations/`, suivant le format requis par `golang-migrate` : `<NNN>_<description>.{up,down}.sql` (préfixe numérique strict, exigence du parser `iofs`). L'isolation par-package rend le préfixe `<feature>_` superflu — chaque `embed.FS` a sa propre racine et son propre numéro de version.

```
platform/auth/migrations/
├── 0001_create_users.up.sql
├── 0001_create_users.down.sql
├── 0002_drop_users_tenant_id.up.sql
└── 0002_drop_users_tenant_id.down.sql
```

Exposées via `embed.FS` au niveau du package :

```go
//go:embed migrations/*.sql
var Migrations embed.FS
```

Le consommateur compose toutes les migrations via `kernel/migrator` :

```go
migrator.Run(cfg.DatabaseURL, []migrator.Source{
    {Name: "tenant",     FS: tenant.Migrations,     Path: "migrations"},
    {Name: "auth",       FS: auth.Migrations,       Path: "migrations"},
    {Name: "membership", FS: membership.Migrations, Path: "migrations"},
    {Name: "audit",      FS: audit.Migrations,      Path: "migrations"},
    // ... éventuellement les migrations métier du SaaS consommateur
})
```

`migrator.Run` itère sur les sources **dans l'ordre fourni** (le consommateur l'utilise pour respecter les dépendances : `tenant` → `auth` → `membership` → `audit`). Chaque source obtient sa propre table de versions `schema_migrations_<name>` via le query param `x-migrations-table` du driver postgres, ce qui permet à chaque package d'évoluer indépendamment sans collision de numéros.

`migrator.Down(...)` parcourt les sources en ordre inverse pour un rollback propre.

### 5. Règles interdites

1. **Pas d'`init()` global** dans un package `platform/`. Toute initialisation passe par `New(Config)`.
2. **Pas de variable globale mutable** exportée ou non. Les constantes sont OK.
3. **Pas de singleton**. Chaque appel à `New()` produit un Module indépendant (utile pour tests, multi-tenancy avancée, sandbox).
4. **Pas d'import depuis `cmd/`** ou un module métier consommateur. Les packages `platform/` sont feuilles côté kernel.
5. **Pas d'import direct entre packages `platform/`** : passer par des interfaces définies dans le package consommé (ex: `notification.Dispatcher`).
6. **Pas de panique sauf au `New(Config)`** si la config est invalide (échec rapide au démarrage). Une fois construit, le Module retourne des erreurs.
7. **Pas de `log.Println` / `fmt.Println`** : tout logging passe par `cfg.Log` (slog).
8. **Pas de `os.Getenv`** dans `platform/` : le consommateur charge la config dans `cfg/` puis passe les valeurs via `Config`.

### 6. Dépendances inter-packages via interfaces

Si `platform/membership` doit envoyer un email d'invitation, il ne dépend **pas** de `platform/notification` directement, mais d'une interface définie côté `notification` :

```go
// platform/notification/dispatcher.go
package notification

type Dispatcher interface {
    Dispatch(ctx context.Context, event Event) error
}
```

Et `platform/membership.Config` contient :

```go
type Config struct {
    // ...
    NotifDispatcher notification.Dispatcher
}
```

Le consommateur branche les deux dans son `main.go` :

```go
notif := notification.New(notification.Config{...})
member := membership.New(membership.Config{
    NotifDispatcher: notif.Dispatcher(),
    // ...
})
```

Cela permet :
- De tester `membership` sans `notification` (mock du Dispatcher).
- De swapper l'implémentation (ex: dispatcher fan-out, dispatcher noop pour CLI/migration).
- D'éviter les cycles d'import.

### 7. Documentation godoc

Chaque package expose :
- Un `doc.go` avec doc package globale (vue d'ensemble + pointeur vers la RFC).
- Une doc godoc sur `Module`, `Config`, `New`, `RegisterRoutes`, et chaque use case public.
- Des exemples runnables (`example_test.go`) pour les patterns d'usage typiques.

Visibles via `go doc github.com/AgileFactory/kernel/platform/auth` ou `pkg.go.dev` (si on rend le module public à terme).

### 8. Tests

- Tests unitaires dans le même package (pas de `_test` séparé).
- Mocks injectables via la `Config` (ex: `MockUserRepository` en `infrastructure_test.go`).
- Tests d'intégration séparés sous `tests/integration/` (utilisent une vraie DB de test).
- Tests cross-tenant obligatoires : `test_isolation.go` qui vérifie qu'un tenant ne voit pas les données d'un autre.

### 9. Versioning sémantique

- Git tag `vX.Y.Z` à la racine du repo `AgileFactory/kernel`.
- **Breaking change = bump majeur** (`v2.0.0`). Inclut :
  - Changement de signature d'une fonction exportée.
  - Suppression d'un champ `Config` exporté.
  - Renommage d'un endpoint HTTP exposé via `RegisterRoutes`.
  - Changement incompatible d'un schéma de migration déjà déployé.
- Mineur (`v0.X.0`) : ajouts compatibles (nouveau use case, nouveau champ Config optionnel, nouvel endpoint).
- Patch (`v0.0.X`) : bug fixes, perf, doc.

Chaque release est documentée dans `CHANGELOG.md` avec section explicite "## Breaking Changes" si présente.

### 10. Vérifications CI

Le pipeline CI du repo `AgileFactory/kernel` vérifie :

- `go vet ./...`
- `go test ./...` avec couverture minimum (>70%).
- `staticcheck ./...`
- **Pas d'import vers `cmd/`** depuis `platform/` ou `kernel/` (script de lint custom).
- **Migrations conformes** : `find platform/*/migrations -name '*.sql'` ne contient que des fichiers `<NNN>_<description>.{up,down}.sql` (NNN strictement numérique, requis par `iofs`).
- **Pas d'`init()` exporté** : grep sur les packages `platform/`.
- **`go doc` sans warning** sur chaque package.

---

## **Drawbacks**

1. **Verbosité initiale** : chaque package porte sa Config + Module + RegisterRoutes, là où un framework type wire/dig automatiserait. Justifié par la lisibilité (le consommateur voit exactement ce qui se passe dans son `main.go`) et par l'absence de magie compile-time.
2. **Migrations cross-package complexes** : si `membership` dépend de `auth.users`, l'ordre d'application des migrations doit être déterministe. Mitigation : `migrator.Run` traite les sources **dans l'ordre fourni par le consommateur** (slice `[]migrator.Source`). Les dépendances cross-package sont documentées dans la RFC du package consommateur, et le `main.go` du SaaS consommateur encode l'ordre.
3. **Friction pour les hot fixes** : un bug dans `platform/auth` nécessite un tag de release pour propager aux SaaS consommateurs. Mitigation : `replace` dans `go.mod` pour le dev local, branches de hotfix taguées rapidement.
4. **Pas adapté pour les packages très petits** (1 fichier). Mitigation : ne pas créer de package si le périmètre ne le justifie pas — préférer un sous-dossier dans un package plus large.

---

## **Alternatives**

### Alternative 1 — Framework de DI (wire, dig, fx)

Rejeté. Argumentation :
- Ajout d'une dépendance externe pour un problème simple (cf. ADR-0008).
- Magie compile-time (wire) ou runtime (dig, fx) → debug plus difficile.
- Le pattern Module manuel reste lisible et auditable.

### Alternative 2 — Un seul gros package par profil (B2B, B2B2C)

Rejeté. Argumentation :
- Couple les briques entre elles (auth + tenant + membership dans un seul package), empêchant la réutilisation sélective.
- Rend les tests plus lents (toute la chaîne s'instancie).
- Casse la séparation Clean Architecture.

### Alternative 3 — Migrations centralisées dans `kernel/migrations/`

Rejeté. Argumentation :
- Couple les packages (mise à jour de `auth/` nécessite éditer `kernel/migrations/`).
- Empêche le SaaS consommateur d'utiliser seulement un sous-ensemble de packages (devrait quand même charger toutes les migrations).
- Le pattern `embed.FS` par package est idiomatique en Go 1.16+.

### Alternative 4 — Application monolithique forkable (status quo)

Rejeté. Argumentation :
- Forker = perdre les mises à jour automatiques de `kernel`.
- Divergence inévitable entre les SaaS : chaque fork évolue indépendamment.
- Pas de baseline de sécurité partagée.

---

## **Plan d'adoption**

Cette RFC est **fondatrice** pour la restructuration packaging (P0-0 du plan de spécialisation). État d'adoption :

1. **P0-0a (DONE)** : `backend/internal/{kernel,platform,delivery}/` déplacés à la racine, module renommé `github.com/AgileFactory/kernel`. Commit `0ebe3d5`.
2. **P0-0b (DONE)** : refactor des 4 plateformes (`auth`, `tenant`, `membership`, `audit`) en pattern Module autonome. Interface `audit.EventLogger` introduite ; champ `audit` renommé `auditLogger` dans les consumers ; `bootstrap/container.go` supprimé au profit d'une composition explicite dans `examples/saas-b2b/main.go`.
3. **P0-0c (DONE)** : migrations déplacées sous `platform/<feature>/migrations/`, exposées via `embed.FS`, composées via `migrator.Run([]migrator.Source{...})` avec table de versions par-package (`schema_migrations_<name>`).
4. **P0-15 (TODO)** : mettre en place les vérifications CI (lint imports, format migrations, no-init).
5. **P0-14 (TODO)** : tag `v0.1.0` une fois P0-8 à P0-12 terminés (outbox + signup-owner + accept-invite + email verification + refresh tokens).
6. Toute nouvelle plateforme (sso, billing, end_customers, etc.) suit le contrat dès sa création.

### Écarts par rapport à la première rédaction de la RFC

- **Préfixe `<feature>_` dans les noms de fichiers migration** : initialement prévu, abandonné car incompatible avec le parser `iofs` de `golang-migrate` (exige un préfixe numérique strict). L'isolation par `embed.FS` par-package rend ce préfixe redondant.
- **Signature `migrator.Run(db, []embed.FS)`** : ajustée en `migrator.Run(databaseURL, []migrator.Source{Name, FS, Path})`. Le nom logique est nécessaire pour scope la table de versions (`x-migrations-table=schema_migrations_<name>`).
- **Split `<feature>.go` / `config.go`** : non-appliqué pour l'instant (les 4 plateformes existantes ont leur `Config` dans `<feature>.go`, fichiers de moins de 100 lignes). À splitter quand un Config dépasse 30 lignes ou nécessite `applyDefaults`/`validate`.

---

## **Références**

- [ADR-0002 — Clean Architecture Trois Niveaux](../adrs/0002-Clean-Architecture-Trois-Niveaux.md)
- [ADR-0008 — Injection Dépendances Manuelle](../adrs/0008-Injection-Dependances-Manuelle.md)
- [docs/architecture/overview.md](../architecture/overview.md)
- Go blog — [Embedding files](https://go.dev/blog/embed)
- Go blog — [Module compatibility promise](https://go.dev/blog/module-compatibility)
