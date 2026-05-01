# _RFC #0005:_ Platform `audit` — Piste d'Audit Transversale

**date:** `2026 May 01`

---

## **Status:** `ACCEPTED`

---

## **Summary**

Ce RFC définit la plateforme `audit`, brique SaaS transversale qui enregistre une trace immuable des actions significatives effectuées dans le système.

La plateforme est **push-only** en Phase 1 (écriture best-effort par tous les use cases concernés) et expose un endpoint de lecture paginé en Phase 2 (ROLE_ADMIN uniquement).

```
Use Case métier → LogEventUseCase.Execute() → audit_logs (DB)
                                                      ↓
                              GET /tenants/{id}/audit-logs (Phase 2)
```

---

## **Motivation**

Un SaaS multi-tenant a des besoins de traçabilité sur plusieurs dimensions :

- **Sécurité** : savoir qui a invité qui, qui s'est connecté, qui a révoqué un accès
- **Conformité** : répondre aux questions "qui a fait quoi et quand ?" (RGPD, audits internes)
- **Support** : débugger des incidents sans accéder aux logs système
- **Transparence** : permettre aux administrateurs d'un tenant de voir l'historique d'activité de leur organisation

Sans piste d'audit centralisée :
- Les logs applicatifs (`app.log`) sont non structurés pour la recherche par tenant/action
- Il n'y a pas de point d'entrée API pour qu'un admin consulte l'historique
- Chaque SaaS réimplémente son propre mécanisme de traçabilité

---

## **Proposal**

### 1. Structure de la plateforme

```
platform/audit/
├── domain/
│   ├── audit_log.go       ← aggregate AuditLog, enum AuditAction
│   ├── repository.go      ← port AuditRepository (Save + ListByTenant)
│   └── errors.go          ← ErrAuditSaveFailed
├── infrastructure/
│   ├── postgres-repo.go   ← implémentation pgx
│   └── query.go           ← SQL constants
├── usecase/
│   ├── log_event.go       ← LogEventUseCase (fire-and-forget)
│   ├── list_audit_logs.go ← ListAuditLogsUseCase (paginé, max 200)
│   └── params.go          ← LogEventParams
└── delivery/http/
    ├── handler.go         ← AuditHandler (GET avec garde ROLE_ADMIN)
    ├── dto.go             ← AuditLogResponse, AuditLogListResponse
    └── routes.go          ← GET /api/v1/tenants/{tenantId}/audit-logs
```

### 2. Aggregate AuditLog

```go
type AuditLog struct {
    ID         string
    TenantID   string          // vide pour événements système
    ActorID    string          // vide pour actions automatiques
    Action     AuditAction     // ENUM typé
    TargetType string          // "membership", "user", "tenant"
    TargetID   string          // UUID de la ressource affectée
    Metadata   map[string]any  // JSONB contextuel
    OccurredAt time.Time
}
```

Actions couvertes en v1 :

| Action | Déclenché par |
|---|---|
| `MEMBER_INVITED` | `InviteMemberUseCase` |
| `MEMBER_ACCEPTED` | `AcceptInviteUseCase` |
| `MEMBER_REVOKED` | `RevokeMemberUseCase` |
| `USER_REGISTERED` | `RegisterUseCase` |
| `USER_LOGGED_IN` | `LoginUseCase` |
| `TENANT_CREATED` | `CreateTenantUseCase` |

### 3. Stratégie d'écriture : Best-Effort

```go
func (uc *LogEventUseCase) Execute(ctx context.Context, params LogEventParams) {
    log := domain.NewAuditLog(uuid.NewString(), ...)
    _ = uc.repo.Save(ctx, log) // erreur ignorée délibérément
}
```

**Principe** : un échec d'écriture de l'audit ne propage jamais d'erreur vers l'opération métier parente. L'audit est un enrichissement, pas un prérequis.

### 4. Stratégie de lecture : Admin-only, paginé

```
GET /api/v1/tenants/{tenantId}/audit-logs?limit=50&offset=0
```

**Garde d'autorisation** : le handler vérifie via `MembershipRepository.FindByUserAndTenant()` que le caller est `ROLE_ADMIN` et `ACTIVE` dans le tenant demandé.

**Réponse** :
```json
{
    "data": [
        {
            "id": "...",
            "action": "MEMBER_INVITED",
            "actorId": "...",
            "targetType": "membership",
            "targetId": "...",
            "metadata": { "invitee_id": "...", "role": "ROLE_AGENT" },
            "occurredAt": "2026-05-01T04:00:00Z"
        }
    ],
    "total": 1,
    "limit": 50,
    "offset": 0
}
```

### 5. Schéma SQL

```sql
CREATE TYPE audit_action AS ENUM (
    'MEMBER_INVITED', 'MEMBER_ACCEPTED', 'MEMBER_REVOKED',
    'USER_REGISTERED', 'USER_LOGGED_IN', 'TENANT_CREATED'
);

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID REFERENCES tenants(id) ON DELETE SET NULL,
    actor_id    UUID REFERENCES users(id)   ON DELETE SET NULL,
    action      audit_action NOT NULL,
    target_type VARCHAR(64),
    target_id   UUID,
    metadata    JSONB        NOT NULL DEFAULT '{}',
    occurred_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

Index : `tenant_id`, `actor_id`, `action`, `occurred_at DESC`.

### 6. Injection dans les use cases

Le `LogEventUseCase` est injecté comme dépendance optionnelle dans les use cases métier. Le pattern est uniforme :

```go
// Après l'opération principale réussie :
uc.audit.Execute(ctx, auditusecase.LogEventParams{
    TenantID:   "...",
    ActorID:    "...",
    Action:     auditdomain.ACTION_MEMBER_INVITED,
    TargetType: "membership",
    TargetID:   membership.ID,
    Metadata:   map[string]any{"role": role},
})
```

---

## **Drawbacks**

### Best-effort = logs potentiellement manquants
En cas de panne DB persistante, des actions peuvent ne pas être auditées. Les erreurs silencieuses doivent être monitorées (métriques, alertes).

### Pas d'ordre garanti
`OccurredAt` est défini à la création de l'objet Go, pas au moment du commit SQL. En cas de concurrent élevé, l'ordre des logs peut différer légèrement de l'ordre réel des transactions.

### `ENUM audit_action` rigide
Ajouter un nouveau type d'action nécessite une migration SQL (`ALTER TYPE ... ADD VALUE`). Ce n'est pas rollbackable directement.

### Croissance non bornée
La table `audit_logs` croît sans TTL. Un mécanisme de rétention/archivage sera nécessaire en production.

---

## **Alternatives**

### Audit transactionnel (même transaction SQL)
- ✅ Garantie forte : impossible de rater un log si l'opération commit
- ❌ Couplage fort entre l'opération et son audit (latence, rollback en cascade)
- ❌ Nécessite de passer une `*pgx.Tx` à travers toutes les couches
- **Verdict : différé** — pour les exigences de conformité SOC2/RGPD strictes

### Outbox pattern (table outbox + worker async)
- ✅ At-least-once delivery, résilient aux pannes
- ❌ Complexité opérationnelle élevée (worker, retry, déduplication)
- **Verdict : Phase 3** — si la conformité réglementaire l'exige

### Event sourcing complet
- ✅ Replay d'état possible, audit parfait
- ❌ Paradigme très différent, incompatible avec la stack CRUD actuelle
- **Verdict : hors scope**

### Logs applicatifs uniquement (pas de table SQL)
- ✅ Zéro table supplémentaire
- ❌ Non interrogeable par tenant via API
- ❌ Non présentable aux administrateurs
- **Verdict : complémentaire**, pas exclusif

---

## **Open Questions**

1. **Rétention** : TTL de 12 mois ? 24 mois ? Archivage dans S3 ?
2. **Super-admin** : accès cross-tenant aux logs pour le support interne ?
3. **Export** : permettre l'export CSV des logs d'audit via l'API ?
4. **ENUM vs VARCHAR** : VARCHAR pour `action` serait plus flexible, mais perd la validation DB native.

---

## **References**

- RFC-0002 : Architecture & Stack Technique du Socle SaaS
- RFC-0003 : Multi-Tenancy Architecture
- RFC-0004 : Platform Membership
- ADR-0009 : Audit Best-Effort Platform
- ADR-0010 : Pagination Uniforme
