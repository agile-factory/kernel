# _RFC #0004:_ Platform `membership` — Cycle de Vie des Appartenances Multi-Tenant

**date:** `2026 May 01`

---

## **Status:** `ACCEPTED`

---

## **Summary**

Ce RFC définit la plateforme `membership`, brique SaaS réutilisable qui gouverne la relation entre un **utilisateur** et un **tenant** dans le socle Agile Factory.

La plateforme couvre le cycle de vie complet d'une appartenance :

```
invite → (PENDING) → accept → (ACTIVE) → revoke → (REVOKED)
```

Elle expose quatre use cases (`invite`, `accept`, `revoke`, `list`) via une API HTTP REST et persiste les données dans la table `memberships` introduite dans RFC-0003.

---

## **Motivation**

Le RFC-0003 a acté le modèle `memberships` (many-to-many user ↔ tenant avec rôle et statut). Mais il n'a pas défini le **comportement applicatif** : comment un admin invite-t-il un utilisateur ? Comment ce dernier accepte-t-il ? Qui peut révoquer ? Ces règles doivent être implémentées de manière cohérente, testable et réutilisable.

Sans cette plateforme :
- Chaque SaaS réimplémente sa propre logique d'invitation
- Les règles de sécurité (seul un admin peut inviter, seul le destinataire peut accepter) sont dispersées dans les handlers HTTP
- L'état `PENDING/ACTIVE/REVOKED` est géré ad hoc, sans machine à états formelle

---

## **Proposal**

### 1. Modèle de domaine

```go
type Membership struct {
    ID        string
    UserID    string
    TenantID  string
    Role      MemberRole   // ROLE_ADMIN | ROLE_OWNER| ROLE_AGENT | ROLE_VIEWER
    Status    MemberStatus // PENDING | ACTIVE | REVOKED
    InvitedBy string       // vide si auto-inscription
    JoinedAt  *time.Time   // nil jusqu'à Accept()
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Deux constructeurs domaine :
- `NewSelfMembership(...)` → statut `ACTIVE` immédiat, pour l'auto-inscription
- `NewInvitedMembership(...)` → statut `PENDING`, pour les invitations

Deux méthodes de transition d'état :
- `Accept()` → `PENDING → ACTIVE`, enregistre `JoinedAt`
- `Revoke()` → `ACTIVE/PENDING → REVOKED`

### 2. Use Cases

| Use Case | Règles métier |
|---|---|
| `InviteMemberUseCase` | Inviteur doit être ROLE_ADMIN **ou ROLE_OWNER** ACTIVE du tenant ; invité ne doit pas avoir de membership existant ; **ROLE_OWNER ne peut pas être assigné via invitation** (`ErrCannotInviteAsOwner`) |
| `AcceptInviteUseCase` | Seul le destinataire (`UserID == callerID`) peut accepter sa propre invitation |
| `RevokeMemberUseCase` | Seul un ROLE_ADMIN ou ROLE_OWNER ACTIVE peut révoquer ; un admin ne peut pas se révoquer lui-même ; **ROLE_OWNER ne peut pas être révoqué** (`ErrCannotRevokeOwner`) |
| `ListMembersUseCase` | Lecture paginée par tenant ou par user |

### 3. API HTTP

```
POST /api/v1/memberships/invite           → 201 Created
POST /api/v1/memberships/{id}/accept      → 200 OK
POST /api/v1/memberships/{id}/revoke      → 200 OK
GET  /api/v1/memberships/tenant/{tenantId}?limit=50&offset=0  → 200 OK (paginé)
GET  /api/v1/memberships/user/{userId}?limit=50&offset=0      → 200 OK (paginé)
```

Authentification : header `X-User-ID` (injecté par le middleware d'auth).

### 4. Schéma SQL

```sql
CREATE TABLE memberships (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    role        member_role   NOT NULL DEFAULT 'ROLE_VIEWER',
    status      member_status NOT NULL DEFAULT 'PENDING',
    invited_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    joined_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_memberships_user_tenant UNIQUE (user_id, tenant_id)
);
```

---

## **Drawbacks**

### Couplage transversal
La plateforme `membership` est consommée par `auth` (auto-inscription), `tenant` (création), et `audit`. Ce couplage est géré par injection de dépendances, mais toute évolution du port `MembershipRepository` nécessite des mises à jour dans plusieurs use cases.

### Pas de notifications intégrées
La v1 ne notifie pas l'invité (email, SMS). Un flux d'invitation complet nécessitera l'intégration de la future plateforme `notification`.

### Unicité user/tenant
La contrainte `UNIQUE (user_id, tenant_id)` empêche les scénarios de réinvitation après révocation. En v2, envisager de permettre plusieurs memberships (un seul `ACTIVE` à la fois).

---

## **Alternatives**

### Invitation par email direct (sans user_id préexistant)
- ✅ Meilleure UX : l'invité n'a pas besoin d'un compte existant
- ❌ Nécessite un système de tokens d'invitation temporaires + envoi d'email
- **Verdict : différé** — à implémenter avec la plateforme `notification`

### Rôles définis dans le domaine `permission` (RBAC granulaire)
- ✅ Plus flexible (permissions par ressource, pas juste par rôle)
- ❌ Complexité excessive pour la v1
- **Verdict : différé** — `platform/permission` est prévu dans RFC-0002

### Invitation via lien public (magic link)
- ✅ Onboarding plus fluide
- ❌ Gestion de l'expiration, token store, sécurité
- **Verdict : différé** — Phase 3

---

## **References**

- RFC-0002 : Architecture & Stack Technique du Socle SaaS
- RFC-0003 : Multi-Tenancy Architecture du Socle SaaS
- ADR-0003 : Multi-Tenancy Shared Schema avec Table `memberships`
- ADR-0008 : Injection de Dépendances Manuelle
