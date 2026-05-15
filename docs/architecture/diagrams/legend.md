# Légende des diagrammes

Convention visuelle partagée par tous les diagrammes mermaid sous `docs/architecture/diagrams/`. Cible la lecture sur GitHub (rendu mermaid natif) **et** le wiki (idem).

---

## Couches

| Symbole | Sens |
|---|---|
| `[[Module]]` | Frontière de package Go importable (ex. `platform/auth`). |
| `[Component]` | Sous-composant d'un module (use case, repo, handler). |
| `((External))` | Système hors-kernel (DB, Stripe, mailer, navigateur). |
| `>Trigger]` | Événement / signal externe (HTTP request, cron tick, outbox poll). |

## Couleurs (mermaid `classDef`)

| Classe | Usage |
|---|---|
| `kernel` | Packages de `kernel/*` (infra transverse). |
| `platform` | Packages de `platform/*` (briques métier réutilisables). |
| `consumer` | Code côté consommateur (main.go, modules business). |
| `external` | Dépendances externes (DB, mailer, IdP, …). |
| `deprecated` | Composants en voie de retrait (ex. ADR-0009 audit best-effort → outbox at-least-once). |

## Flèches

| Notation | Sens |
|---|---|
| `A --> B` | Appel synchrone (B doit répondre pour que A continue). |
| `A -.-> B` | Appel asynchrone (fire-and-forget, queue, événement outbox). |
| `A ==> B` | Transaction étendue (A et B partagent la même tx via `kernel/txctx`). |
| `A --x B` | Appel refusé (auth / RBAC / quota). |

## Mermaid types utilisés

- **Sequence diagrams** pour les flux temporels (signup, invite, refresh).
- **Flowchart** pour les compositions de modules (layering).
- **erDiagram** pour le schéma PostgreSQL.

## Diagrammes disponibles

- [`layering.md`](layering.md) — composition des modules + dépendances.
- [`signup-flow.md`](signup-flow.md) — séquence `POST /api/v1/auth/signup-owner`.
- [`invite-flow.md`](invite-flow.md) — séquence `invite + accept-invite`.
- [`outbox-flow.md`](outbox-flow.md) — publish → dispatch → handler avec retries.
- [`erd.md`](erd.md) — schéma relationnel PostgreSQL (toutes les tables livrées).
