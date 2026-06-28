# _RFC #0033:_ Platform `end_customers` — Gestion des Clients Finaux B2B2C

**date:** `2026 Jun 28`

---

## **Status:** `ACCEPTED`

> **Review Aziz** : architecture cohérente avec les modules existants. Pattern Config/Module/New/RegisterRoutes respecté. `UNIQUE NULLS NOT DISTINCT` valide sur PG 16. Dépendances P1bis bien ordonnées.
> **Review Lucas** : interfaces mockables, domaine sans couplage infra. Point à corriger en implémentation : validation `metadata` (max 20 clés, max 256 chars par valeur) doit être explicitement enforced dans `CreateEndCustomerUseCase` et `UpdateEndCustomerUseCase`, pas seulement documentée.

---

## **Summary**

Ce RFC définit le module `platform/end_customers`, brique fondatrice du profil **B2B2C**.

Un **end-customer** est un client final géré par un tenant SaaS (ex : locataire d'une agence immobilière, patient d'une clinique, abonné d'un opérateur). Ce concept est distinct des `users` (membres du workspace, RFC-0007) : les end-customers n'ont pas accès au back-office du tenant — ils interagissent via un portail ou une app customer-facing.

Ce module fournit :
- Le modèle domaine `EndCustomer`
- CRUD complet scoped au tenant (avec pagination curseur RFC-0006)
- Système de tags pour la segmentation
- Événements outbox pour les hooks B2B2C downstream (facturation, notifications, auth)
- API surface `/customer/v1/` distincte de `/api/v1/`

---

## **Motivation**

### Problème 1 — Absence de représentation du client final

En B2B pur (v0.1.0-v0.2.0), seul le `User` (membre du workspace) existe. Pour les SaaS B2B2C (immobilier, santé, microfinance, distribution CEDEAO), le tenant doit gérer une base de clients finaux qui :

- ne sont pas des membres du workspace
- ont un cycle de vie propre (création, suspension, archivage)
- déclenchent des actions downstream (facturation Merchant Mode, notifications, auth portail)

### Problème 2 — Couplage billing Merchant Mode

RFC-0020 §Merchant Mode référence `end_customers(id)` via FK dans `tenant_invoices`. Ce module est un prérequis bloquant pour le Merchant Mode B2B2C.

### Problème 3 — Segmentation et recherche

Les tenants B2B2C ont besoin de segmenter leurs clients (tags, statut, date de création) pour les workflows de notification batch et de facturation récurrente.

---

## **Proposal**

### 1. Modèle domaine

```go
// platform/end_customers/domain/end_customer.go
package domain

import (
    "time"

    "github.com/agile-factory/kernel/kernel/errors"
)

type EndCustomerStatus string

const (
    StatusActive    EndCustomerStatus = "ACTIVE"
    StatusSuspended EndCustomerStatus = "SUSPENDED"
    StatusDeleted   EndCustomerStatus = "DELETED"
)

type EndCustomer struct {
    ID         string
    TenantID   string
    ExternalID string    // identifiant métier optionnel (numéro client tenant)
    Email      string
    Phone      string    // E.164 (+221XXXXXXXXX)
    FirstName  string
    LastName   string
    Status     EndCustomerStatus
    Tags       []string
    Metadata   map[string]string // champs libres, max 20 clés × 256 chars
    CreatedAt  time.Time
    UpdatedAt  time.Time
    DeletedAt  *time.Time
}

func (ec *EndCustomer) FullName() string {
    return ec.FirstName + " " + ec.LastName
}

func (ec *EndCustomer) Suspend() error {
    if ec.Status != StatusActive {
        return errors.InvalidInput("end_customer must be ACTIVE to be suspended")
    }
    ec.Status = StatusSuspended
    return nil
}

func (ec *EndCustomer) Restore() error {
    if ec.Status != StatusSuspended {
        return errors.InvalidInput("end_customer must be SUSPENDED to be restored")
    }
    ec.Status = StatusActive
    return nil
}
```

### 2. Port Repository

```go
// platform/end_customers/domain/repository.go
package domain

import "context"

type ListParams struct {
    TenantID string
    Cursor   string
    Limit    int
    Status   *EndCustomerStatus
    Tag      string
    Search   string // full-text sur email + nom
}

type EndCustomerRepository interface {
    Create(ctx context.Context, ec *EndCustomer) error
    FindByID(ctx context.Context, tenantID, id string) (*EndCustomer, error)
    FindByEmail(ctx context.Context, tenantID, email string) (*EndCustomer, error)
    FindByExternalID(ctx context.Context, tenantID, externalID string) (*EndCustomer, error)
    Update(ctx context.Context, ec *EndCustomer) error
    SoftDelete(ctx context.Context, tenantID, id string) error
    List(ctx context.Context, params ListParams) ([]*EndCustomer, string, error) // ([]*EC, nextCursor, err)
    Count(ctx context.Context, tenantID string, status *EndCustomerStatus) (int64, error)
}
```

### 3. Use cases

#### `CreateEndCustomerUseCase`

```
1. Vérifier permission : caller est OWNER ou ADMIN du tenant
2. Valider : email E.164 si phone fourni, email format, firstName/lastName non vides
3. Unicité (email, tenant_id) — erreur CONFLICT si déjà existant
4. Unicité (external_id, tenant_id) si external_id fourni
5. BEGIN TRANSACTION
   5a. INSERT end_customers
   5b. INSERT event_outbox {type: "end_customer.created", payload: {id, tenant_id, email}}
6. COMMIT
7. Retourner EndCustomer créé
```

#### `UpdateEndCustomerUseCase`

```
1. Permission : OWNER ou ADMIN
2. Charger EndCustomer (not deleted)
3. Appliquer les champs fournis (partial update)
4. Unicité email si email modifié
5. UPDATE + event_outbox {type: "end_customer.updated"}
```

#### `SuspendEndCustomerUseCase` / `RestoreEndCustomerUseCase`

```
1. Permission : OWNER ou ADMIN
2. Charger, appeler ec.Suspend() / ec.Restore()
3. UPDATE + event_outbox {type: "end_customer.suspended" | "end_customer.restored"}
```

#### `DeleteEndCustomerUseCase`

```
1. Permission : OWNER seulement (suppression irréversible côté API)
2. Soft-delete : deleted_at = NOW(), status = DELETED
3. event_outbox {type: "end_customer.deleted"}
Note RGPD : hard-delete via super-admin endpoint dédié (future RFC-0027)
```

#### `ListEndCustomersUseCase`

```
1. Permission : OWNER, ADMIN, AGENT (read-only)
2. Appliquer filtres (status, tag, search)
3. Cursor-pagination (RFC-0006) sur (created_at DESC, id)
4. Retourner ([]*EndCustomer, nextCursor)
```

### 4. Schéma SQL

```sql
-- platform/end_customers/migrations/0001_create_end_customers.up.sql

CREATE TABLE end_customers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    external_id  VARCHAR(255),
    email        VARCHAR(255),
    phone        VARCHAR(20),
    first_name   VARCHAR(100) NOT NULL,
    last_name    VARCHAR(100) NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'ACTIVE'
                     CHECK (status IN ('ACTIVE','SUSPENDED','DELETED')),
    metadata     JSONB        NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,

    CONSTRAINT uq_end_customer_email_tenant
        UNIQUE NULLS NOT DISTINCT (tenant_id, email),
    CONSTRAINT uq_end_customer_external_id_tenant
        UNIQUE NULLS NOT DISTINCT (tenant_id, external_id)
);

CREATE INDEX idx_end_customers_tenant_status
    ON end_customers(tenant_id, status)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_end_customers_cursor
    ON end_customers(tenant_id, created_at DESC, id)
    WHERE deleted_at IS NULL;

-- Recherche full-text (email + nom)
CREATE INDEX idx_end_customers_search
    ON end_customers
    USING gin(to_tsvector('simple',
        coalesce(first_name,'') || ' ' ||
        coalesce(last_name,'')  || ' ' ||
        coalesce(email,'')));
```

```sql
-- platform/end_customers/migrations/0002_create_end_customer_tags.up.sql

CREATE TABLE end_customer_tags (
    end_customer_id UUID NOT NULL REFERENCES end_customers(id) ON DELETE CASCADE,
    tag             VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (end_customer_id, tag)
);

CREATE INDEX idx_end_customer_tags_tenant
    ON end_customer_tags(tag)
    INCLUDE (end_customer_id);
```

### 5. API HTTP

Surface montée via `RegisterRoutes(mux)` :

```
# Gestion (OWNER/ADMIN)
POST   /customer/v1/customers                    CreateEndCustomer
GET    /customer/v1/customers                    ListEndCustomers (cursor, filtres)
GET    /customer/v1/customers/{id}               GetEndCustomer
PATCH  /customer/v1/customers/{id}               UpdateEndCustomer
DELETE /customer/v1/customers/{id}               DeleteEndCustomer (soft)
POST   /customer/v1/customers/{id}/suspend       SuspendEndCustomer
POST   /customer/v1/customers/{id}/restore       RestoreEndCustomer
```

Middleware de route : `auth.Required` + `tenant.Inject` (même stack que `/api/v1/`).

### 6. Événements outbox émis

| Type | Payload | Consommateurs downstream |
|------|---------|--------------------------|
| `end_customer.created` | `{id, tenant_id, email, phone}` | customer_notifications (bienvenue), billing (init compte) |
| `end_customer.updated` | `{id, tenant_id, fields_changed}` | audit |
| `end_customer.suspended` | `{id, tenant_id}` | customer_notifications, billing (gel facturation) |
| `end_customer.restored` | `{id, tenant_id}` | customer_notifications |
| `end_customer.deleted` | `{id, tenant_id}` | billing (annuler factures draft), customer_auth (révoquer tokens) |

### 7. Module wiring

```go
// platform/end_customers/end_customers.go
package endcustomers

type Config struct {
    Pool        *pgxpool.Pool
    Repo        domain.EndCustomerRepository
    Checker     permission.Checker
    Publisher   events.Publisher
    AuditLogger audit.EventLogger
    Logger      *slog.Logger
}

type Module struct {
    CreateUC  *usecase.CreateEndCustomerUseCase
    UpdateUC  *usecase.UpdateEndCustomerUseCase
    ListUC    *usecase.ListEndCustomersUseCase
    SuspendUC *usecase.SuspendEndCustomerUseCase
    RestoreUC *usecase.RestoreEndCustomerUseCase
    DeleteUC  *usecase.DeleteEndCustomerUseCase
}

func New(cfg Config) *Module { ... }
func (m *Module) RegisterRoutes(mux *http.ServeMux) { ... }
```

---

## **Drawbacks**

1. **Données sensibles** : end-customers contiennent email + phone → obligation RGPD/CEDEAO de journaliser les accès et de supporter le hard-delete sur demande (RFC-0027 futur).
2. **JSONB metadata** : flexible mais pas indexable finement. Si les tenants ont besoin de filtrer sur les métadonnées, il faudra un schéma de champs custom (RFC futur).
3. **Email nullable** : un end-customer peut n'avoir que le phone (cas courant CEDEAO Mobile Money). La contrainte UNIQUE s'applique uniquement quand email IS NOT NULL — géré via `UNIQUE NULLS NOT DISTINCT`.

---

## **Alternatives**

### Réutiliser `platform/user` pour les end-customers

- ✅ Pas de nouveau module
- ❌ Mélange membres du workspace et clients finaux dans la même table
- ❌ Impossible d'avoir des permissions asymétriques (staff voit tout, customer voit seulement le sien)
- **Verdict : rejeté** — séparation nette entre staff et end-customers est un invariant B2B2C

### Tags via colonne ARRAY PostgreSQL plutôt qu'une table séparée

- ✅ Plus simple, une seule table
- ❌ Impossible d'indexer efficacement pour le filtre `WHERE tag = ?`
- **Verdict : table séparée** — perf de recherche par tag justifie la join

---

## **Plan d'adoption**

1. Écrire RFC-0034 (customer_auth), RFC-0035 (white_label), RFC-0036 (customer_notifications) — dépendants de ce module
2. Review Aziz + Lucas
3. Implémenter : migrations → domain → infra → usecase → delivery → tests
4. Intégrer dans `examples/saas-b2b/main.go` avec flag `B2B2C_ENABLED`
5. ADR post-implémentation

---

## **References**

- [RFC-0020 — Platform Billing Merchant Mode](0020-Platform-Billing-Hybride-Merchant-Mode.md) *(tenant_invoices.end_customer_id FK)*
- [RFC-0004 — Platform Membership](0004-Platform-Membership-Lifecycle.md) *(pattern invite-token, réutilisé pour optionnel end-customer invite)*
- [RFC-0006 — Stratégie Pagination](0006-Strategie-Pagination-APIs.md) *(cursor v2 obligatoire)*
- [RFC-0015 — Event Bus Outbox](0015-Event-Bus-Interne.md)
- [RFC-0034 — Platform customer_auth](0034-Platform-Customer-Auth.md) *(à créer)*
- [RFC-0036 — Platform customer_notifications](0036-Platform-Customer-Notifications.md) *(à créer)*
- [RFC-0038 — Conventions Package Réutilisable](0038-Conventions-Package-Reutilisable.md)
