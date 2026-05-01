# _RFC #0015:_ Event Bus Interne — Postgres LISTEN/NOTIFY vs NATS

**date:** `2026 May 01`

---

## **Status:** `PROPOSED`

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

## **Open Questions**

1. Doit-on persister tous les événements publiés (event store) ou seulement les audit logs ?
2. Quelle est la limite pratique de Postgres LISTEN/NOTIFY avant de migrer vers NATS ?
3. Comment garantir l'ordre des événements inter-plateforme ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique (question ouverte #2)
- RFC-0005 : Platform Audit (principal consommateur)
- RFC-0011 : Platform Notification (consommateur secondaire)
- [PostgreSQL LISTEN/NOTIFY](https://www.postgresql.org/docs/current/sql-listen.html)
- [NATS JetStream Documentation](https://docs.nats.io/nats-concepts/jetstream)
