# _RFC #0015:_ Event Bus Interne — Postgres LISTEN/NOTIFY vs NATS

**date:** `2026 May 01`

---

## **Status:** `PROPOSED` (amendé v1.1 — Outbox Pattern, 2026 May 04)

---

## **Summary**

Ce RFC tranche la question ouverte du RFC-0002 sur le **bus d'événements interne** du socle Agile Factory, nécessaire pour la communication découplée entre plateformes (`audit`, `notification`, `billing`) sans couplage direct entre use cases.

La décision retenue est : **Postgres LISTEN/NOTIFY pour la v1** (zéro infrastructure supplémentaire), avec un port abstrait permettant la migration vers NATS ou autre en v2.

---

## **Motivation**

Plusieurs plateformes ont besoin de réagir aux événements des autres sans couplage direct :

```
MemberInvited → [notification] envoyer email d'invitation
MemberInvited → [audit] logger l'événement
UserRegistered → [notification] envoyer email de bienvenue
SubscriptionExpired → [notification] alerter l'admin
SubscriptionExpired → [auth] restreindre l'accès
```

Sans bus d'événements, ces réactions sont injectées directement dans les use cases (couplage fort). Avec un bus, chaque plateforme publie un événement et les consommateurs réagissent indépendamment.

Le couplage actuel (injection directe de `LogEventUseCase` dans tous les use cases) est un **anti-pattern temporaire** acceptable pour la v1, destiné à être remplacé par un bus d'événements.

---

## **Proposal**

### 1. Port abstrait (kernel)

```go
// kernel/events/bus.go
type Event struct {
    Type    string         // "member.invited", "user.registered"
    Payload map[string]any
    Meta    EventMeta
}

type EventMeta struct {
    TenantID  string
    ActorID   string
    OccurredAt time.Time
    TraceID   string
}

type Publisher interface {
    Publish(ctx context.Context, event Event) error
}

type Subscriber interface {
    Subscribe(eventType string, handler EventHandler) error
}

type EventHandler func(ctx context.Context, event Event) error
```

### 2. Implémentation v1 : Postgres LISTEN/NOTIFY

```go
// kernel/events/postgres_bus.go
func (b *PostgresBus) Publish(ctx context.Context, event Event) error {
    payload, _ := json.Marshal(event)
    _, err := b.db.Exec(ctx,
        `SELECT pg_notify($1, $2)`,
        event.Type, string(payload),
    )
    return err
}
```

Les `LISTEN` sont enregistrés dans des goroutines dédiées au démarrage de l'application.

**Avantages pour la v1 :**
- Zéro infrastructure supplémentaire (Postgres est déjà présent)
- Transactions atomiques possible (`pg_notify` peut être émis dans une transaction)
- Délivrabilité `at-most-once` (acceptable pour la v1)

### 3. Migration vers NATS (v2)

Quand les limites de Postgres LISTEN/NOTIFY sont atteintes (>1000 canaux, >100 messages/s, clustering multi-nœuds) :

```go
// kernel/events/nats_bus.go
type NATSBus struct { conn *nats.Conn }

func (b *NATSBus) Publish(ctx context.Context, event Event) error {
    payload, _ := json.Marshal(event)
    return b.conn.Publish(event.Type, payload)
}
```

Le changement d'implémentation ne modifie **aucun code métier** (seul le `container.go` change).

### 4. Refactoring de l'audit avec le bus

Avec le bus, `LogEventUseCase` n'est plus injecté dans chaque use case :

```go
// Avant (couplage direct)
uc.audit.Execute(ctx, auditusecase.LogEventParams{...})

// Après (via bus)
uc.bus.Publish(ctx, events.Event{Type: "member.invited", Payload: {...}})
```

`platform/audit` s'abonne à tous les événements et les persiste.

---

## **Drawbacks**

### Postgres LISTEN/NOTIFY : at-most-once
Si le consommateur est déconnecté au moment de la notification, l'événement est perdu. Mitigation : outbox pattern ou persistance des événements en base.

### Pas de replay des événements perdus
Contrairement à Kafka, Postgres LISTEN/NOTIFY ne stocke pas d'historique. Un événement perdu est définitivement perdu.

### Complexité de la gestion des goroutines
Les abonnés LISTEN tournent dans des goroutines longues durée. Leur gestion (reconnexion, shutdown gracieux) nécessite du code soigné.

---

## **Alternatives**

### Injection directe entre use cases (status quo)
- ✅ Simple, pas d'infrastructure
- ❌ Couplage fort, violation du principe de séparation des préoccupations
- **Verdict : anti-pattern temporaire, à remplacer**

### NATS (dès la v1)
- ✅ At-least-once, JetStream pour la persistance
- ✅ Clustering natif
- ❌ Infrastructure supplémentaire à opérer
- ❌ Complexité prématurée pour la v1
- **Verdict : v2**

### Kafka
- ✅ At-least-once, replay complet, log append
- ❌ Infrastructure lourde (ZooKeeper ou KRaft)
- ❌ Latence plus élevée que NATS
- **Verdict : pour les SaaS à très haute volumétrie (>100k events/s)**

### Redis Pub/Sub
- ✅ Simple, déjà utilisé si cache Redis présent
- ❌ At-most-once comme Postgres
- ❌ Dépendance Redis supplémentaire si pas déjà présent
- **Verdict : possible mais Postgres préféré (zéro infra supplémentaire)**

---

## **Amendment v1.1 — Outbox Pattern (2026 May 04)**

### Pourquoi un outbox

La garantie `at-most-once` initiale est inacceptable pour les consommateurs critiques :

- **RFC-0005 (Audit)** : un événement perdu = drift permanent du journal d'audit, problème de conformité.
- **RFC-0011 (Notification)** : un email d'invitation perdu = utilisateur bloqué.
- **Futur RFC Webhooks BaaS** : un webhook perdu = perte de confiance des tenants externes (un BaaS qui rate des events n'est pas un BaaS sérieux).
- **RFC-0017 (Object Storage)** : AV scanning et GC asynchrones s'appuient sur des events fiables.

Un consommateur déconnecté au moment du `pg_notify` perd définitivement l'événement. L'outbox transactionnel résout ce gap **sans changer l'infrastructure** (Postgres seul) en garantissant **at-least-once** côté publication.

### Modèle outbox transactionnel

```sql
CREATE TABLE event_outbox (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    VARCHAR(100) NOT NULL,
    payload       JSONB NOT NULL,
    meta          JSONB NOT NULL,            -- TenantID, ActorID, OccurredAt, TraceID
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at  TIMESTAMPTZ,
    attempts      INT NOT NULL DEFAULT 0,
    last_error    TEXT
);

CREATE INDEX idx_outbox_unpublished
    ON event_outbox (created_at)
    WHERE published_at IS NULL;

CREATE INDEX idx_outbox_retention
    ON event_outbox (created_at)
    WHERE published_at IS NOT NULL;
```

### Flux de publication

```go
// kernel/events/outbox_publisher.go
func (p *OutboxPublisher) Publish(ctx context.Context, event Event) error {
    // Insertion DANS la transaction métier en cours (récupérée via context)
    tx := txctx.MustFromContext(ctx)
    return tx.Exec(ctx, `
        INSERT INTO event_outbox (event_type, payload, meta)
        VALUES ($1, $2, $3)
    `, event.Type, event.Payload, event.Meta)
}
```

L'invariant clé : **l'event est inséré dans la même transaction que la mutation métier**. Si la transaction commit, l'event est garanti d'être publié (tôt ou tard). Si elle rollback, l'event disparaît avec elle. Aucun cas de divergence.

### Dispatcher

```go
// kernel/events/outbox_dispatcher.go
func (d *OutboxDispatcher) Run(ctx context.Context) {
    ticker := time.NewTicker(d.pollInterval) // défaut 1s
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C:
            d.dispatchBatch(ctx) // SELECT FOR UPDATE SKIP LOCKED LIMIT 100
        }
    }
}
```

Pour chaque event non publié :
1. `pg_notify(event_type, payload)` (best-effort, listeners actifs reçoivent en temps réel).
2. `UPDATE event_outbox SET published_at = NOW() WHERE id = $1`.
3. En cas d'erreur : `attempts++`, `last_error = ...`, retry avec **backoff exponentiel** (1s, 2s, 4s, 8s, ..., max 5 min).
4. Après `MAX_ATTEMPTS = 10` échecs, l'event passe en quarantaine (alert + rétention étendue).

Le dispatcher est **single-writer** (un seul leader élu via Postgres advisory lock `pg_try_advisory_lock`) pour garantir l'ordre par `created_at` et éviter les doublons.

### Garanties

| Propriété | Garantie |
|---|---|
| Publication | **At-least-once** (dès que la transaction métier commit) |
| Ordre | **FIFO par event_type** (single-writer + index sur `created_at`) ; cross-type non garanti |
| Réception côté consommateur | At-most-once via `pg_notify` ; **idempotency obligatoire** côté handler |
| Persistance | 30 jours (configurable via `EVENT_OUTBOX_RETENTION_DAYS`) |
| Replay | Endpoint admin (voir ci-dessous) |

### Idempotency requirement (consommateurs)

Chaque consommateur critique **doit** maintenir une table de déduplication :

```sql
CREATE TABLE processed_events (
    consumer    VARCHAR(100) NOT NULL,
    event_id    UUID NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (consumer, event_id)
);
```

Avant de traiter un event, le handler tente un `INSERT` dans `processed_events`. Si conflit (`event_id` déjà vu), il skip. Cette contrainte est **non négociable** pour les consommateurs branchés sur l'outbox — sans idempotency, le retry après crash duplique les effets de bord (emails, audits, webhooks).

### Replay

Endpoint administrateur (réservé super-admin RFC-0007) :

```
POST   /admin/events/{id}/replay        → Re-publie un event spécifique
POST   /admin/events/replay             → Re-publie un range (event_type + période)
GET    /admin/events?status=quarantine  → Liste les events en quarantaine
```

Le replay **réinsère** un event avec un nouveau `id` (l'ancien est conservé pour l'audit), évitant tout conflit avec l'idempotency.

### Réponses aux Open Questions originales

| Question | Réponse |
|---|---|
| Q1 — Persister tous les events ou juste l'audit ? | **Tous via outbox**, rétention 30 jours par défaut, GC périodique. L'audit reste une vue dérivée filtrée. |
| Q3 — Garantir l'ordre inter-plateforme ? | **FIFO par event_type** garanti (single-writer dispatcher) ; ordre cross-type non garanti — acceptable car les consommateurs réagissent par type. |

Q2 (limite Postgres LISTEN/NOTIFY avant migration NATS) reste ouverte — l'outbox permet de différer cette migration plus longtemps puisque la persistance compense les limites de débit de `pg_notify`.

### Migration depuis l'état actuel

L'injection directe de `LogEventUseCase` (anti-pattern documenté §Motivation) reste comme **fallback synchrone** pendant la transition. Migration progressive, un module à la fois :

1. **Phase 1** — Audit (RFC-0005) : devient consommateur outbox. Couplage direct supprimé.
2. **Phase 2** — Notification (RFC-0011) : passe en async fiable. Résout le « v1 best-effort synchrone ».
3. **Phase 3** — Tous les autres consommateurs futurs (Webhooks BaaS, AV scanning RFC-0017, GC).

Pendant la transition, un test d'intégration vérifie que chaque event publié par un use case migré est bien consommé par l'audit (équivalence de comportement avec le couplage direct).

### Impact sur les RFC dépendants

- **RFC-0005 (Audit)** : devient consommateur outbox — élimine l'injection de `LogEventUseCase` dans tous les use cases.
- **RFC-0011 (Notification)** : passe en async fiable, résout la déduplication v1.
- **RFC-0017 (Object Storage)** : AV scanning et GC s'appuient sur des events fiables.
- **Futur RFC Webhooks BaaS** : bénéficie nativement de la garantie at-least-once.

---

## **Open Questions**

1. ~~Doit-on persister tous les événements publiés (event store) ou seulement les audit logs ?~~ → **Résolu par l'outbox v1.1**.
2. Quelle est la limite pratique de Postgres LISTEN/NOTIFY avant de migrer vers NATS ?
3. ~~Comment garantir l'ordre des événements inter-plateforme ?~~ → **Résolu : FIFO par event_type via outbox v1.1**.
4. Quelle stratégie de partitionnement de l'outbox quand le volume dépasse ~10M lignes/mois (partitionnement temporel par `created_at` ?) ?
5. L'élection du leader du dispatcher via advisory lock suffit-elle pour un déploiement multi-AZ, ou faut-il un coordinator externe (etcd / Consul) ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique (question ouverte #2)
- RFC-0005 : Platform Audit (principal consommateur)
- RFC-0011 : Platform Notification (consommateur secondaire)
- RFC-0017 : Platform Object Storage (consommateur pour AV scanning et GC asynchrones)
- [PostgreSQL LISTEN/NOTIFY](https://www.postgresql.org/docs/current/sql-listen.html)
- [Transactional Outbox Pattern (microservices.io)](https://microservices.io/patterns/data/transactional-outbox.html)
- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
