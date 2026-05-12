# _RFC #0006:_ Stratégie de Pagination des APIs de Liste

**date:** `2026 May 01` — **mis à jour :** `2026 May 01`

---

## **Status:** `ACCEPTED` — v2 `IMPLEMENTED`

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

| Endpoint | Cardinalité | Stratégie v1 | Stratégie v2 |
|---|---|---|---|
| `GET /memberships/tenant/{id}` | Faible (<200) | In-memory | — (cardinalité bornée) |
| `GET /memberships/user/{id}` | Très faible (<20) | In-memory | — (cardinalité bornée) |
| `GET /tenants/{id}/audit-logs` | Haute (illimitée) | SQL `LIMIT/OFFSET` | **Cursor-based (keyset)** |
| `GET /audit-logs` | Haute (illimitée) | SQL `LIMIT/OFFSET` | **Cursor-based (keyset)** |

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

### 4. Cursor-Based (implémenté en v2)

Pour les tables à très haute cardinalité (>10k lignes), `OFFSET` dégrade les performances (full scan jusqu'à l'offset). La pagination cursor-based est **implémentée en v2** sur les deux endpoints d'audit (`/api/v2/...`) :

```json
{
    "data": [...],
    "total":       -1,
    "limit":       50,
    "offset":      0,
    "next_cursor": "eyJvY2N1cnJlZF9hdCI6Ii4uLiIsImlkIjoiLi4uIn0=",
    "has_more":    true
}
```

**Mécanique :**
- Le curseur encode `{ occurred_at, id }` en Base64url (opaque pour les clients).
- Le keyset SQL utilise `WHERE (occurred_at, id) < ($N::timestamptz, $M::uuid)` avec un index composite `(tenant_id, occurred_at DESC, id ASC)`.
- `total = -1` est une valeur sentinelle signalant le mode cursor (pas de `COUNT(*)`).
- La technique `limit+1` détecte `has_more` sans requête supplémentaire.
- Les routes v1 (`/api/v1/...`) restent intactes — pas de breaking change.

**Routes exposées :**
```
GET /api/v2/tenants/{tenantId}/audit-logs?limit=50&after=<cursor>
GET /api/v2/audit-logs?limit=50&after=<cursor>
```

L'enveloppe v1 reste **compatible** : `next_cursor` et `has_more` ont été ajoutés sans retirer `offset` ni `total`.

---

## **Drawbacks**

### `COUNT(*)` coûteux sur grandes tables (v1 uniquement)
Pour retourner `total`, une requête `COUNT(*)` est nécessaire en v1. Résolu en v2 : `total = -1`, technique `limit+1` pour `has_more`.

### `OFFSET` dégradant (v1 uniquement)
Au-delà de ~10k lignes, PostgreSQL scanne et ignore `OFFSET` lignes. **Résolu en v2** par keyset sur `(occurred_at DESC, id ASC)` avec index composite.

### Pagination in-memory charge toute la collection
La stratégie B charge toujours l'ensemble des données en mémoire pour ensuite les slicer. Acceptable tant que la cardinalité est connue comme faible et bornée.

---

## **Alternatives**

### Cursor-based dès la v1
- ✅ Performances constantes
- ✅ Stable aux insertions concurrentes (offset peut décaler les pages)
- ❌ Complexité d'implémentation plus élevée
- ❌ Pas de navigation directe vers une page arbitraire
- **Verdict : différé en v1, implémenté en v2** — routes `/api/v2/audit-logs` et `/api/v2/tenants/{id}/audit-logs`

### Keyset pagination (`WHERE id > :last_id ORDER BY id`)
- ✅ Plus simple que cursor-based, performances constantes
- ❌ Nécessite un ordre total stable (UUID v7 ou composite `created_at + id`)
- **Verdict : implémenté en v2** — keyset composite `(occurred_at DESC, id ASC)` sans dépendance à UUID v7 (RFC-0013 encore `PROPOSED`)

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
3. ~~Quand migrer `audit_logs` vers cursor-based ?~~ **Résolu** : implémenté en v2 sur `/api/v2/audit-logs`.
4. Faut-il ajouter des filtres (`?action=`, `?actor_id=`) en v2 des endpoints cursor-based ? Différé à v2.1.

---

## **References**

- RFC-0002 : Architecture & Stack Technique
- RFC-0004 : Platform Membership (utilise pagination in-memory)
- RFC-0005 : Platform Audit (utilise pagination SQL)
- ADR-0010 : Pagination Uniforme `{ data, total, limit, offset }`
- [JSON:API Specification — Pagination](https://jsonapi.org/format/#fetching-pagination)
- [Stripe API Pagination](https://stripe.com/docs/api/pagination)
