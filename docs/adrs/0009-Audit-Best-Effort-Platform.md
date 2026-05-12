# _ADR #0009:_ Piste d'Audit Best-Effort avec `platform/audit`

**date:** `2026 May 01` — **mis à jour :** `2026 May 12`

> ⚠️ **AMENDEMENT (2026-05-12)** : La stratégie d'écriture **best-effort (fire-and-forget)** est **dépréciée** au profit de l'**outbox transactionnel at-least-once** (RFC-0015 v1.1). Cette migration est planifiée en P1-3. Voir [section Amendement](#amendement-2026-05-12--migration-vers-outbox-at-least-once) ci-dessous.

---

## **Status:** `IMPLEMENTED` — v2 cursor `IMPLEMENTED` — best-effort **DEPRECATED** (voir amendement)

---

## **Context**

Un SaaS multi-tenant a besoin de traçabilité des actions sensibles : qui a invité qui ? Qui a accepté une invitation ? Qui s'est connecté ? Ces données sont essentielles pour la conformité, le debugging et la transparence envers les administrateurs.

Deux questions de design se posent :
1. **Où persister** : dans la même transaction que l'opération principale, ou de manière indépendante ?
2. **Qui peut lire** : tout utilisateur ? Seulement les admins du tenant ? Quid des super-admins de la plateforme ?

---

## **Decision**

### Architecture

La plateforme `audit` est une **plateforme transversale** consommée par toutes les autres. Elle suit la même Clean Architecture (domain / infrastructure / usecase / delivery).

La table `audit_logs` utilise :
- `JSONB metadata` pour des données contextuelles flexibles sans changer le schéma
- `audit_action` ENUM pour des actions typées et indexables
- `tenant_id`, `actor_id`, `target_id` optionnels (NULL via `NULLIF('')`) pour les événements système

### Stratégie d'écriture : **Best-Effort (fire-and-forget)**

```go
func (uc *LogEventUseCase) Execute(ctx context.Context, params LogEventParams) {
    log := domain.NewAuditLog(...)
    _ = uc.repo.Save(ctx, log) // erreur délibérément ignorée
}
```

**Un échec d'audit ne propage pas d'erreur vers l'appelant et ne rollback jamais l'opération métier.**

### Stratégie de lecture : **Admin-only, paginé (v1 + v2)**

La lecture s'effectue à deux niveaux d'isolation, chacun exposé en v1 (offset) et v2 (cursor) :

| Version | Endpoint | Mécanique | Remarque |
|---|---|---|---|
| v1 | `GET /api/v1/tenants/{tenantId}/audit-logs` | `LIMIT/OFFSET` + `COUNT(*)` | Maintenu, stable |
| v2 | `GET /api/v2/tenants/{tenantId}/audit-logs` | Keyset cursor `(occurred_at, id)` | Haute performance |
| v1 | `GET /api/v1/audit-logs` | `LIMIT/OFFSET` + `COUNT(*)` | Maintenu, stable |
| v2 | `GET /api/v2/audit-logs` | Keyset cursor `(occurred_at, id)` | Super-admins only |

**Gardes d'autorisation (identiques v1/v2) :**

1. **Vue Tenant** (`/tenants/{tenantId}/audit-logs`) :
   Seul un utilisateur avec `ROLE_OWNER` ou `ROLE_ADMIN` et un membership `ACTIVE` dans ce tenant peut lire ses logs. La vérification est faite dans le handler via le `MembershipRepository.FindByUserAndTenant()`.

2. **Vue Globale** (`/audit-logs`) :
   Réservée aux super-administrateurs de la plateforme. La vérification se base sur le rôle du JWT principal injecté dans le contexte (`kernelauth.IsPlatformAdmin()`).

**Détails d'implémentation v2 :**
- Curseur opaque `base64url(json{ occurred_at, id })` — jamais construit côté client
- Keyset SQL : `WHERE (occurred_at, id) < ($N::timestamptz, $M::uuid) ORDER BY occurred_at DESC, id ASC`
- Technique `limit+1` pour détecter `has_more` sans `COUNT(*)`
- `total = -1` dans la réponse v2 (sentinelle mode cursor)
- Index composite ajouté via migration `202605010005` : `(tenant_id, occurred_at DESC, id ASC)` et `(occurred_at DESC, id ASC)`

---

## **Alternatives Considered**

### Audit transactionnel (même transaction que l'opération)
- ✅ Garantie forte : l'audit ne peut pas manquer si l'opération réussit
- ❌ Couplage fort : l'audit impacte la latence et peut faire échouer une invitation si la DB audit est lente
- ❌ Nécessite de passer une transaction SQL à travers toutes les couches (brise l'abstraction repo)
- **Verdict : rejeté pour la v1** — acceptable si la conformité réglementaire l'exige

### Outbox pattern (table outbox + worker async)
- ✅ Garantie "at-least-once" delivery
- ✅ Résilient aux pannes réseau
- ❌ Complexité opérationnelle élevée (worker, table outbox, retry logic)
- **Verdict : différé** — pour les besoins de conformité forte (SOC2, GDPR)

### Logging des audit events dans les logs applicatifs
- ✅ Aucune table supplémentaire
- ❌ Pas interrogeable (pas de filtre par tenant, par action)
- ❌ Non présentable aux utilisateurs finaux via l'API
- **Verdict : complémentaire**, pas exclusif

---

## **Consequences**

### **Pros**
- Un échec DB sur `audit_logs` n'impacte jamais l'expérience utilisateur
- La plateforme est injectée en tant que `*LogEventUseCase` sans coupler les use cases à l'infrastructure d'audit
- L'endpoint de lecture est accessible en Phase 2 sans modifier le domaine
- JSONB metadata permet d'enrichir les logs sans migration de schéma
- **v2** : pagination cursor-based sans `COUNT(*)` ni dégradation `OFFSET` — performances constantes même à plusieurs millions de lignes
- **v2** : routes `/api/v2/` additives, aucun breaking change sur les clients v1

### **Cons**
- Best-effort = possibilité de logs manquants en cas de panne DB persistante
- Les erreurs d'audit silencieuses doivent être monitorées séparément (métriques, alertes)
- Pas de garantie d'ordre des événements (OccurredAt peut différer du commit de l'opération)
- **v2** : `total = -1` est une sentinelle — les clients qui affichent "page X sur N" doivent utiliser la v1 ou adapter leur UI

---

## **Amendement 2026-05-12 — Migration vers Outbox at-least-once**

### Contexte de l'amendement

Le plan de spécialisation B2B du boilerplate (mai 2026) cible la conformité **SOC2 / ISO27001**. Ces certifications requièrent une traçabilité d'audit **fiable et vérifiable** — ce que le modèle best-effort ne peut garantir :

- Un crash entre l'opération métier et l'écriture de l'audit log laisse une **lacune silencieuse** dans la piste d'audit.
- Aucune garantie d'ordre ou d'exhaustivité.
- Non auditable par un organisme de certification tiers.

### Nouvelle stratégie : Outbox at-least-once

À partir de la phase P1-3, `platform/audit` migre vers la consommation de l'**outbox transactionnel** (RFC-0015 v1.1) :

```
Use Case
    → BEGIN TRANSACTION
        → opération métier (INSERT/UPDATE)
        → INSERT event_outbox {type: "audit.*", payload: {action, actor, tenant, ...}}
    → COMMIT

OutboxDispatcher (polling asynchrone)
    → SELECT FOR UPDATE SKIP LOCKED depuis event_outbox
    → AuditConsumer.Handle(event)
        → INSERT audit_logs
    → Marquer event comme traité
```

**Garanties** :
- L'event d'audit est inséré dans la **même transaction** que l'opération métier : si la transaction est committée, l'audit sera écrit (at-least-once).
- Un crash du dispatcher → reprise au prochain poll (l'event outbox n'est pas marqué traité).
- Idempotency : `processed_events` empêche la double-écriture en cas de replay.

### Impact sur le code

1. **Retirer** l'injection directe de `*LogEventUseCase` dans tous les use cases métier.
2. **Supprimer** l'appel `go uc.audit.Execute(...)` (goroutine fire-and-forget).
3. Les use cases émettent des events dans l'outbox via `events.Publisher` (déjà injecté).
4. Un nouveau `AuditConsumer` dans `platform/audit/` s'abonne à l'outbox et persiste les `audit_logs`.

### Compatibilité descendante

- L'**API de lecture** (`GET /api/v1/tenants/{id}/audit-logs`, v1 + v2 cursor) est inchangée.
- Les clients existants ne voient aucun breaking change.
- La table `audit_logs` reste inchangée.

### Signature HMAC optionnelle (SOC2 Tier 3 — P2)

Une extension optionnelle de la migration P1-3 permettra d'ajouter une **signature HMAC** sur chaque `audit_log` pour détecter les altérations. Définie dans RFC-0032 (Audit Compliance Tiers + Cryptographic Signing — à créer en P2).

### Références de l'amendement

- [RFC-0015 — Event Bus Outbox](../rfcs/0015-Event-Bus-Interne.md)
- Plan de spécialisation B2B/B2B2C — tâche P1-3
