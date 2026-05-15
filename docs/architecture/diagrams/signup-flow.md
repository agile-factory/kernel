# Séquence — `POST /api/v1/auth/signup-owner`

Crée tenant + user + membership OWNER + token de vérification email + access/refresh tokens, dans **une seule transaction**. Émet un événement `auth.signup_owner` dans la même tx.

```mermaid
sequenceDiagram
    autonumber
    actor Client
    participant Handler as auth/delivery/http<br/>AuthHandler
    participant UC as auth/usecase<br/>SignupOwnerUseCase
    participant Tx as kernel/txctx
    participant DB as PostgreSQL
    participant Bus as kernel/events<br/>OutboxPublisher
    participant Audit as platform/audit<br/>EventLogger

    Client->>Handler: POST /api/v1/auth/signup-owner<br/>{tenantName, tenantSlug, email, password}
    Handler->>Handler: DTO.Validate()<br/>(password ≥ 12 chars, etc.)
    Handler->>UC: Execute(ctx, params)

    UC->>Tx: ExecInTx(ctx, pool, fn)
    Tx->>DB: BEGIN

    UC->>DB: INSERT INTO tenants (...)
    DB-->>UC: tenant_id

    UC->>DB: INSERT INTO tenant_settings (...)
    DB-->>UC: ok

    UC->>UC: bcrypt(password)
    UC->>DB: INSERT INTO users (...)
    DB-->>UC: user_id

    UC->>DB: INSERT INTO memberships<br/>(user_id, tenant_id, ROLE_OWNER, ACTIVE)
    DB-->>UC: membership_id

    UC->>UC: generate verify token (128 chars)
    UC->>DB: INSERT INTO email_verification_tokens
    DB-->>UC: ok

    UC->>UC: generate refresh token + SHA-256(token)
    UC->>DB: INSERT INTO refresh_tokens (token_hash)
    DB-->>UC: ok

    UC->>Bus: Publish("auth.signup_owner",<br/>{email, verification_token, tenant_name})
    Bus->>DB: INSERT INTO event_outbox (...)
    DB-->>Bus: ok

    UC-->>Tx: nil (success)
    Tx->>DB: COMMIT
    DB-->>Tx: ok

    UC->>Audit: Execute(LogEventParams)<br/>USER_REGISTERED + TENANT_CREATED
    Note over Audit: fire-and-forget<br/>(goroutine + ctx detached)

    UC->>UC: sign JWT (access token, 1h)
    UC-->>Handler: {tenantId, userId, accessToken, refreshToken, ...}
    Handler-->>Client: 201 Created

    Note over Bus,DB: Outbox dispatcher (autre goroutine)<br/>poll → handler notification<br/>→ envoi email de vérification
```

## Invariants

| # | Invariant | Justification |
|---|---|---|
| 1 | **Tout ou rien** | La tx couvre tenants → users → memberships → tokens → event_outbox. Un rollback annule **toutes** les écritures, y compris l'événement → pas de fantôme d'email envoyé sans user persisté. |
| 2 | **Audit hors tx** | `Audit.Execute` se déclenche après le commit. Si l'audit échoue, le signup reste valide (ADR-0009 best-effort). |
| 3 | **Refresh token tenant-scoped** | À la création, le token est lié au tenant qu'on vient de créer (`refresh_tokens.tenant_id` non-nul). Les login suivants émettront des tokens non-scopés. |
| 4 | **Email non-vérifié ≠ blocage** | L'access token est délivré immédiatement (`EmailVerificationPending: true`). Les login ultérieurs **exigent** la vérification (RFC-0018 §4). |

## Variantes d'échec

- **Slug déjà pris** : `409 Conflict` → `TENANT_ALREADY_EXISTS`.
- **Email déjà pris** : `409 Conflict` → `USER_ALREADY_EXISTS`.
- **Password < 12 chars** : `422 Unprocessable Entity` → `VALIDATION_FAILED`.
- **Publisher nil** (mode dégradé) : la tx commit sans émettre l'événement → pas d'email envoyé. Mode dev/test uniquement.

## Références code

- `platform/auth/usecase/signup_owner.go` — orchestrateur.
- `platform/auth/delivery/http/handler.go` — handler.
- `platform/auth/domain/email.go` — validation.
- [RFC-0018](../rfcs/0018-Amendements-B2B-Onboarding.md) §3 — spec.
