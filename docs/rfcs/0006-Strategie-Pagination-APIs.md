# _RFC #0006:_ Stratégie de Pagination des APIs de Liste

**date:** `2026 May 01`

---

## **Status:** `ACCEPTED`

---

## **Summary**

Ce RFC définit la **convention de pagination unifiée** pour tous les endpoints de liste du socle Agile Factory, actuels et futurs.

La stratégie repose sur :
1. Une **enveloppe JSON uniforme** : `{ data, total, limit, offset }`
2. Des **paramètres de requête standards** : `?limit=N&offset=N`
3. Une **distinction de stratégie** selon la cardinalité attendue : pagination SQL pour les collections non bornées, pagination in-memory pour les collections à faible cardinalité connue

---

## **Motivation**

Sans convention établie dès le départ :
- Chaque endpoint de liste retourne un format différent (`[]`, `{ items: [] }`, `{ results: [], count: N }`)
- Les clients (web, mobile) doivent gérer plusieurs formats de pagination
- L'ajout de métadonnées (total, next_cursor) devient un breaking change

Les endpoints existants concernés :

| Endpoint | Cardinalité | Stratégie v1 |
|---|---|---|
| `GET /memberships/tenant/{id}` | Faible (\<200) | In-memory |
| `GET /memberships/user/{id}` | Très faible (\<20) | In-memory |
| `GET /tenants/{id}/audit-logs` | Haute (illimitée) | SQL |

---

## **Proposal**

### 1. Enveloppe de réponse

Tout endpoint de liste retourne obligatoirement :

```json
{
    "data":   [ ... ],
    "total":  142,
    "limit":  50,
    "offset": 0
}
```

- `data` : tableau des éléments de la page courante
- `total` : nombre total d'éléments (toutes pages confondues)
- `limit` : nombre d'éléments demandés (peut être inférieur si fin de collection)
- `offset` : point de départ de la page

### 2. Paramètres de requête

```
GET /api/v1/resource?limit=50&offset=100
```

| Paramètre | Défaut | Maximum | Comportement si invalide |
|---|---|---|---|
| `limit` | `50` | `200` | Valeur par défaut |
| `offset` | `0` | — | Valeur par défaut |

Le serveur **cappe silencieusement** `limit` à 200. Il ne retourne pas d'erreur pour une valeur invalide (non-numérique) : il utilise le défaut.

### 3. Stratégies d'implémentation

#### Stratégie A : Pagination SQL (haute cardinalité)

Pour les collections potentiellement grandes (audit_logs, futures factures, notifications) :

```sql
SELECT ... FROM audit_logs
WHERE tenant_id = $1
ORDER BY occurred_at DESC
LIMIT $2 OFFSET $3
```

`total` est calculé en comptant avec `SELECT COUNT(*)` ou en utilisant la valeur retournée par la même requête sans LIMIT (selon le coût).

#### Stratégie B : Pagination in-memory (faible cardinalité connue)

Pour les collections connues comme petites (memberships par user, par exemple) :

```go
func applyPage(all []*domain.Membership, limit, offset int) []*domain.Membership {
    if offset >= len(all) { return nil }
    end := min(offset+limit, len(all))
    return all[offset:end]
}
```

`total` = `len(all)` (taille de la collection complète en mémoire).

### 4. Évolution vers Cursor-Based (prévue)

Pour les tables à très haute cardinalité (>10k lignes), `OFFSET` dégrade les performances (full scan jusqu'à l'offset). La migration vers cursor-based est planifiée en v2 :

```json
{
    "data": [...],
    "next_cursor": "eyJpZCI6IjEyMyIsIm9jY3VycmVkX2F0IjoiMjAyNiJ9",
    "has_more": true
}
```

L'enveloppe actuelle est **compatible** avec cette évolution (ajout de `next_cursor` sans retirer `offset`).

---

## **Drawbacks**

### `COUNT(*)` coûteux sur grandes tables
Pour retourner `total`, une requête `COUNT(*)` est nécessaire. Sur des tables de millions de lignes, cela peut être lent sans optimisations (index covering, approximate count via `pg_class.reltuples`).

### `OFFSET` dégradant
Au-delà de ~10k lignes, PostgreSQL doit scanner et ignorer `OFFSET` lignes. Planifié pour v2 : cursor-based pour les tables critiques.

### Pagination in-memory charge toute la collection
La stratégie B charge toujours l'ensemble des données en mémoire pour ensuite les slicer. Acceptable tant que la cardinalité est connue comme faible et bornée.

---

## **Alternatives**

### Cursor-based dès la v1
- ✅ Performances constantes
- ✅ Stable aux insertions concurrentes (offset peut décaler les pages)
- ❌ Complexité d'implémentation plus élevée
- ❌ Pas de navigation directe vers une page arbitraire
- **Verdict : différé** — introduit en v2 pour les collections à haute cardinalité

### Keyset pagination (`WHERE id > :last_id ORDER BY id`)
- ✅ Plus simple que cursor-based, performances constantes
- ❌ Nécessite un ordre total stable (UUID v7 ou composite `created_at + id`)
- **Verdict : différé** — recommandé si migration vers UUID v7 (voir Open Questions RFC-0002)

### Pas d'enveloppe (retourner un tableau nu `[]`)
- ✅ Réponse plus simple
- ❌ Impossible d'ajouter `total` ou métadonnées sans breaking change futur
- **Verdict : rejeté**

### Link Header (RFC 5988 - Web Linking)
- ✅ Standard HTTP pour la pagination
- ❌ Moins lisible que JSON pour les clients mobiles
- ❌ Nécessite parsing des headers
- **Verdict : rejeté** — JSON body préféré pour la cohérence

---

## **Open Questions**

1. Faut-il normaliser le nom `data` ou le laisser sémantique (`memberships`, `logs`) ? Actuellement `data` est utilisé pour l'uniformité.
2. Faut-il inclure `page` (numéro de page calculé) en plus de `offset` pour les clients qui pensent en pages ?
3. Quand migrer `audit_logs` vers cursor-based ? Seuil : 100k lignes en prod ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0004 : Platform Membership (utilise pagination in-memory)
- RFC-0005 : Platform Audit (utilise pagination SQL)
- ADR-0010 : Pagination Uniforme `{ data, total, limit, offset }`
- [JSON:API Specification — Pagination](https://jsonapi.org/format/#fetching-pagination)
- [Stripe API Pagination](https://stripe.com/docs/api/pagination)
