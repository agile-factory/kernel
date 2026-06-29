# _ADR #0016:_ Platform `end_customers` — B2B2C Foundation (RFC-0033)

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0033 définit le module `platform/end_customers` : les consommateurs du portail customer-facing d'un tenant (B2B2C). Distinct de `auth.User` (membres workspace). Ce module est la fondation de toute la surface B2B2C — `customer_auth`, `white_label`, `customer_notifications` en dépendent.

Implémenté sur `feat/p1-v0.2.0` en 2026-06-28.

---

## **Decision**

### Architecture

Structure standard `platform/` du kernel :
- `domain/` — entité `EndCustomer`, port `EndCustomerRepository`, erreurs domain
- `infrastructure/` — `PostgresEndCustomerRepository` + `query.go`
- `usecase/` — un fichier par use case : `create`, `get`, `list`, `update`, `delete`, `suspend`, `restore`
- `delivery/http/` — `EndCustomerHandler` + routes
- `end_customers.go` — `Module` / `Config` + `RegisterRoutes`
- `migrations/` — `0001_create_end_customers` + `0002_create_end_customer_tags`

### `execWithOptionalTx` — nil-pool pattern

Les use cases write passent par `execWithOptionalTx(ctx, db, fn)` au lieu d'appeler `txctx.ExecInTx` directement. Si `db` est nil (mode test avec stubs en mémoire), `fn` est appelée sans transaction. Cela permet aux tests unitaires de fonctionner sans connexion Postgres tout en garantissant l'atomicité en production.

```go
func execWithOptionalTx(ctx context.Context, db *pgxpool.Pool, fn func(context.Context) error) error {
    if db == nil {
        return fn(ctx)
    }
    return txctx.ExecInTx(ctx, db, fn)
}
```

`end_customers.Config.DB` est optionnel (nil en tests, pool en production).

### Tags — table séparée, pas JSONB

Les tags end-customer sont stockés dans `end_customer_tags(end_customer_id, tag)` plutôt qu'en colonne JSONB. Raison : index B-tree efficace pour filtrage par tag, contrainte de cardinalité (`UNIQUE(end_customer_id, tag)`), pas de surcharge applicative pour parser/sérialiser JSONB.

### Index GIN partiel pour full-text search

```sql
CREATE INDEX idx_end_customers_search ON end_customers
    USING gin(to_tsvector('simple', first_name || ' ' || last_name || ' ' || COALESCE(email, '') || ' ' || COALESCE(external_id, '')))
    WHERE deleted_at IS NULL;
```

Partiel `WHERE deleted_at IS NULL` : les clients supprimés (soft-delete) ne participent pas à la recherche, ce qui réduit la taille de l'index de manière significative dans les tenants actifs depuis longtemps.

### Metadata validation

Validée dans le use case (pas en base) : max 20 clés, valeur max 256 chars, clés non vides. Stockée en JSONB avec contrainte de vérification `jsonb_typeof(metadata) = 'object'`.

### Statuts

`ACTIVE → SUSPENDED → ACTIVE` (restore), `ACTIVE → DELETED` (soft-delete `deleted_at`). Pas d'état `DELETED` comme valeur de `status` — le soft-delete est géré par la colonne `deleted_at` séparément, pour conserver `status` à sa dernière valeur opérationnelle.

### Permissions

Cinq constantes ajoutées dans `kernel/permission/permission.go` :
`end_customer:create`, `end_customer:list`, `end_customer:read`, `end_customer:update`, `end_customer:delete`, `end_customer:suspend`.

---

## **Alternatives rejetées**

### Intégrer end_customers dans `platform/user`

- ✅ Moins de packages
- ❌ Confusion conceptuelle : `auth.User` est un membre workspace, `EndCustomer` est un consommateur du portail. Cycle de vie, permissions et JWT distincts.
- **Verdict : rejeté** — séparation stricte des contextes bornés

### Soft-delete via colonne `status = 'DELETED'`

- ✅ Schéma plus simple
- ❌ On perd la dernière valeur de `status` opérationnel (était `ACTIVE` ou `SUSPENDED` avant la suppression ?)
- **Verdict : rejeté** — `deleted_at TIMESTAMPTZ` séparé, `status` reflète l'état business

### Tags en colonne `TEXT[]`

- ✅ Simple
- ❌ Pas d'index efficace par tag, pas de contrainte d'unicité par (client, tag) nativement
- **Verdict : rejeté** — table de jointure `end_customer_tags`

---

## **Conséquences**

- `platform/customer_auth` peut maintenant être implémenté (dépend de `end_customers.FindByID`)
- `platform/customer_notifications` peut intégrer les templates email par end_customer
- Pattern `execWithOptionalTx` réutilisable dans tous les modules write futurs
- Les tests sont 100% in-memory (no Postgres) : CI rapide et stable
