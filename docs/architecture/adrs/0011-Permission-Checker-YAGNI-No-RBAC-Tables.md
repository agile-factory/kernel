# _ADR #0011:_ PermissionChecker v1 — Interface only, sans tables RBAC (YAGNI)

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0010 décrivait un RBAC complet avec 3 nouvelles tables SQL (`roles`, `role_permissions`, `user_roles`) permettant aux admins de créer des rôles personnalisés par tenant. Avant implémentation, l'analyse P1 a révélé que :

1. **Aucun SaaS consommateur** n'avait encore de besoin réel de rôles personnalisés — tous les cas d'usage vus en P0 (invite, revoke, audit, tenant settings) se résolvent avec les 4 rôles fixes : OWNER, ADMIN, AGENT, VIEWER.
2. **Ajouter les 3 tables en avance** aurait introduit ~400 lignes de code (migrations, repos, use-cases, HTTP handlers) pour un besoin hypothétique, augmentant la surface de test et la charge de maintenance.
3. **Le `membership.role` existe déjà** en base (P0 livré) — il est la source naturelle d'autorité pour la v1.

Décision prise en accord avec le principe **YAGNI** (You Aren't Gonna Need It) et validée dans RFC-0039 avant implémentation.

---

## **Decision**

`kernel/permission/` expose une interface `Checker` avec une seule méthode `Require(ctx, perm)`. L'implémentation v1 (`MembershipBasedChecker`) :

- Lit `userID` et `tenantID` depuis le context (injectés par le middleware HTTP existant via `kernel/auth`)
- Appelle `membership.FindByUserAndTenant` — **aucune nouvelle requête DB introduite**, juste une réutilisation du repo déjà injecté dans les use-cases
- Consulte une map statique `rolePermissions` (en mémoire, au démarrage) qui associe chaque `MemberRole` à un `Set<Permission>`
- Les `Permission` sont des constantes string de la forme `module:action` déclarées dans `kernel/permission/permission.go`

Zéro nouvelle table SQL. Zéro nouvelle migration. Le package ne porte aucun `embed.FS`.

**Critère de déclenchement v2 :** un SaaS consommateur ouvre une issue GitHub avec un cas d'usage concret de rôle personnalisé. À ce moment, les 3 tables RFC-0010 sont ajoutées et `MembershipBasedChecker` est remplacé par une implémentation DB-backed, sans changer l'interface `Checker`.

---

## **Alternatives Considered**

### A. Implémenter les 3 tables RBAC dès v1 (RFC-0010 original)

- ✅ Prêt pour les rôles custom dès le départ
- ❌ ~400 lignes de code, 3 migrations, 3 repos, overhead de maintenance pour 0 consommateur actuel
- **Verdict : rejeté — YAGNI**

### B. Utiliser uniquement `auth.IsPlatformAdmin` sans interface

- ✅ Ultra-simple
- ❌ Pas de granularité par permission, impossible d'exprimer "ADMIN peut faire X mais pas Y"
- **Verdict : rejeté — insuffisant pour les use-cases P1 (billing, notification)**

### C. Middleware HTTP au lieu d'interface injectée dans les use-cases

- ✅ Centralisé, déclaratif
- ❌ Couplage fort entre routing et logique métier ; les use-cases ne peuvent pas tester les permissions sans HTTP
- **Verdict : rejeté — l'interface injectée est plus testable et suit la Clean Architecture (ADR-0002)**

---

## **Consequences**

### **Pros**

- P1-0 livré en ~150 lignes (vs ~550 estimées avec les tables)
- Interface stable : les use-cases de P1-1 à P1-4 peuvent commencer à utiliser `checker.Require()` immédiatement
- 8 tests unitaires, aucune dépendance DB (stub in-memory) — rapide en CI
- La migration v2 ne touchera que l'implémentation, pas les callsites

### **Cons**

- Les 4 rôles fixes ne couvrent pas les besoins de SaaS avec des workflows complexes (ex: CRM multi-équipes)
- La map statique `rolePermissions` doit être mise à jour manuellement quand un nouveau module ajoute des permissions — risque d'oubli sans test de couverture
- Pas de support "rôle par tenant" (un utilisateur a le même rôle dans tous les tenants qu'il rejoint avec la même membership)
