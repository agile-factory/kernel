# _ADR #0003:_ Multi-Tenancy Shared Schema avec Table `memberships`

**date:** `2026 May 01`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

Agile Factory doit supporter plusieurs tenants (organisations) dans une même base de données. La question centrale est : comment modéliser la relation entre un utilisateur et un tenant ?

La première implémentation utilisait un `tenant_id` direct sur la table `users` (relation 1:1). Ce modèle s'est avéré insuffisant : un utilisateur doit pouvoir appartenir à **plusieurs tenants** avec des rôles différents dans chacun.

---

## **Decision**

Adopter un modèle **Many-to-Many** via une table de jonction `memberships` :

```sql
memberships (
    id          UUID PRIMARY KEY,
    user_id     UUID REFERENCES users(id),
    tenant_id   UUID REFERENCES tenants(id),
    role        member_role  -- ROLE_ADMIN | ROLE_AGENT | ROLE_VIEWER
    status      member_status -- PENDING | ACTIVE | REVOKED
    invited_by  UUID REFERENCES users(id),
    joined_at   TIMESTAMPTZ,
    UNIQUE (user_id, tenant_id)
)
```

La stratégie est **Shared Database, Shared Schema** : toutes les tables métier futures porteront un `tenant_id` avec un index composé `(tenant_id, created_at)`.

Le cycle de vie d'un membership suit une **machine à états** :
- `PENDING` → invitation envoyée, pas encore acceptée
- `ACTIVE` → accès accordé
- `REVOKED` → accès retiré

---

## **Alternatives Considered**

### 1:1 `tenant_id` sur `users`
- ✅ Simple à implémenter
- ❌ Un utilisateur ne peut pas appartenir à plusieurs tenants
- ❌ Inflexible pour les cas multi-organisation
- **Verdict : rejeté et migré**

### Database par tenant
- ✅ Isolation physique totale (conformité, sécurité)
- ❌ Coût opérationnel élevé : N bases à maintenir
- ❌ Connection pool explose au-delà de 50 tenants
- ❌ Onboarding = provisioning base de données
- **Verdict : différé** — reconsidéré si un client exige l'isolation physique

### Schema par tenant (même DB, schémas séparés)
- ✅ Isolation logique sans coût infra
- ❌ Gestion des migrations complexe (N schémas à migrer)
- ❌ Support partiel selon les drivers
- **Verdict : rejeté**

---

## **Consequences**

### **Pros**
- Un utilisateur peut être admin d'un tenant et agent d'un autre
- Flux d'invitation natif (PENDING → ACTIVE → REVOKED)
- Audit du cycle de vie complet de chaque membership
- Extensible : on peut ajouter des permissions fines par membership sans changer le schéma utilisateur

### **Cons**
- Toute table métier future doit systématiquement porter `tenant_id`
- Un oubli de filtre `tenant_id` en infrastructure pourrait exposer des données cross-tenant (atténué par des tests d'isolation et possibilité d'activer Postgres Row-Level Security)
- Migrations plus complexes lors de l'ajout de contraintes multi-tenant à des tables existantes
