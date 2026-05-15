# Séquence — Outbox transactionnel (publish + dispatch)

RFC-0015 v1.1 — at-least-once via PostgreSQL outbox + single-writer dispatcher avec `pg_try_advisory_lock`. Garantit qu'un événement n'est pas perdu si le commit business commit, et qu'il n'est pas publié si la tx rollback.

---

## Publish (côté use case)

```mermaid
sequenceDiagram
    autonumber
    participant UC as Use case<br/>(ex. SignupOwner)
    participant Tx as kernel/txctx
    participant Pub as OutboxPublisher
    participant DB as PostgreSQL

    UC->>Tx: ExecInTx(ctx, pool, fn)
    Tx->>DB: BEGIN
    Tx->>Tx: install tx in ctx

    UC->>DB: INSERT business row
    DB-->>UC: ok

    UC->>Pub: Publish(ctx, Event{Type, Payload, Meta})
    Pub->>Tx: MustFromContext(ctx)<br/>(retrieve installed tx)
    Tx-->>Pub: tx
    Pub->>DB: INSERT INTO event_outbox<br/>(event_type, payload JSONB, meta JSONB)
    Note over Pub,DB: Même tx que l'insert business →<br/>atomicité garantie

    UC-->>Tx: nil (success)
    Tx->>DB: COMMIT

    alt UC retourne err
        Tx->>DB: ROLLBACK
        Note over DB: Business row ET event_outbox row<br/>annulées ensemble
    end
```

## Dispatch (goroutine `bus.Run(ctx)`)

```mermaid
sequenceDiagram
    autonumber
    participant Disp as OutboxDispatcher
    participant DB as PostgreSQL
    participant H as Handler<br/>(ex. notification)

    loop chaque PollInterval (défaut 5s)
        Disp->>DB: pg_try_advisory_lock(advisoryLockID)
        alt Lock pris par autre instance
            DB-->>Disp: false
            Note over Disp: skip ce tick, single-writer
        else Lock acquis
            DB-->>Disp: true
            Disp->>DB: BEGIN
            Disp->>DB: SELECT * FROM event_outbox<br/>WHERE published_at IS NULL<br/>AND next_attempt_at <= NOW()<br/>AND attempts < max_attempts<br/>ORDER BY created_at<br/>LIMIT batch_size<br/>FOR UPDATE SKIP LOCKED

            DB-->>Disp: batch d'événements
            loop pour chaque event
                Disp->>H: Handle(ctx, event)
                alt Handler succès
                    H-->>Disp: nil
                    Disp->>DB: UPDATE event_outbox<br/>SET published_at = NOW()
                else Handler échec
                    H-->>Disp: error
                    Disp->>DB: UPDATE event_outbox<br/>SET attempts = attempts+1,<br/>last_error = $1,<br/>next_attempt_at = NOW() + backoff(attempts)
                    Note over Disp: backoff exponentiel<br/>(2^attempts secondes, max 1h)
                end
            end
            Disp->>DB: COMMIT
            Disp->>DB: pg_advisory_unlock(advisoryLockID)
        end
    end

    Note over Disp,DB: Si attempts == max_attempts (défaut 10) :<br/>l'event reste en table, published_at NULL<br/>→ quarantaine, listing admin manuel
```

## Garanties

| Garantie | Mécanisme |
|---|---|
| **At-least-once** | Pas d'`ACK before commit`. Si crash après handler succès mais avant `UPDATE published_at`, le handler sera rejoué au prochain poll. |
| **Atomicité publish/business** | `INSERT INTO event_outbox` dans la même tx que le business write. |
| **Single-writer** | `pg_try_advisory_lock` rend `bus.Run` idempotent — déployer N instances ne duplique pas. |
| **Pas de starvation** | `SKIP LOCKED` permet à un instance B de prendre un event que l'instance A vient de skip à cause d'un lock row, sans deadlock. |
| **Backoff** | Exponentiel `2^attempts` borné à 1h. Évite d'éroder le throughput sur une dépendance down. |

## Antipatterns à éviter

- ❌ **Publish hors tx** : `Publisher.Publish` lève `txctx.ErrNoTx`. C'est volontaire — un event publié hors tx peut leaker un état non commité.
- ❌ **Handler avec side-effects non-idempotents** : un email peut être envoyé deux fois (at-least-once). Le handler `notification` log dans `notification_logs` chaque tentative → l'opérateur voit les doublons, mais le destinataire aussi.
- ❌ **Long handler bloquant** : un handler de 30s bloque tout le batch. Si nécessaire, faire le handler enqueue → travail async hors batch.

## Configuration recommandée

| Champ `events.Config` | Dev | Prod |
|---|---|---|
| `PollInterval` | 1s | 5s |
| `BatchSize` | 10 | 50 |
| `MaxAttempts` | 3 | 10 |
| `AdvisoryLockID` | constant (ex. `0x6F75747820626F78`) | idem (doit être stable entre déploiements) |

## Références code

- `kernel/events/publisher.go` — OutboxPublisher.
- `kernel/events/dispatcher.go` — OutboxDispatcher.
- `kernel/events/migrations/0001_create_event_outbox.up.sql`.
- [RFC-0015](../rfcs/0015-Event-Bus-Interne.md) — spec complète.
