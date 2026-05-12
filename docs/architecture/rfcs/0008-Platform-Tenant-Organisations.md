# _RFC #0008:_ Platform `tenant` — Gestion des Organisations

**date:** `2026 May 01`

> ⚠️ **Partiellement supersédé par [RFC-0018 — Amendements B2B Onboarding](0018-Amendements-B2B-Onboarding.md)** (2026-05-12) : `POST /api/v1/tenants` (création publique) est supprimé et remplacé par le flux `signup-owner` atomique (RFC-0018) et `POST /admin/v1/tenants` (super-admin). L'Open Question #2 ("qui peut créer un tenant ?") est tranchée dans RFC-0018. Ce RFC reste valide pour l'historique et le modèle de domaine Tenant.

---

## **Status:** `ACCEPTED` — création publique **SUPERSEDED BY RFC-0018**

---

## **Summary**

Ce RFC définit la plateforme `tenant`, brique SaaS fondamentale qui représente une **organisation cliente** dans le socle Agile Factory.

Un tenant est l'unité d'isolation principale du système. Toutes les données métier sont partitionnées par `tenant_id`. La plateforme `tenant` gère la création, la recherche et l'état des tenants.

---

## **Motivation**

Dans une architecture multi-tenant shared schema, le tenant est la brique la plus fondamentale. Il doit être :
- **Adressable de manière unique** par un slug normalisé (ex: `acme-corp`)
- **Activable/désactivable** pour gérer les abonnements et suspensions
- **Référencé dans toutes les tables métier** via `tenant_id`

Sans plateforme dédiée, la création de tenants serait dispersée dans les handlers sans validation de slug, sans contrainte d'unicité au niveau applicatif, et sans base pour les futures extensions (settings, billing, branding).

---

## **Proposal**

### 1. Modèle de domaine

```go
type Tenant struct {
    ID        string
    Name      string
    Slug      Slug    // value object normalisé (lowercase, kebab-case)
    IsActive  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

Le `Slug` est un value object qui :
- Convertit en minuscules
- Remplace les espaces par des tirets
- Rejette les caractères non alphanumériques (sauf tirets)
- Impose une longueur minimale de 2 caractères

### 2. Use Cases

#### `CreateTenantUseCase`

```
1. Valider et normaliser le slug
2. Vérifier l'unicité du slug
3. Créer l'aggregate Tenant
4. Persister
5. Émettre un log d'audit TENANT_CREATED
```

#### `FindTenantUseCase`

```
FindByID(ctx, tenantID) → Tenant | ErrTenantNotFound
```

Utilisé par les autres use cases (`RegisterUseCase`, `InviteMemberUseCase`) pour vérifier l'existence et l'état actif d'un tenant.

### 3. API HTTP

```
POST /api/v1/tenants
Body: { "id": "uuid", "name": "Acme Corp", "slug": "acme-corp" }
→ 201 Created { id, name, slug, isActive, createdAt }

GET /api/v1/tenants/{id}
→ 200 OK { id, name, slug, isActive, createdAt }
```

### 4. Schéma SQL

```sql
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(100) NOT NULL UNIQUE,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tenants_slug ON tenants(slug);
```

### 5. Extensibilité prévue

La table `tenants` est intentionnellement minimale en v1. Les extensions prévues (RFC-0003) :

| Table | Contenu | Phase |
|---|---|---|
| `tenant_settings` | Logo, couleurs, langue, timezone, domaine custom | v1.1 |
| `subscriptions` | Plan actif, date de début, date de fin | v2 |
| `usage_quotas` | Compteurs, limites par plan | v2 |

---

## **Drawbacks**

### Pas de désactivation douce
En v1, `is_active = false` désactive le tenant mais ne prévient pas les utilisateurs connectés (les tokens JWT existants restent valides jusqu'à expiration). Une invalidation active nécessiterait une blacklist de tokens.

### Pas de suppression (soft-delete uniquement)
La suppression physique d'un tenant CASCADE sur toutes les données. La v1 ne propose pas de suppression : désactivation seulement. Un mécanisme de GDPR-delete est prévu ultérieurement.

---

## **Alternatives**

### Slug auto-généré depuis le nom
- ✅ Moins de friction à la création
- ❌ Collisions possibles si deux organisations ont le même nom
- **Verdict : disponible comme helper mais non forcé**

### Tenant identifié par subdomain uniquement (pas de slug)
- ✅ URL propres dès le départ (`acme.app.com`)
- ❌ Résolution du tenant par subdomain nécessite un middleware DNS-aware
- **Verdict : différé** — custom domain prévu dans `tenant_settings`

### Tenant créé automatiquement lors du premier `register`
- ✅ Onboarding zero-friction
- ❌ Nom/slug du tenant non maîtrisé
- ❌ Découple la création de tenant du processus d'onboarding
- **Verdict : différé** — à évaluer pour les SaaS B2C

---

## **Open Questions**

1. Faut-il un endpoint `PATCH /tenants/{id}` dès la v1 pour renommer/désactiver ?
2. Qui peut créer un tenant : n'importe quel utilisateur, ou un super-admin seulement ?
3. Limite du nombre de tenants par utilisateur sur le plan Free ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0003 : Multi-Tenancy Architecture
- RFC-0004 : Platform Membership (membership créé à l'inscription dans un tenant)
- RFC-0005 : Platform Audit (audit TENANT_CREATED)
- ADR-0003 : Multi-Tenancy Shared Schema
