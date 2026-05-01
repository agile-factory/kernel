# _ADR #0010:_ Pagination Uniforme `{ data, total, limit, offset }` sur les Endpoints de Liste

**date:** `2026 May 01`

---

## **Status:** `ACCEPTED`

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

| Endpoint | Stratégie | Justification |
|---|---|---|
| `GET /audit-logs` | SQL `LIMIT/OFFSET` (repo) | Cardinalité haute, croissance illimitée |
| `GET /memberships/tenant/{id}` | In-memory slice | Cardinalité typiquement \<100 |
| `GET /memberships/user/{id}` | In-memory slice | Cardinalité typiquement \<20 |

Les modules futurs (billing, notifications) **doivent implémenter la pagination SQL au niveau du repository dès le départ**.

---

## **Alternatives Considered**

### Cursor-based pagination (`after=<cursor>`)
- ✅ Plus performant pour les grandes tables (pas de COUNT)
- ✅ Stable aux insertions (offset peut décaler les pages)
- ❌ Plus complexe à implémenter et à utiliser côté client
- ❌ Incompatible avec les sauts de page directs
- **Verdict : différé** — à adopter pour les endpoints à très haute cardinalité (>10k lignes)

### Keyset pagination (`WHERE id > :last_id`)
- ✅ Performances constantes, pas de dégradation avec l'offset
- ❌ Nécessite un ordre total stable (UUID v7 ou `created_at + id`)
- **Verdict : différé** — recommandé pour la v2 des audit_logs

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
- `COUNT(*)` implicite dans `total` peut être coûteux sur de grandes tables (atténué par les index)
- La pagination in-memory charge toujours l'ensemble des données (acceptable pour cardinalités \<100)
- `OFFSET` dégrade les performances au-delà de ~10k lignes (plan prévu : cursor-based en v2)
