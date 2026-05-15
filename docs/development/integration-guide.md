# Guide d'intégration — Consommer `kernel` depuis un repo SaaS

Ce guide t'amène d'un repo Go vide à une app SaaS B2B fonctionnelle qui consomme `github.com/AgileFactory/kernel`. Compte environ **30 minutes** la première fois.

> **Pré-requis** : Go 1.25+, Docker & Docker Compose, accès au repo privé `AgileFactory/kernel`.

---

## 1. Configurer l'accès au repo privé

`kernel` est privé. Go a besoin de pouvoir le résoudre via HTTPS authentifié.

```bash
# Marque toute l'org AgileFactory comme privée (évite que GOPROXY essaie de la résoudre publiquement)
export GOPRIVATE=github.com/AgileFactory/*

# Rewrite git pour utiliser un PAT au lieu d'auth interactive
git config --global \
    url."https://<TON_USER>:<TON_PAT>@github.com/AgileFactory/".insteadOf \
    "https://github.com/AgileFactory/"
```

Le PAT a besoin du scope `repo` (full read). Ajoute `GOPRIVATE` à ton `~/.bashrc` / `~/.zshrc` si tu travailles fréquemment avec.

---

## 2. Initialiser le module consommateur

```bash
mkdir my-saas && cd my-saas
go mod init github.com/mon-org/my-saas
go get github.com/AgileFactory/kernel@v0.1.0
```

---

## 3. Brancher Postgres en local

`kernel` ne fournit pas de `docker-compose.yml` à copier — chaque consommateur définit le sien. Voici le minimum :

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:
```

```bash
docker compose up -d
```

---

## 4. Le fichier `main.go` minimal

C'est la pièce centrale. Tu peux copier-coller depuis [`examples/saas-b2b/main.go`](../../examples/saas-b2b/main.go) du repo `kernel` et adapter.

```go
package main

import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    deliveryhttp "github.com/AgileFactory/kernel/delivery/http"
    "github.com/AgileFactory/kernel/kernel/config"
    "github.com/AgileFactory/kernel/kernel/events"
    "github.com/AgileFactory/kernel/kernel/logger"
    "github.com/AgileFactory/kernel/kernel/migrator"
    "github.com/AgileFactory/kernel/kernel/server"
    "github.com/AgileFactory/kernel/platform/audit"
    auditinfra "github.com/AgileFactory/kernel/platform/audit/infrastructure"
    "github.com/AgileFactory/kernel/platform/auth"
    authinfra "github.com/AgileFactory/kernel/platform/auth/infrastructure"
    "github.com/AgileFactory/kernel/platform/membership"
    memberinfra "github.com/AgileFactory/kernel/platform/membership/infrastructure"
    "github.com/AgileFactory/kernel/platform/tenant"
    tenantinfra "github.com/AgileFactory/kernel/platform/tenant/infrastructure"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    cfg := config.Load()

    log, _, _ := logger.New(logger.Config{
        Level: cfg.LogLevel, Environment: cfg.Environment, Dir: cfg.LogDir,
    })

    db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
    if err != nil { log.Error("db connect", "error", err); os.Exit(1) }
    defer db.Close()

    // (1) Migrations — events d'abord pour la table event_outbox
    if err := migrator.Run(cfg.DatabaseURL, []migrator.Source{
        {Name: "events",     FS: events.Migrations,     Path: "migrations"},
        {Name: "tenant",     FS: tenant.Migrations,     Path: "migrations"},
        {Name: "auth",       FS: auth.Migrations,       Path: "migrations"},
        {Name: "membership", FS: membership.Migrations, Path: "migrations"},
        {Name: "audit",      FS: audit.Migrations,      Path: "migrations"},
    }); err != nil { log.Error("migrate", "error", err); os.Exit(1) }

    // (2) Bus outbox
    bus, err := events.New(events.Config{Pool: db, Logger: log})
    if err != nil { log.Error("events", "error", err); os.Exit(1) }

    // (3) Repos partagés (UN seul instance par repo)
    auditRepo          := auditinfra.NewPostgresAuditRepository(db)
    tenantRepo         := tenantinfra.NewPostgresTenantRepository(db)
    tenantSettingsRepo := tenantinfra.NewPostgresTenantSettingsRepository(db)
    memberRepo         := memberinfra.NewPostgresMembershipRepository(db)
    userRepo           := authinfra.NewPostgresUserRepository(db)
    refreshTokenRepo   := authinfra.NewPostgresRefreshTokenRepository(db)
    emailVerifRepo     := authinfra.NewPostgresEmailVerificationTokenRepository(db)
    inviteTokenRepo    := memberinfra.NewPostgresInviteTokenRepository(db)

    // (4) Modules — composition explicite
    auditMod  := audit.New(audit.Config{Repo: auditRepo, MembershipRepo: memberRepo, Logger: log})
    tenantMod := tenant.New(tenant.Config{
        Pool: db, Repo: tenantRepo, SettingsRepo: tenantSettingsRepo,
        MembershipRepo: memberRepo, AuditLogger: auditMod.EventLogger(), Logger: log,
    })
    memberMod := membership.New(membership.Config{
        Pool: db, Repo: memberRepo, InviteTokenRepo: inviteTokenRepo,
        TenantRepo: tenantRepo, UserLookup: userRepo,
        AuditLogger: auditMod.EventLogger(), Publisher: bus.Publisher(), Logger: log,
    })
    authMod := auth.New(auth.Config{
        Pool: db, UserRepo: userRepo,
        TenantRepo: tenantRepo, TenantSettingsRepo: tenantSettingsRepo,
        MembershipRepo: memberRepo, InviteTokenRepo: inviteTokenRepo,
        RefreshTokenRepo: refreshTokenRepo, EmailVerifRepo: emailVerifRepo,
        MembershipListUC: memberMod.ListUC,
        AuditLogger: auditMod.EventLogger(), Publisher: bus.Publisher(),
        JWTSecret: cfg.JWTSecret, TokenLifespan: cfg.TokenLifespan, Logger: log,
    })

    // (5) HTTP routing + middlewares globaux + dispatcher
    mux := http.NewServeMux()
    auditMod.RegisterRoutes(mux)
    tenantMod.RegisterRoutes(mux)
    memberMod.RegisterRoutes(mux)
    authMod.RegisterRoutes(mux)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go bus.Run(ctx)

    srv := server.New(cfg, deliveryhttp.ApplyGlobalMiddlewares(mux, log))

    // Graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        if err := srv.Start(); err != nil && err != http.ErrServerClosed {
            log.Error("server", "error", err)
        }
    }()
    <-stop
    log.Info("shutting down")
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()
    _ = srv.Shutdown(shutdownCtx)
}
```

> **Pourquoi pas de framework DI ?** ADR-0008 : composition manuelle explicite. Plus de boilerplate, mais zéro magie au boot — ce que tu lis = ce qui s'exécute.

---

## 5. `.env` minimum

```bash
APP_ENV=development
DATABASE_URL=postgres://app:app@localhost:5432/app
JWT_SECRET=change-this-to-something-at-least-32-bytes-long
PORT=8080
LOG_LEVEL=debug
TOKEN_LIFESPAN=1h
```

⚠️ `JWT_SECRET` < 32 bytes en prod = refus de démarrer (`auth.New` validera). En dev, n'importe quoi convient.

---

## 6. Premier lancement

```bash
go run .
```

Tu devrais voir :

```
INFO migrate events: applied 2 migrations
INFO migrate tenant: applied 2 migrations
INFO migrate auth: applied 5 migrations
INFO migrate membership: applied 4 migrations
INFO migrate audit: applied 2 migrations
INFO outbox dispatcher starting poll_interval=5s batch_size=50 max_attempts=10
INFO server listening port=8080
```

Test rapide :

```bash
curl -X POST http://localhost:8080/api/v1/auth/signup-owner \
  -H "Content-Type: application/json" \
  -d '{
    "tenantName": "Acme",
    "tenantSlug": "acme",
    "email": "alice@acme.com",
    "password": "supersecretpassword"
  }'
```

Réponse 201 :

```json
{
  "tenantId": "...",
  "userId": "...",
  "membershipId": "...",
  "accessToken": "eyJ...",
  "refreshToken": "...",
  "emailVerificationPending": true,
  "user": { ... }
}
```

---

## 7. Ajouter ses propres modules métier

`kernel` est **socle**. Tu ajoutes tes modules business **à côté** des modules `platform/*`. Convention :

```
my-saas/
├── cmd/
│   └── api/main.go
├── internal/
│   └── domain/
│       └── order/                  # ton module business
│           ├── domain/
│           ├── usecase/
│           ├── infrastructure/
│           ├── delivery/http/
│           └── order.go            # ta Module + Config + New
├── go.mod
└── docker-compose.yml
```

Ton module suit RFC-0038 (mêmes contrats que les `platform/*`) :

```go
// internal/domain/order/order.go
package order

import (
    "embed"
    "net/http"

    "github.com/AgileFactory/kernel/platform/audit"
    "github.com/AgileFactory/kernel/kernel/events"
)

//go:embed migrations/*.sql
var Migrations embed.FS

type Config struct {
    Pool        *pgxpool.Pool
    Repo        domain.OrderRepository
    AuditLogger audit.EventLogger
    Publisher   events.Publisher
}

type Module struct { /* ... */ }
func New(cfg Config) *Module { /* ... */ }
func (m *Module) RegisterRoutes(mux *http.ServeMux) { /* ... */ }
```

Puis dans ton `main.go`, tu wires comme les modules kernel :

```go
orderMod := order.New(order.Config{Pool: db, ...})
orderMod.RegisterRoutes(mux)
// ajouter aux migrations
{Name: "order", FS: order.Migrations, Path: "migrations"},
```

---

## 8. Authentifier des routes business

Pour qu'une route soit derrière JWT + tenant-aware, utilise les middlewares :

```go
orderHandler := orderMod.Handler()
mux.Handle("POST /api/v1/orders",
    deliveryhttp.WithTenant(
        deliveryhttp.WithAuth(orderHandler, cfg.JWTSecret),
    ),
)
```

L'auth middleware extrait le user_id du JWT et le pousse en context via `kernel/auth`. Le tenant middleware extrait le `tenant_id` du header `X-Tenant-Id` (ou du JWT pour les flows tenant-scoped).

Côté handler :

```go
import kauth "github.com/AgileFactory/kernel/kernel/auth"

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    userID, _ := kauth.UserIDFromContext(r.Context())
    tenantID, _ := kauth.TenantIDFromContext(r.Context())
    // ...
}
```

---

## 9. Publier un événement depuis ton use case

```go
import (
    "github.com/AgileFactory/kernel/kernel/events"
    "github.com/AgileFactory/kernel/kernel/txctx"
)

func (uc *CreateOrder) Execute(ctx context.Context, params Params) error {
    return txctx.ExecInTx(ctx, uc.pool, func(ctx context.Context) error {
        if err := uc.repo.Save(ctx, order); err != nil { return err }
        return uc.publisher.Publish(ctx, events.Event{
            Type:    "order.created",
            Payload: map[string]any{"order_id": order.ID, "total": order.Total},
            Meta:    events.EventMeta{TenantID: order.TenantID},
        })
    })
}
```

Et côté wiring, subscribe à l'event où tu en as besoin :

```go
bus.Subscribe("order.created", func(ctx context.Context, ev events.Event) error {
    return inventoryMod.HandleOrderCreated(ctx, ev)
})
```

---

## 10. Tests

`kernel` fournit ses propres tests (`go test ./...`). Côté consommateur, on recommande :

- **Tests unitaires** sur tes use cases — mockent les repos via interfaces (les tiens, pas ceux du kernel).
- **Tests d'intégration** avec une vraie DB Postgres (testcontainers-go ou un `pg_dump` local).
- **Pas de mock de Postgres** — les query SQL kernel sont vérifiées en CI sur une vraie DB.

---

## 11. Mise en production

| Item | À faire |
|---|---|
| `JWT_SECRET` ≥ 32 bytes | Générer avec `openssl rand -base64 32`. Stocker en secret manager. |
| `BCRYPT_COST` = 12 | Plus lent au login mais 4× plus difficile à brute-forcer qu'un coût 10. |
| `TLS termination` | Devant l'app (Cloudflare, ALB, nginx). Kernel sert en HTTP plain. |
| `Logging JSON` | `APP_ENV=production` bascule le formatter slog en JSON. |
| `Migrations zero-downtime` | Toujours `up.sql` rétro-compatible avec la version N-1. Voir [`docs/operations/runbook.md`](../operations/runbook.md). |
| `Outbox monitoring` | Alert sur `SELECT count(*) FROM event_outbox WHERE published_at IS NULL AND attempts >= max_attempts`. |
| `Backup DB` | RPO 1h recommandé. La table `event_outbox` est critique : un crash après commit business sans backup = perte d'événement. |

---

## Aller plus loin

- [`docs/architecture/overview.md`](../architecture/overview.md) — vue d'ensemble.
- [`docs/architecture/technical-deepdive.md`](../architecture/technical-deepdive.md) — invariants critiques.
- [`docs/architecture/diagrams/`](../architecture/diagrams/) — séquences & ERD.
- [`docs/api/openapi.yaml`](../api/openapi.yaml) — référence API HTTP.
- [`docs/operations/runbook.md`](../operations/runbook.md) — exploitation prod.
