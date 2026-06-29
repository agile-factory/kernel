# _RFC #0010:_ Platform `permission` — Contrôle d'Accès Basé sur les Rôles (RBAC)

**date:** `2026 May 01`

---

## **Status:** `PROPOSED — EN ATTENTE DE VALIDATION (Aziz / Lucas)`

> **Amendement 2026-06-28 (Cheikh)** : Questions ouvertes résolues ci-dessous. Scope v1 simplifié pour éviter la dette technique. Soumis pour revue avant implémentation P1.

---

## **Summary**

Ce RFC définit la plateforme `permission`, brique SaaS qui gouverne le **contrôle d'accès granulaire** au sein d'un tenant dans le socle Agile Factory.

La v1 utilise un RBAC simple à trois rôles (`ROLE_ADMIN`, `ROLE_AGENT`, `ROLE_VIEWER`) défini dans `platform/membership`. Ce RFC propose une évolution vers un **RBAC extensible par SaaS**, où les permissions sont déclarées par les modules métier et vérifiées de manière centralisée.

---

## **Motivation**

Le modèle actuel (`membership.role`) couvre les besoins de base mais est insuffisant pour des SaaS complexes :
- Un SaaS RH peut vouloir `HR_MANAGER`, `RECRUITER`, `INTERVIEWER`
- Un SaaS immobilier peut vouloir `OWNER`, `AGENT`, `INSPECTOR`
- Différents modules peuvent vouloir des permissions différentes sur la même ressource (`billing:read`, `billing:write`, `reports:export`)

Sans plateforme `permission` dédiée, ces besoins sont gérés ad hoc dans chaque module, créant des incohérences de sécurité.

---

## **Proposal**

### 1. Modèle de domaine

```go
// Permission atomique
type Permission struct {
    Resource string // "membership", "invoice", "report"
    Action   string // "read", "write", "delete", "export"
}
// Permission string: "membership:read", "invoice:write"

// Rôle = ensemble de permissions nommé
type Role struct {
    ID          string
    TenantID    string  // nil = rôle plateforme global
    Name        string  // "ROLE_ADMIN", "HR_MANAGER"
    Permissions []Permission
    IsSystem    bool    // true = non modifiable
}
```

### 2. Architecture en deux couches

**Couche 1 — Rôles système (plateforme)**  
Définis dans `platform/permission/domain`, non modifiables :
- `ROLE_ADMIN` → toutes permissions plateforme
- `ROLE_AGENT` → permissions de lecture + actions métier
- `ROLE_VIEWER` → lecture seule

**Couche 2 — Rôles personnalisés (par tenant)**  
Créés et configurés par les admins d'un tenant dans `modules/` :
- `HR_MANAGER` → `employee:write`, `payroll:read`, `recruitment:write`
- `RECRUITER` → `recruitment:write`, `interview:write`

### 3. Interface de vérification (kernel)

```go
// kernel/permission/checker.go
type PermissionChecker interface {
    Can(ctx context.Context, userID, tenantID string, perm Permission) (bool, error)
}
```

Utilisé dans les use cases :

```go
if ok, _ := uc.perm.Can(ctx, callerID, tenantID, permission.New("invoice", "write")); !ok {
    return nil, errors.Forbidden("insufficient permissions")
}
```

### 4. Schéma SQL

```sql
CREATE TABLE roles (
    id        UUID PRIMARY KEY,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name      VARCHAR(100) NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    UNIQUE (tenant_id, name)
);

CREATE TABLE role_permissions (
    role_id  UUID REFERENCES roles(id) ON DELETE CASCADE,
    resource VARCHAR(100) NOT NULL,
    action   VARCHAR(50)  NOT NULL,
    PRIMARY KEY (role_id, resource, action)
);

CREATE TABLE user_roles (
    user_id   UUID REFERENCES users(id)   ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    role_id   UUID REFERENCES roles(id)   ON DELETE CASCADE,
    PRIMARY KEY (user_id, tenant_id, role_id)
);
```

### 5. Migration depuis le RBAC v1

Le modèle `membership.role` (ROLE_ADMIN/AGENT/VIEWER) reste en place pour la compatibilité. La plateforme `permission` est **additive** : elle enrichit le contrôle d'accès sans supprimer le modèle existant.

---

## **Drawbacks**

### Complexité significative
RBAC granulaire nécessite plus de tables, plus de joins, et une interface de vérification cohérente dans tous les use cases.

### Cache nécessaire pour les performances
Vérifier les permissions à chaque requête implique des lectures DB fréquentes. Un cache (Redis ou in-memory avec TTL court) sera nécessaire en production.

### Risque de "permission hell"
Un système de permissions trop granulaire devient difficile à gérer pour les admins. UX soignée indispensable.

---

## **Alternatives**

### RBAC simplifié (3 rôles fixes, pour toujours)
- ✅ Simple, suffisant pour 80% des SaaS
- ❌ Bloquant pour les SaaS avec des rôles métier complexes
- **Verdict : maintenu en v1, RBAC granulaire en v2**

### ABAC (Attribute-Based Access Control)
- ✅ Le plus flexible (permissions basées sur attributs du contexte)
- ❌ Beaucoup plus complexe à implémenter et à débugger
- **Verdict : hors scope**

### OPA (Open Policy Agent)
- ✅ Standard industry, langage Rego pour les politiques
- ❌ Infrastructure externe, latence réseau
- **Verdict : différé** — pour les SaaS avec des exigences de conformité strictes

---

## **Open Questions — RÉSOLUES (amendement 2026-06-28)**

### Q1 : Création des rôles personnalisés — UI admin vs code ?

**Réponse : code-defined uniquement en v1, UI-configurable en v2.**

Raisonnement : aucune UI d'administration n'existe encore. Autoriser la création de rôles via l'API sans UI de gestion crée une surface de sécurité non maîtrisée. Les SaaS consommateurs déclarent leurs rôles métier dans leur `main.go` via `SeedRoles([]permission.RoleDefinition{...})` au démarrage. La migration vers des rôles configurables via UI est différée à v2, lorsqu'un vrai cas d'usage le justifie.

### Q2 : Audit des changements de permissions ?

**Réponse : oui, via RFC-0005 (audit trail existant), événements `permission.role_assigned` et `permission.role_revoked`.**

Les changements d'attribution de rôle sont des événements de sécurité critiques. Ils doivent être tracés dans `audit_logs` avec `actor_id`, `target_user_id`, `role_id`, `tenant_id`. L'implémentation réutilise le `AuditLogger` (interface RFC-0005) déjà injecté dans les modules.

### Q3 : Cache — application ou Redis ?

**Réponse : in-memory par requête (no-op cache en v1), Redis différé à v2.**

Ajouter Redis comme dépendance opérationnelle en P1 augmente la charge d'infrastructure sans gain mesuré (les SaaS P1 auront <1000 req/s). La stratégie v1 :
- `PermissionChecker` charge les rôles une seule fois par requête (lazy, via context)
- Pas de cache cross-requête
- Si les benchmarks P1 montrent une régression, Redis est ajouté en v2 comme backend optionnel du `PermissionChecker`

---

## **Recommandation de scope v1 — SIMPLIFICATION ANTI-DETTE TECHNIQUE**

> **Alerte prudence** : le schéma SQL proposé (3 nouvelles tables : `roles`, `role_permissions`, `user_roles`) introduit une complexité significative pour un besoin non encore prouvé par les SaaS consommateurs.

### Scope v1 recommandé (sans les 3 tables)

**Phase v1 :** `PermissionChecker` interface uniquement, branché sur `membership.role`

```go
// kernel/permission/checker.go — AUCUNE nouvelle table SQL
type PermissionChecker interface {
    Can(ctx context.Context, userID, tenantID string, perm Permission) (bool, error)
}

// Implémentation v1 : lit depuis membership.role (déjà en DB)
type MembershipBasedChecker struct {
    memberRepo membership.Repository
}

func (c *MembershipBasedChecker) Can(ctx context.Context, userID, tenantID string, perm Permission) (bool, error) {
    member, err := c.memberRepo.GetByUserAndTenant(ctx, userID, tenantID)
    if err != nil { return false, err }
    return roleHasPermission(member.Role, perm), nil
}
```

**Phase v2 :** Ajout des tables `roles`, `role_permissions`, `user_roles` quand un SaaS consommateur a un besoin réel de rôles personnalisés.

**Critère de déclenchement v2 :** au moins un SaaS consommateur soumit une issue GitHub avec un cas d'usage concret de rôle personnalisé.

Cette approche suit le principe **YAGNI** (You Aren't Gonna Need It) et évite de livrer de la complexité préventive.

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0003 : Multi-Tenancy
- RFC-0004 : Platform Membership (rôles v1)
- [NIST RBAC Standard](https://csrc.nist.gov/projects/role-based-access-control)
