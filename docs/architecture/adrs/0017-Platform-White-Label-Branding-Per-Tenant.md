# _ADR #0017:_ Platform `white_label` — Branding Per-Tenant B2B2C (RFC-0035)

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0035 définit le module `platform/white_label` : configuration de marque per-tenant pour le portail B2B2C customer-facing. Implémenté sur `feat/p1-v0.2.0` en 2026-06-28 (commit `f98e074`).

---

## **Decision**

### Architecture

Structure standard `platform/` du kernel :
- `domain/` — `BrandSettings` entity, `DomainVerificationStatus`, ports `BrandRepository`, erreurs sentinel
- `infrastructure/` — `PostgresBrandRepository` (upsert pattern, PRIMARY KEY = tenant_id)
- `usecase/` — `GetBrandSettings`, `UpdateBrandSettings`, `VerifyCustomDomain`, `GetPublicBrand`
- `delivery/http/` — `BrandHandler` (4 endpoints)
- `white_label.go` — `Module` / `Config` + `RegisterRoutes`
- `migrations/` — `0001_create_brand_settings.up/down.sql`

### Upsert pattern (one row per tenant)

`brand_settings` est une table 1:1 avec `tenants` (PRIMARY KEY = tenant_id). `Upsert()` utilise `INSERT ... ON CONFLICT (tenant_id) DO UPDATE` pour créer ou remplacer. Raison : le module n'a pas de notion de "create first, then update" — le tenant peut patcher dès le premier appel.

### DomainVerificationStatus — 5 états distincts

La vérification DNS distingue 5 états au lieu des 3 états habituels :
- `NOT_SET` : pas de domaine configuré
- `PENDING` : domaine soumis, en attente de vérification
- `TIMEOUT` : erreur réseau transitoire — relancer est sûr
- `VERIFIED` : TXT enregistrement trouvé et correspond au token
- `FAILED` : TXT trouvé mais ne correspond pas — reconfiguration requise

`TIMEOUT` distinct de `FAILED` (RFC-0035 amendment Lucas) : un timeout réseau ne doit pas pénaliser le tenant de manière permanente. `FAILED` signifie que le tenant a mal configuré son DNS.

### DNSResolver injectable

```go
type DNSResolver interface {
    LookupTXT(ctx context.Context, name string) ([]string, error)
}
```

Production : `usecase.NetResolver()` qui wrape `net.DefaultResolver`. Tests : stub qui retourne des enregistrements précis sans réseau. `context.WithTimeout(ctx, 5*time.Second)` appliqué dans le use case (pas dans le resolver) pour garantir que la vérification DNS ne bloque jamais une requête HTTP.

### Permissions

Deux constantes ajoutées dans `kernel/permission/permission.go` :
- `BrandRead` (`brand:read`) : tout membre du tenant
- `BrandWrite` (`brand:write`) : OWNER ou ADMIN

`VerifyCustomDomain` utilise `BrandWrite` (pas une permission spécifique) — la vérification de domaine est une opération ADMIN/OWNER.

### Endpoint public non authentifié

`GET /public/v1/brand` est résolu par Host header (portail custom domain) ou `?tenant_id=` query param. Il retourne `publicBrandResponse` qui omet les champs sensibles : token de vérification DNS, `EmailFooterHTML`.

### EmailFooterHTML — warning XSS documenté

`EmailFooterHTML` est stocké en TEXT sans contrainte SQL de longueur (contrainte dans le use case : max 2000 chars). Il est exclusivement réservé au rendu côté serveur dans les emails HTML. Il NE DOIT JAMAIS être rendu dans un navigateur sans sanitisation (XSS). Ce fait est documenté en commentaire Go sur le champ et dans le modèle domaine.

---

## **Alternatives rejetées**

### Stocker le branding dans tenant_settings (JSONB)

- ✅ Pas de nouvelle table
- ❌ Pas d'index efficace sur custom_domain, pas de contraintes fines, pas d'index partiel
- **Verdict : rejeté** — table dédiée `brand_settings` avec index sur `custom_domain_status = 'VERIFIED'`

### Vérification DNS dans un worker outbox

- ✅ Non-bloquant pour la requête HTTP
- ❌ Complexité supplémentaire, contraire à YAGNI en v1
- **Verdict : différé** — timeout 5s dans le use case suffit pour v1

### 3 états DNS (NOT_SET, VERIFIED, FAILED)

- ✅ Plus simple
- ❌ Timeout réseau → FAILED définitif → tenant bloqué sans faute de sa part
- **Verdict : rejeté** — `TIMEOUT` ajouté comme état transitoire retryable

---

## **Conséquences**

- `platform/customer_notifications` (RFC-0036) peut lire `EmailLogoURL` et `EmailFooterHTML` pour les templates email
- `platform/customer_auth` (RFC-0034) peut utiliser `AppName` pour les OTP emails
- `GetPublicBrand` (`/public/v1/brand`) est disponible pour le portail end-customer
- Migration `tenant_settings` (retrait de `logo_url` / `primary_color`) est différée en déploiement J+7 selon le plan d'adoption RFC-0035
