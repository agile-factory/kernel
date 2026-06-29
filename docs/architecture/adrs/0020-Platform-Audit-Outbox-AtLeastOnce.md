# ADR-0020 — Audit Platform: Outbox At-Least-Once Delivery

**date:** `2026-06-29`  
**auteur:** Cheikh (Agile Factory)  
**status:** ACCEPTED  
**RFC:** [RFC-0040](../rfcs/0040-Platform-Audit-Outbox-AtLeastOnce.md)  
**remplace:** ADR-0009 §best-effort (DEPRECATED)

---

## Contexte

L'audit platform utilisait une stratégie fire-and-forget (goroutine + `repo.Save` direct) décrite dans ADR-0009. Cette stratégie exposait trois défauts documentés : perte silencieuse, non-transactionnalité, et accumulation de goroutines non trackées sous charge. ADR-0009 avait déjà déprécié cette section et préconisé une migration vers l'outbox at-least-once.

## Décision

Migration complète vers le bus outbox `kernel/events` (RFC-0015). Validé par Aziz et Lucas via RFC-0040 avec 5 amendements.

## Architecture

```
Caller.Execute(ctx, params)
    │ txctx.ExecInTx — INSERT INTO event_outbox
    ▼
event_outbox table
    │ OutboxDispatcher (1s poll, pg_try_advisory_lock)
    ▼
ConsumeAuditLogUseCase.Handle(ctx, ev)
    │ INSERT INTO audit_logs ... ON CONFLICT (outbox_event_id) DO NOTHING
    ▼
audit_logs table
```

## Patterns établis

### 1. Publish synchrone (pas de goroutine)

`LogEventUseCase.Execute` appelle `txctx.ExecInTx` directement. L'INSERT dans `event_outbox` est une opération locale (~1-5ms) — la goroutine n'apportait aucun bénéfice et réintroduisait le problème des goroutines non trackées.

### 2. `audit.New` retourne `(*Module, error)`

Cohérent avec `user.New` et `notification.New`. Tous les champs de `Config` sont obligatoires : `DB`, `Repo`, `MembershipRepo`, `Publisher`, `Logger`.

### 3. Typed payload struct

`auditLogPayload` est partagé entre `LogEventUseCase` (publisher) et `ConsumeAuditLogUseCase` (consumer) via JSON round-trip (`json.Marshal` → `map[string]any` → publisher, puis `json.Marshal` + `json.Unmarshal` → typed struct côté consumer). Cela garantit l'accord sur les noms et types de champs à la compilation.

### 4. Déduplication via `outbox_event_id`

- Migration `0003` ajoute `outbox_event_id UUID UNIQUE` à `audit_logs`.
- Le dispatcher peuple `ev.ID` (outbox row UUID) avant d'invoquer les handlers (addendum `kernel/events`).
- Le consumer stocke `ev.ID` dans `OutboxEventID` et l'INSERT utilise `ON CONFLICT (outbox_event_id) DO NOTHING`.
- NULL ne déclenche pas le UNIQUE constraint → les lignes pré-outbox ne sont pas affectées.

### 5. `audit.Module.Subscribe(sub events.Subscriber)`

Registre le consumer sur le bus. Appelé dans `main.go` avant `bus.Run`, cohérent avec `notifMod.Subscribe(bus)` et `userMod.Subscribe(bus)`.

### 6. Interface `EventLogger` inchangée

`Execute(ctx context.Context, params LogEventParams)` — zéro breaking change pour les appelants (auth, membership, tenant, user, white_label, customer_auth, customer_notifications).

## Conséquences

**Gains :**
- At-least-once delivery avec retry backoff exponentiel (dispatcher RFC-0015)
- Idempotency via `ON CONFLICT (outbox_event_id) DO NOTHING`
- Suppression des goroutines non trackées
- Erreurs de configuration détectées au démarrage (`New` retourne une erreur)

**Compromis :**
- Délai de persistance ~1s (poll interval du dispatcher) — acceptable pour de l'audit
- Fenêtre de perte "opération métier commit / publish échoue" reste possible (deux tx séparées). Adressé en P2 par transactional outbox strict.

## Wiring `main.go`

```go
auditMod, err := audit.New(audit.Config{
    DB:             db,
    Repo:           auditRepo,
    MembershipRepo: memberRepo,
    Publisher:      bus.Publisher(),
    Logger:         log,
})
// ...
auditMod.Subscribe(bus)  // avant bus.Run
```

## Fichiers impactés

| Fichier | Changement |
|---------|-----------|
| `kernel/events/events.go` | `Event.ID string` ajouté |
| `kernel/events/dispatcher.go` | `event.ID = id` dans `fetchBatch` |
| `platform/audit/migrations/0003_*` | `outbox_event_id UUID UNIQUE` |
| `platform/audit/domain/audit_log.go` | `OutboxEventID string` |
| `platform/audit/infrastructure/query.go` | INSERT + `ON CONFLICT (outbox_event_id) DO NOTHING` |
| `platform/audit/infrastructure/postgres-repo.go` | 9ème param `OutboxEventID` |
| `platform/audit/usecase/log_event.go` | Réécriture synchrone + typed payload |
| `platform/audit/usecase/consume_audit_log.go` | Nouveau — consumer handler |
| `platform/audit/audit.go` | `New` → `(*Module, error)`, `Subscribe`, Config étendu |
| `platform/audit/usecase/audit_test.go` | Tests consumer, suppression fire-and-forget |
| `platform/auth/usecase/auth_test.go` | `noopAuditLogger` (noop stub) |
| `examples/saas-b2b/main.go` | Wiring `DB`, `Publisher`, `auditMod.Subscribe(bus)` |
