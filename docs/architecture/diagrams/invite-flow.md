# Séquence — Invitation & acceptation d'un membre

Deux flux distincts qui se rejoignent côté DB sur la table `invite_tokens` (RFC-0018 §6). Le token brut **n'est jamais persisté en clair après émission** — il est dans la réponse HTTP de `/invite` (à transmettre au mail) et stocké tel quel dans `invite_tokens.token` avec un index partial unique sur les rows pending.

---

## 1. Invitation (OWNER/ADMIN invite un email)

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Admin authentifié
    participant Handler as membership/delivery/http<br/>MembershipHandler
    participant UC as membership/usecase<br/>InviteMemberUseCase
    participant TenantRepo as tenant.Repository
    participant UserRepo as auth.UserRepo<br/>(UserLookup)
    participant InviteRepo as membership<br/>InviteTokenRepository
    participant Tx as kernel/txctx
    participant DB as PostgreSQL
    participant Bus as kernel/events
    participant Audit as platform/audit

    Admin->>Handler: POST /api/v1/memberships/invite<br/>Bearer JWT + {tenantId, email, role}
    Handler->>Handler: DTO.Validate() (role ∈ {ADMIN, AGENT, VIEWER})
    Handler->>Handler: MembershipRepo.IsOwnerOrAdmin(actor, tenantId)?
    alt Caller non autorisé
        Handler-->>Admin: 403 PERMISSION_DENIED
    end

    Handler->>UC: Execute(ctx, params)
    UC->>TenantRepo: FindByID(tenantId)
    TenantRepo-->>UC: tenant (must exist + active)
    UC->>UserRepo: FindByEmail(email)
    UserRepo-->>UC: user OR nil (flag: existing)

    UC->>Tx: ExecInTx(ctx, pool, fn)
    Tx->>DB: BEGIN
    UC->>UC: generate token (128 chars URL-safe)
    UC->>InviteRepo: Save(tenantId, email, role, token, 7d-TTL)
    InviteRepo->>DB: INSERT INTO invite_tokens
    DB-->>InviteRepo: invite_token row
    UC->>Bus: Publish("member.invited", {email, token, role, ...})
    Bus->>DB: INSERT INTO event_outbox
    UC-->>Tx: nil
    Tx->>DB: COMMIT

    UC->>Audit: Execute(MEMBER_INVITED)
    UC-->>Handler: invite_token row + raw token
    Handler-->>Admin: 201 Created<br/>{token, expiresAt, ...}

    Note over Bus,DB: Outbox dispatcher → notification.handleMemberInvited<br/>→ template member_invited → EmailSender.Send
```

---

## 2. Acceptation (invité clique sur le lien dans le mail)

```mermaid
sequenceDiagram
    autonumber
    actor Invitee as Invité (non-authentifié)
    participant Handler as auth/delivery/http<br/>AuthHandler
    participant UC as auth/usecase<br/>AcceptInviteUseCase
    participant InviteRepo as invite_token<br/>repo
    participant UserRepo as auth.UserRepo
    participant MemberRepo as membership<br/>Repository
    participant Tx as kernel/txctx
    participant DB as PostgreSQL
    participant Bus as kernel/events
    participant Audit as platform/audit

    Invitee->>Handler: POST /api/v1/auth/accept-invite<br/>{token, password?}
    Handler->>Handler: DTO.Validate()
    Handler->>UC: Execute(ctx, params)

    UC->>Tx: ExecInTx(ctx, pool, fn)
    Tx->>DB: BEGIN

    UC->>InviteRepo: FindByToken(token)
    InviteRepo->>DB: SELECT ... WHERE token = $1<br/>AND consumed_at IS NULL<br/>AND revoked_at IS NULL<br/>AND expires_at > NOW()
    DB-->>InviteRepo: invite row
    alt Token invalide / expiré / consommé
        UC-->>Handler: 400 INVALID_TOKEN / 410 GONE
        Handler-->>Invitee: error
    end

    UC->>UserRepo: FindByEmail(invite.email)
    UserRepo-->>UC: user OR nil

    alt User n'existe pas
        UC->>UC: password requis → bcrypt(password)
        UC->>UserRepo: Save(new user, email_verified=true)
        Note over UserRepo: le passage par l'invite vaut<br/>vérification d'email
    end

    UC->>MemberRepo: Save(user_id, tenant_id, role, ACTIVE)
    UC->>InviteRepo: MarkConsumed(token)
    UC->>UC: generate refresh token
    UC->>DB: INSERT INTO refresh_tokens (token_hash, tenant_id)

    UC->>Bus: Publish("auth.invite_accepted", {...})
    Bus->>DB: INSERT INTO event_outbox

    UC-->>Tx: nil
    Tx->>DB: COMMIT

    UC->>Audit: Execute(MEMBER_ACCEPTED)
    UC->>UC: sign JWT
    UC-->>Handler: {userId, membershipId, tenantId, accessToken, refreshToken}
    Handler-->>Invitee: 200 OK
```

## Invariants

| # | Invariant | Implémentation |
|---|---|---|
| 1 | **Pas de double-redemption** | `UNIQUE(token)` + `consumed_at IS NULL` partial index. Race tx isole. |
| 2 | **Pas de membership PENDING** | RFC-0018 §6 remplace le couple `(membership PENDING + token)` par un seul `invite_token` row. |
| 3 | **Email auto-vérifié à l'acceptation** | Le clic sur le lien dans le mail = preuve de possession de l'email. Pas de `email_verification_tokens` row pour ce parcours. |
| 4 | **Token ne porte AUCUN PII en lui-même** | C'est juste une chaîne aléatoire. Email + rôle vivent en DB. |
| 5 | **Réinvitation après révocation** | `UNIQUE(user_id, tenant_id) WHERE status != 'REVOKED'` → un user révoqué peut être réinvité. |

## Références code

- `platform/membership/usecase/invite.go` — flux invite.
- `platform/auth/usecase/accept_invite.go` — flux accept.
- `platform/membership/migrations/0003_create_invite_tokens.up.sql`.
- [RFC-0018](../rfcs/0018-Amendements-B2B-Onboarding.md) §6.
