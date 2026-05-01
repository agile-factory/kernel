# _ADR #0010:_ Pagination Uniforme `{ data, total, limit, offset }` sur les Endpoints de Liste

**date:** `2026 May 01` — **mis à jour :** `2026 May 01`

---

## **Status:** `ACCEPTED` — v2 cursor `IMPLEMENTED`

---

## **Context**

Plusieurs endpoints retournent des collections : membres d'un tenant, memberships d'un utilisateur, logs d'audit. Sans pagination, ces endpoints peuvent retourner des payloads non bornés, impactant la performance et l'expérience client.

Le projet utilise deux contextes différents pour les listes :
1. **Listes à faible cardinalité** (`memberships` par tenant/user) : typiquement \<100 éléments — un fetch complet en mémoire reste acceptable
2. **Listes à haute cardinalité** (`audit_logs`) : potentiellement des milliers de lignes — la pagination SQL est impérative

---

## **Decision**

**Toute réponse de liste** utilise une enveloppe JSON uniforme :

```json
{
  "data":   [...],
  "total":  42,
  "limit":  50,
  "offset": 0
}
```

**Paramètres de requête standard** : `?limit=N&offset=N`
- Défaut : `limit=50`, `offset=0`
- Maximum : `limit=200` (cappé côté serveur)

**Deux stratégies selon la cardinalité :**

| Endpoint | Stratégie v1 | Stratégie v2 | Justification |
|---|---|---|---|
| `GET /api/v1/audit-logs` | SQL `LIMIT/OFFSET` (repo) | — | v1 maintenue |
| `GET /api/v2/audit-logs` | — | **Cursor keyset** | Cardinalité haute, croissance illimitée |
| `GET /api/v1/tenants/{id}/audit-logs` | SQL `LIMIT/OFFSET` | — | v1 maintenue |
| `GET /api/v2/tenants/{id}/audit-logs` | — | **Cursor keyset** | Cardinalité haute, croissance illimitée |
| `GET /memberships/tenant/{id}` | In-memory slice | — | Cardinalité typiquement <100 |
| `GET /memberships/user/{id}` | In-memory slice | — | Cardinalité typiquement <20 |

Les modules futurs (billing, notifications) **doivent implémenter la pagination SQL au niveau du repository dès le départ**.

---

## **Alternatives Considered**

### Cursor-based pagination (`after=<cursor>`)
- ✅ Plus performant pour les grandes tables (pas de COUNT)
- ✅ Stable aux insertions (offset peut décaler les pages)
- ❌ Plus complexe à implémenter et à utiliser côté client
- ❌ Incompatible avec les sauts de page directs
- **Verdict : implémenté en v2** — routes `/api/v2/audit-logs` et `/api/v2/tenants/{id}/audit-logs`.
  Curseur opaque `base64url(json{occurred_at, id})`, keyset SQL sur `(occurred_at DESC, id ASC)`, `total=-1` sentinelle, technique `limit+1` pour `has_more`.

### Keyset pagination (`WHERE id > :last_id`)
- ✅ Performances constantes, pas de dégradation avec l'offset
- ❌ Nécessite un ordre total stable (UUID v7 ou `created_at + id`)
- **Verdict : implémenté en v2** — keyset composite `(occurred_at DESC, id ASC)` sur `audit_logs`, sans dépendance à UUID v7

### Retourner un tableau nu `[]` sans enveloppe
- ✅ Plus simple
- ❌ Impossible d'ajouter `total` ou métadonnées sans breaking change
- ❌ Incohérent avec les conventions API modernes (RFC 7807, JSON:API)
- **Verdict : rejeté**

---

## **Consequences**

### **Pros**
- Format de réponse uniforme pour tous les clients (mobile, web, CLI)
- `total` permet aux clients d'afficher des indicateurs de progression ("page 2 sur 10")
- Le cap à 200 protège contre les requêtes abusives sans authentification supplémentaire
- Compatible avec une migration future vers cursor-based (ajout de `next_cursor` sans breaking change)

### **Cons**
- `COUNT(*)` implicite dans `total` peut être coûteux sur de grandes tables (atténué par les index) — **éliminé en v2** (`total = -1`)
- La pagination in-memory charge toujours l'ensemble des données (acceptable pour cardinalités <100)
- `OFFSET` dégrade les performances au-delà de ~10k lignes — **résolu en v2** par keyset sur `(occurred_at DESC, id ASC)` + index composite
