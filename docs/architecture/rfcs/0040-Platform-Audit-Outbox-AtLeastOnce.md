# RFC #0040 — Audit Platform: Migration Best-Effort → Outbox At-Least-Once

**date:** `2026-06-29`  
**auteur:** Cheikh (Agile Factory)  
**destinataires:** Aziz, Lucas  
**dépend de:** RFC-0015 (kernel/events outbox bus — IMPLEMENTED)  
**remplace (partiellement):** ADR-0009 best-effort section (DEPRECATED)

---

## **Status:** `ACCEPTED`

**Amendments incorporated 2026-06-29 (Aziz + Lucas review):**
1. Publish synchronously (`txctx.ExecInTx` — no goroutine). The outbox INSERT is a fast local operation; the goroutine added untracked-goroutine risk with no benefit.
2. `Config.DB` and `Config.Publisher` are **mandatory**. `audit.New` returns `(*Module, error)` like `user.New` / `notification.New`.
3. `outbox_event_id UUID UNIQUE` added to `audit_logs`. Consumer stores `ev.ID` (the outbox row UUID) and uses `ON CONFLICT (outbox_event_id) DO NOTHING` for idempotency.
4. Typed `auditLogPayload` struct with `json.Marshal/Unmarshal` instead of `stringVal`/`metadataVal` helpers.
5. `kernel/events` addendum: `Event.ID` field populated by the dispatcher (outbox row UUID) so handlers can use it for deduplication.

---

## Contexte

ADR-0009 (mis à jour 2026-05-12) a déprécié la stratégie fire-and-forget pour l'audit et préconise une migration vers l'outbox at-least-once de RFC-0015. Cette RFC définit l'architecture concrète de cette migration.

### Problème actuel

```go
// LogEventUseCase.Execute — goroutine fire-and-forget
go func() {
    defer cancel()
    if err := uc.repo.Save(bgCtx, log); err != nil {
        slog.Error("audit: best-effort save failed", ...)
    }
}()
```

**Défauts :**
1. **Perte silencieuse** — si le goroutine échoue (pool timeout, crash pod), l'audit log est perdu sans aucune garantie de retry.
2. **Non-transactionnel** — l'opération métier commit, mais l'audit peut ne jamais être persisté.
3. **Goroutine non trackée** — sous charge, des centaines de goroutines d'audit peuvent s'accumuler sans backpressure.

---

## Décision

### Architecture

```
Caller (auth, membership, tenant, user, ...)
    │
    ▼
audit.EventLogger.Execute(ctx, params)   ← interface inchangée
    │  [synchronous txctx.ExecInTx — INSERT into event_outbox]
    ▼
event_outbox table  ─────────────────────────────────────────────┐
                                                                   │
                          kernel/events OutboxDispatcher (1s poll) │
                                                                   │
                                               AuditConsumer.Handle(ctx, ev)
                                                   │  [ON CONFLICT outbox_event_id DO NOTHING]
                                                   ▼
                                             audit_logs table
```

### Détails d'implémentation

**1. `kernel/events.Event` — champ `ID` ajouté**

Le dispatcher peuple `ev.ID` avec l'UUID de la ligne `event_outbox` avant d'invoquer les handlers. Les handlers peuvent ainsi utiliser cet ID comme clé de déduplication.

```go
type Event struct {
    ID      string         `json:"id,omitempty"` // outbox row UUID, set by dispatcher
    Type    string         `json:"type"`
    Payload map[string]any `json:"payload"`
    Meta    EventMeta      `json:"meta"`
}
```

**2. `LogEventUseCase` — publish synchrone via `txctx.ExecInTx`**

```go
// auditLogPayload is the typed contract stored in the outbox.
type auditLogPayload struct {
    TenantID   string         `json:"tenant_id"`
    ActorID    string         `json:"actor_id"`
    Action     string         `json:"action"`
    TargetType string         `json:"target_type"`
    TargetID   string         `json:"target_id"`
    Metadata   map[string]any `json:"metadata"`
    OccurredAt time.Time      `json:"occurred_at"`
}

type LogEventUseCase struct {
    db        *pgxpool.Pool
    publisher events.Publisher
}

func (uc *LogEventUseCase) Execute(ctx context.Context, params LogEventParams) {
    payload := auditLogPayload{
        TenantID:   params.TenantID,
        ActorID:    params.ActorID,
        Action:     string(params.Action),
        TargetType: params.TargetType,
        TargetID:   params.TargetID,
        Metadata:   params.Metadata,
        OccurredAt: time.Now().UTC(),
    }
    err := txctx.ExecInTx(ctx, uc.db, func(ctx context.Context) error {
        m, err := toPayloadMap(payload)
        if err != nil { return err }
        return uc.publisher.Publish(ctx, events.Event{
            Type:    "audit.log_created",
            Payload: m,
            Meta:    events.EventMeta{TenantID: params.TenantID, OccurredAt: payload.OccurredAt},
        })
    })
    if err != nil {
        slog.Error("audit: outbox publish failed", "error", err, "action", params.Action)
    }
}
```

**Propriétés :**
- Interface `EventLogger.Execute(ctx, params)` **inchangée** — zéro breaking change pour les appelants.
- Le write va dans `event_outbox` (at-least-once via le dispatcher RFC-0015).
- Si le publish échoue, l'erreur est loggée — audit ne bloque jamais le business.
- La goroutine est supprimée — le publish synchrone est une INSERT locale (~1-5ms).

**3. `AuditConsumer` — handler registré sur le bus**

```go
// platform/audit/usecase/consume_audit_log.go
type ConsumeAuditLogUseCase struct {
    repo domain.AuditRepository
}

func (uc *ConsumeAuditLogUseCase) Handle(ctx context.Context, ev events.Event) error {
    var p auditLogPayload
    b, _ := json.Marshal(ev.Payload)
    json.Unmarshal(b, &p)

    log := &domain.AuditLog{
        ID:            uuid.NewString(),
        TenantID:      p.TenantID,
        ActorID:       p.ActorID,
        Action:        domain.AuditAction(p.Action),
        TargetType:    p.TargetType,
        TargetID:      p.TargetID,
        Metadata:      p.Metadata,
        OccurredAt:    p.OccurredAt,
        OutboxEventID: ev.ID, // deduplication key
    }
    return uc.repo.Save(ctx, log)
}
```

INSERT uses `ON CONFLICT (outbox_event_id) DO NOTHING` — idempotent under at-least-once redelivery.

**4. `audit.Config` — `DB` et `Publisher` obligatoires**

```go
type Config struct {
    DB             *pgxpool.Pool              // required
    Repo           domain.AuditRepository     // required
    MembershipRepo memberdomain.MembershipRepository // required
    Publisher      events.Publisher           // required
    Logger         *slog.Logger               // required
}
```

`audit.New` retourne `(*Module, error)` et valide tous les champs.

**5. `audit.Module.Subscribe`**

```go
func (m *Module) Subscribe(sub events.Subscriber) {
    sub.Subscribe("audit.log_created", m.consumeUC.Handle)
}
```

Appelé dans `main.go` avant `bus.Run`.

**6. Migration `0003_add_audit_logs_outbox_event_id`**

```sql
-- up
ALTER TABLE audit_logs
    ADD COLUMN outbox_event_id UUID UNIQUE;

-- down
ALTER TABLE audit_logs
    DROP COLUMN outbox_event_id;
```

---

## Alternatives rejetées

### A. Retry synchrone (3 tentatives, backoff exponentiel)
**Rejeté** — l'audit bloque le retour HTTP si les 3 tentatives s'écoulent.

### B. Modifier `EventLogger.Execute` pour retourner une erreur
**Rejeté** — breaking change pour tous les appelants (auth, membership, tenant, user, white_label, customer_auth, customer_notifications).

### C. Transactional outbox strict (publish dans la même tx que l'opération métier)
**Rejeté pour cette version** — nécessite que les appelants passent leur tx dans le ctx, ce qui impose un couplage fort. Déféré à P2 quand les appelants seront refactorisés.

### D. Table dédiée `audit_outbox` (séparée de `event_outbox`)
**Rejeté** — introduit une deuxième infrastructure d'outbox à maintenir.

### E. Garder la goroutine (proposition initiale)
**Rejeté (Aziz + Lucas)** — la goroutine ne fait que réintroduire le problème des goroutines non trackées. L'INSERT dans `event_outbox` est une opération locale rapide (~1-5ms) — il n'y a aucun bénéfice à la décharger.

### F. `Config.DB`/`Publisher` optionnels avec fallback direct
**Rejeté (Aziz + Lucas)** — les fallbacks silencieux cachent des erreurs de configuration. Validation obligatoire comme `user.New` / `notification.New`.

---

## Conséquences

- **Gaining :** At-least-once delivery — un audit log peut être rejoué si le handler échoue.
- **Gaining :** Pas de breaking change — `EventLogger.Execute` garde la même signature.
- **Gaining :** Backpressure naturelle — le dispatcher RFC-0015 gère les retries avec backoff.
- **Gaining :** Idempotency — `ON CONFLICT (outbox_event_id) DO NOTHING` protège contre les doublons.
- **Gaining :** Suppression des goroutines non trackées.
- **Caveat :** Délai de persistance ~1s (poll interval du dispatcher). Les logs n'apparaissent pas instantanément — acceptable pour de l'audit.
- **Caveat :** La fenêtre "opération métier commit, publish échoue" existe toujours (les deux writes sont dans des tx séparées). Adressé en P2 par le transactional outbox strict (option C).
