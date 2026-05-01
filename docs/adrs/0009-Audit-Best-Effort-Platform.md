# _ADR #0009:_ Piste d'Audit Best-Effort avec `platform/audit`

**date:** `2026 May 01`

---

## **Status:** `IMPLEMENTED`

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

### Stratégie de lecture : **Admin-only, paginé**

La lecture s'effectue à deux niveaux d'isolation :

1. **Vue Tenant** (`GET /api/v1/tenants/{tenantId}/audit-logs`) :
   Seul un utilisateur avec `ROLE_OWNER` ou `ROLE_ADMIN` et un membership `ACTIVE` dans ce tenant peut lire ses logs. La vérification est faite dans le handler via le `MembershipRepository.FindByUserAndTenant()`.

2. **Vue Globale** (`GET /api/v1/audit-logs`) :
   Réservée aux super-administrateurs de la plateforme. La vérification se base sur le rôle du JWT principal injecté dans le contexte (`kernelauth.IsPlatformAdmin()`).

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

### **Cons**
- Best-effort = possibilité de logs manquants en cas de panne DB persistante
- Les erreurs d'audit silencieuses doivent être monitorées séparément (métriques, alertes)
- Pas de garantie d'ordre des événements (OccurredAt peut différer du commit de l'opération)
