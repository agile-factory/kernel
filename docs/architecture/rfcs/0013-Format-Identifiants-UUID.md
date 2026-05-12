# _RFC #0013:_ Format des Identifiants — UUID v4 vs UUID v7

**date:** `2026 May 01`

---

## **Status:** `DRAFT` — migration rétroactive décidée

---

## **Summary**

Ce RFC tranche la question ouverte du RFC-0002 sur le **format des identifiants** utilisés dans le socle Agile Factory.

Le choix entre UUID v4 (aléatoire) et UUID v7 (monotone, trié par temps) impacte les performances des index PostgreSQL, la lisibilité des logs et la capacité à trier les entités par ordre de création sans colonne `created_at` supplémentaire.

---

## **Motivation**

Tous les agrégats du socle utilisent des UUID comme clés primaires. Le format actuel (v4 via `github.com/google/uuid`) est **aléatoire**, ce qui cause :

1. **Fragmentation des index B-tree** : les UUID v4 insérés aléatoirement fragmentent l'index primaire PostgreSQL, dégradant les performances d'insertion et de lecture au-delà de ~1M lignes
2. **Tri impossible sans `created_at`** : pour trier des entités par création, une colonne `created_at` est nécessaire (overhead)
3. **Logs illisibles** : `550e8400-e29b-41d4-a716-446655440000` ne donne aucune information temporelle

UUID v7 (RFC 9562, 2024) est **monotone et trié par temps** : les UUIDs générés séquentiellement sont naturellement ordonnés, éliminant la fragmentation d'index.

---

## **Proposal**

### Adopter UUID v7 pour tous les nouveaux identifiants

**Format UUID v7** :
```
0190e5c0-xxxx-7xxx-xxxx-xxxxxxxxxxxx
└──────┘              
timestamp (48 bits) → tri chronologique natif
```

**Génération en Go** (via `github.com/google/uuid` v1.6+) :

```go
id, err := uuid.NewV7()
// ou
id := uuid.Must(uuid.NewV7()).String()
```

### Règles d'application

| Contexte | Format |
|---|---|
| Nouvelles tables (dès ce RFC) | UUID v7 |
| Tables existantes (audit_logs, memberships, users, tenants, etc.) | **Migration rétroactive vers UUID v7** (voir §Migration) |
| IDs générés côté client (import, migration externe) | UUID v4 accepté (validation format uniquement) |

### Migration rétroactive des tables existantes

La migration s'applique à **toutes les tables existantes**, table par table dans l'ordre des dépendances FK.

#### Étape 1 — Génération des nouveaux UUIDs v7

Une migration `golang-migrate` ajoute une colonne temporaire `new_id UUID` sur chaque table. Les UUID v7 sont générés **côté Go** via `uuid.NewV7()` (pas de fonction native PostgreSQL pour UUID v7 encore disponible).

#### Étape 2 — Mise à jour des clés étrangères

Pour chaque table, les FK référençant l'ancienne PK sont mises à jour avant de swapper la PK :

```sql
-- Exemple pour memberships.user_id → users.id
UPDATE memberships m
   SET user_id = u.new_id
  FROM users u
 WHERE m.user_id = u.id;
```

#### Étape 3 — Swap de la PK et CLUSTER

```sql
-- Swap : new_id devient la nouvelle PK
ALTER TABLE users DROP CONSTRAINT users_pkey;
ALTER TABLE users RENAME COLUMN id TO old_id;
ALTER TABLE users RENAME COLUMN new_id TO id;
ALTER TABLE users ADD PRIMARY KEY (id);

-- Réorganise physiquement les pages selon l'ordre UUID v7
CLUSTER users USING users_pkey;
ANALYZE users;
```

#### Étape 4 — Nettoyage

```sql
ALTER TABLE users DROP COLUMN old_id;
```

**Ordre de migration (dépendances FK) :**

1. `users` (racine)
2. `tenants`
3. `memberships` (FK → users + tenants)
4. `audit_logs` (FK → users + tenants)
5. `subscriptions` (FK → tenants)
6. `tenant_customers` / `tenant_invoices` (FK → tenants)

**Note JWT :** après la migration, les JWTs existants contiennent l'ancien `user_id` (UUID v4). Faire une rotation du secret JWT ou incrémenter `token_version` pour invalider les sessions actives.

---

## **Drawbacks**

### Dépendance à `uuid` v1.6+
UUID v7 nécessite la version 1.6+ du package `github.com/google/uuid`. Vérifier la compatibilité avec les autres dépendances.

### Pas de tri garanti à la milliseconde
UUID v7 est trié à la milliseconde. Si plusieurs UUIDs sont générés dans la même milliseconde, leur ordre relatif dépend des bits aléatoires restants (ordre non garanti au sein d'une ms).

### Révélation partielle du timestamp
UUID v7 expose le timestamp de création dans l'identifiant. Pour des entités sensibles, cela peut révéler "quand" la ressource a été créée. Mitigation : chiffrement de l'ID en externe ou utilisation d'un alias public.

---

## **Alternatives**

### UUID v4 (status quo)
- ✅ Déjà en place, aucun changement nécessaire
- ❌ Fragmentation d'index croissante avec le volume
- **Verdict : abandonné** — migration rétroactive vers UUID v7 décidée pour toutes les tables

### ULID (Universally Unique Lexicographically Sortable Identifier)
- ✅ Trié lexicographiquement, human-friendly
- ❌ Pas un standard IETF, format non-UUID
- ❌ Dépendance externe supplémentaire
- **Verdict : rejeté** — UUID v7 est désormais standardisé (RFC 9562)

### KSUID (K-Sortable Unique Identifier)
- ✅ Trié par temps, encodage base62 compact
- ❌ Non standard, bibliothèque externe
- **Verdict : rejeté**

### Entiers auto-incrémentés (`BIGSERIAL`)
- ✅ Index parfaits, tri natif
- ❌ Prévisibles (énumération possible)
- ❌ Problèmes de sharding/distribution
- **Verdict : rejeté** pour les PKs publiques

---

## **Open Questions**

1. Faut-il exposer l'UUID v7 directement dans l'API ou utiliser un identifiant court pour les URLs (ex: `nanoid`) ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique (question ouverte #4)
- [RFC 9562 — Universally Unique IDentifiers (UUID)](https://www.rfc-editor.org/rfc/rfc9562)
- [UUID v7 Performance in PostgreSQL](https://www.postgresql.org/message-id/flat/CAAhFRxitJv%3DyoGnXUgeLB_O%2BM7J2BJAmb5jqAT9gZ3bij3uLUA%40mail.gmail.com)
- [github.com/google/uuid v1.6](https://pkg.go.dev/github.com/google/uuid)
