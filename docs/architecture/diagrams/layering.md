# Layering & dépendances inter-modules

Vue d'ensemble des packages livrés (v0.1.0) et de leurs dépendances. Aligné avec ADR-0002 (Clean Architecture 3 niveaux) et RFC-0038 (conventions de package).

```mermaid
flowchart TB
    subgraph consumer["Consumer SaaS repo"]
        Main[main.go]
    end

    subgraph kernel_layer["kernel/* — infra transverse"]
        Config[config]
        Logger[logger]
        Migrator[migrator]
        Errors[errors]
        Txctx[txctx]
        Events[events<br/>outbox]
        Server[server]
        Resp[response]
        Valid[validator]
        KAuth[auth<br/>ctx helpers]
    end

    subgraph delivery_layer["delivery/http — middlewares"]
        Recovery[Recovery]
        LogMW[Logger MW]
        AuthMW[Auth MW]
        TenantMW[Tenant MW]
    end

    subgraph platform_layer["platform/* — briques SaaS"]
        Audit[[audit]]
        Tenant[[tenant]]
        Auth[[auth]]
        Member[[membership]]
        Notif[[notification]]
    end

    subgraph external_layer["Dépendances externes"]
        PG[(PostgreSQL)]
        Mail((Email Sender))
    end

    Main --> Config
    Main --> Logger
    Main --> Migrator
    Main --> Events
    Main --> Server
    Main --> Audit
    Main --> Tenant
    Main --> Auth
    Main --> Member
    Main --> Notif

    Tenant --> Audit
    Auth --> Audit
    Auth --> Tenant
    Auth --> Member
    Member --> Audit
    Member --> Tenant

    Auth -.-> Events
    Member -.-> Events
    Notif -.-> Events

    Events --> Txctx
    Txctx --> PG

    Audit --> Errors
    Tenant --> Errors
    Auth --> Errors
    Member --> Errors
    Notif --> Errors

    Auth --> KAuth
    AuthMW --> KAuth
    TenantMW --> KAuth

    Main --> Recovery
    Recovery --> LogMW

    Notif --> Mail
    Audit --> PG
    Tenant --> PG
    Auth --> PG
    Member --> PG
    Notif --> PG

    classDef kernel fill:#e3f2fd,stroke:#1976d2,color:#000
    classDef platform fill:#e8f5e9,stroke:#388e3c,color:#000
    classDef consumer fill:#fff3e0,stroke:#f57c00,color:#000
    classDef external fill:#fce4ec,stroke:#c2185b,color:#000

    class Config,Logger,Migrator,Errors,Txctx,Events,Server,Resp,Valid,KAuth kernel
    class Recovery,LogMW,AuthMW,TenantMW kernel
    class Audit,Tenant,Auth,Member,Notif platform
    class Main consumer
    class PG,Mail external
```

## Lecture

- **`kernel/*`** ne dépend que de la stdlib + pgx/pgxpool + slog. **Aucune** dépendance vers `platform/*` (enforced par `scripts/lint.sh`).
- **`platform/*`** dépend de `kernel/*` et **peut** dépendre d'autres `platform/*` (ex. `auth → membership → tenant → audit`). Le cycle audit↔membership est **cassé** en construisant les repos hors des modules (cf. `examples/saas-b2b/main.go` étape 3).
- **`delivery/http`** héberge les middlewares partagés (Recovery, Logger MW, Auth MW, Tenant MW). Les routes sont déclarées **par module** via `RegisterRoutes(mux)`.
- Flèches **pleines** = appel synchrone. Flèches **pointillées** = publication d'événement outbox (asynchrone, at-least-once).

## Composition côté consommateur

L'ordre canonique du wiring dans `main.go` :

1. `config.Load()` → variables d'env.
2. `logger.New()` → fan-out slog.
3. `pgxpool.New()` → pool DB partagé.
4. `migrator.Run([events, tenant, auth, membership, audit, notification], ...)`.
5. `events.New()` → bus outbox.
6. Construction des repos (un seul instance par repo, partagé entre modules).
7. Construction des modules (`audit.New`, `tenant.New`, `membership.New`, `auth.New`, `notification.New`).
8. `notifMod.Subscribe(bus)` **avant** `bus.Run(ctx)`.
9. `module.RegisterRoutes(mux)` pour chaque module.
10. `server.New(cfg, deliveryhttp.ApplyGlobalMiddlewares(mux, log)).Start()`.

Le canonique complet vit dans [`examples/saas-b2b/main.go`](../../../examples/saas-b2b/main.go).
