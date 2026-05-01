# _RFC #0013:_ Format des Identifiants — UUID v4 vs UUID v7

**date:** `2026 May 01`

---

## **Status:** `PROPOSED`

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
| Nouvelles tables (à partir de ce RFC) | UUID v7 |
| Tables existantes (audit_logs, memberships, etc.) | UUID v4 maintenu (pas de migration rétroactive) |
| IDs générés côté client (import, migration) | UUID v4 accepté (validation format uniquement) |

### Migration des tables existantes

**Non recommandée en v1** : la migration des PKs existantes de v4 vers v7 est coûteuse et présente un risque élevé. Les tables existantes continuent avec UUID v4. Les nouvelles tables adoptent UUID v7 dès leur création.

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
- **Verdict : maintenu pour les tables existantes uniquement**

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
2. Le timestamp dans l'UUID v7 représente-t-il un risque RGPD pour certains types de données ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique (question ouverte #4)
- [RFC 9562 — Universally Unique IDentifiers (UUID)](https://www.rfc-editor.org/rfc/rfc9562)
- [UUID v7 Performance in PostgreSQL](https://www.postgresql.org/message-id/flat/CAAhFRxitJv%3DyoGnXUgeLB_O%2BM7J2BJAmb5jqAT9gZ3bij3uLUA%40mail.gmail.com)
- [github.com/google/uuid v1.6](https://pkg.go.dev/github.com/google/uuid)
