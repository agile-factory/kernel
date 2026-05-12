# _RFC #0014:_ Stratégie de Versioning de l'API HTTP

**date:** `2026 May 01`

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC tranche la question ouverte du RFC-0002 sur la **stratégie de versioning de l'API REST** du socle Agile Factory.

La décision retenue est : **versioning par URL** (`/api/v1/...`), avec migration vers `/api/v2/...` uniquement en cas de breaking change majeur. Le versioning par header (`Accept-Version: v1`) est écarté.

---

## **Motivation**

Un socle SaaS réutilisable est consommé par plusieurs clients (web, mobile, intégrations tierces). Sans stratégie de versioning :
- Un breaking change dans l'API casse silencieusement tous les clients
- Les clients mobiles (app stores) ne peuvent pas être mis à jour instantanément
- Les partenaires avec des intégrations ont besoin d'une période de transition

Il faut décider **où** et **comment** le numéro de version est exprimé, avant que l'API soit publiquement consommée.

---

## **Proposal**

### Versioning par URL

```
/api/v1/auth/register
/api/v1/tenants/{id}/audit-logs
/api/v2/tenants/{id}/audit-logs  ← future version si breaking change
```

**Règle de promotion de version** : une nouvelle version (`v2`) est créée **uniquement** lors d'un breaking change :
- Suppression d'un champ de réponse
- Changement de type d'un champ
- Changement de sémantique d'un endpoint
- Suppression d'un endpoint

Les **non-breaking changes** (ajout de champ optionnel, nouvel endpoint) ne requièrent **pas** de nouvelle version.

### Coexistence des versions

Les versions coexistent pendant une **période de dépréciation** (minimum 6 mois) :

```
/api/v1/... → DEPRECATED (header Deprecation: true, Sunset: date)
/api/v2/... → CURRENT
```

### Convention de routage

```go
// container.go
mux.HandleFunc("GET /api/v1/tenants/{id}/audit-logs", auditHandlerV1.ListByTenant)
mux.HandleFunc("GET /api/v2/tenants/{id}/audit-logs", auditHandlerV2.ListByTenant) // futur
```

Les handlers de différentes versions peuvent partager les mêmes use cases (seule la couche delivery change).

---

## **Drawbacks**

### Duplication des routes
Maintenir deux versions actives signifie deux ensembles de routes, DTOs et potentiellement handlers. Mitigé par le partage des use cases.

### URLs longues
`/api/v1/tenants/{id}/audit-logs` est plus verbeux que `/tenants/{id}/audit-logs`. Acceptable pour la clarté.

### Pollution du mux
Avec de nombreuses versions et routes, le `ServeMux` peut devenir dense. Un registraire de routes dédié est prévu pour la v1.1.

---

## **Alternatives**

### Versioning par header (`Accept-Version: v1`)
- ✅ URLs propres (pas de version dans le chemin)
- ❌ Impossible à tester depuis un navigateur ou curl sans tooling
- ❌ Non indexable (même URL, comportement différent)
- ❌ Difficile à configurer dans les gateways API (Nginx, Cloudflare)
- **Verdict : rejeté** — la lisibilité et la debuggabilité priment

### Versioning par sous-domaine (`v1.api.app.com`)
- ✅ Séparation complète
- ❌ Complexité DNS et TLS élevée
- ❌ CORS plus complexe
- **Verdict : rejeté**

### Pas de versioning (évolution continue sans breaking changes)
- ✅ Aucune gestion de version
- ❌ Contraint tous les breaking changes, bloque l'évolution
- **Verdict : rejeté** — impraticable sur la durée

### Calendar versioning (`/api/2026-05-01/...`)
- ✅ Indique la date de l'API (Stripe utilise ce modèle)
- ❌ Moins intuitif que v1/v2 pour les développeurs juniors
- **Verdict : différé** — à reconsidérer si l'API devient publique et très versionnée

---

## **Open Questions**

1. Après combien de versions simultanées faut-il forcer la migration (`v1` et `v2` max, ou `v1`, `v2`, `v3` ?) ?
2. Les webhooks suivent-ils la même convention de versioning ?
3. Faut-il un endpoint `GET /api/version` ou `GET /health` qui expose la version courante ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique (question ouverte #3)
- [Stripe API Versioning](https://stripe.com/docs/api/versioning)
- [Zalando RESTful API Guidelines — Versioning](https://opensource.zalando.com/restful-api-guidelines/#versioning)
- [Microsoft REST API Guidelines — Versioning](https://github.com/microsoft/api-guidelines/blob/vNext/azure/Guidelines.md#api-versioning)
