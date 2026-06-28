# _RFC #0035:_ Platform `white_label` — Branding Per-Tenant B2B2C

**date:** `2026 Jun 28`

---

## **Status:** `PROPOSED`

---

## **Summary**

Ce RFC définit le module `platform/white_label` : personnalisation de l'expérience customer-facing par tenant.

En B2B2C, le tenant souhaite que ses end-customers voient **sa marque**, pas celle de la plateforme. Le white-label couvre :
- Identité visuelle (logo, couleurs, favicon, police)
- Domaine personnalisé (`app.monentreprise.sn` → portail end-customer)
- Templates email/SMS brandés (en-tête, pied de page, couleur CTA)
- Textes d'interface personnalisables (nom de l'app customer, textes légaux)

Note : `tenant_settings` (RFC-0020) contient déjà `logo_url` et `primary_color`. Ce RFC **étend** ces colonnes et crée le module de gestion associé.

---

## **Motivation**

### Problème 1 — tenant_settings insuffisant pour le branding complet

RFC-0020 a ajouté `logo_url` et `primary_color` dans `tenant_settings` pour les besoins basiques du Merchant Mode. Le branding B2B2C nécessite :
- `secondary_color`, `accent_color` (système de couleurs complet)
- `favicon_url`, `app_name` (identité de l'app customer)
- `custom_domain` + vérification DNS (`CNAME` vers la plateforme)
- Templates email brandés par tenant (footer, logo dans les emails RFC-0036)

### Problème 2 — Domaine personnalisé sans workflow de vérification

`tenant_settings.custom_domain` est stocké mais il n'y a pas de workflow de vérification DNS. Un tenant peut déclarer un domaine qu'il ne possède pas → risque de phishing.

### Problème 3 — Pas de surface API dédiée

Les réglages white-label sont éparpillés dans `PATCH /api/v1/tenants/{id}/settings`. Ce RFC crée une surface dédiée `/api/v1/tenants/{id}/brand` plus claire.

---

## **Proposal**

### 1. Modèle domaine

```go
// platform/white_label/domain/brand.go
package domain

import "time"

type DomainVerificationStatus string

const (
    DomainNotSet    DomainVerificationStatus = "NOT_SET"
    DomainPending   DomainVerificationStatus = "PENDING"
    DomainVerified  DomainVerificationStatus = "VERIFIED"
    DomainFailed    DomainVerificationStatus = "FAILED"
)

type BrandSettings struct {
    TenantID          string
    AppName           string   // "MonApp" — affiché dans les emails et le portail
    LogoURL           string   // URL absolue vers l'objet stocké (RFC-0017)
    FaviconURL        string
    PrimaryColor      string   // hex #RRGGBB
    SecondaryColor    string
    AccentColor       string
    FontFamily        string   // 'inter' | 'roboto' | 'lato' | 'custom'
    CustomFontURL     string   // si FontFamily == 'custom', URL woff2

    // Domaine personnalisé
    CustomDomain           string
    CustomDomainStatus     DomainVerificationStatus
    CustomDomainToken      string    // token DNS TXT de vérification (généré, non modifiable)
    CustomDomainVerifiedAt *time.Time

    // Templates email (override des defaults RFC-0036)
    EmailFooterHTML  string   // max 2000 chars
    EmailLogoURL     string   // logo spécifique aux emails (peut différer du logo app)

    // Textes légaux
    TermsURL         string
    PrivacyURL       string
    SupportEmail     string
    SupportPhone     string

    UpdatedAt time.Time
}
```

### 2. Port Repository

```go
// platform/white_label/domain/repository.go
package domain

import "context"

type BrandRepository interface {
    Upsert(ctx context.Context, b *BrandSettings) error
    FindByTenantID(ctx context.Context, tenantID string) (*BrandSettings, error)
    FindByCustomDomain(ctx context.Context, domain string) (*BrandSettings, error)
    UpdateDomainStatus(ctx context.Context, tenantID string, status DomainVerificationStatus, verifiedAt *time.Time) error
}
```

### 3. Use cases

#### `GetBrandSettingsUseCase`

```
1. Permission : n'importe quel membre du tenant (read)
2. FindByTenantID — si non trouvé, retourner defaults (AppName = tenant.Name, etc.)
```

#### `UpdateBrandSettingsUseCase`

```
1. Permission : OWNER ou ADMIN
2. Valider : hex couleurs, URL logo (https:// requis), AppName max 50 chars
3. Si custom_domain modifié :
   3a. Générer token DNS TXT (UUID v4 préfixé "_kernel-verify=")
   3b. Passer custom_domain_status = PENDING
4. Upsert BrandSettings
5. event_outbox {type: "white_label.brand_updated", payload: {tenant_id}}
```

#### `VerifyCustomDomainUseCase`

```
1. Permission : OWNER
2. Charger BrandSettings, vérifier status == PENDING
3. DNS lookup TXT sur "_kernel-verify.<custom_domain>"
4. Si enregistrement TXT correspond au token stocké :
   4a. UpdateDomainStatus(VERIFIED, NOW())
   4b. event_outbox {type: "white_label.domain_verified"}
5. Sinon : UpdateDomainStatus(FAILED) + retourner erreur avec instructions
Note : la vérification DNS utilise net.LookupTXT (stdlib).
```

#### `GetPublicBrandUseCase`

```
Use case non authentifié — utilisé par le portail customer-facing pour charger le thème.
1. FindByCustomDomain si Host header correspond
2. Sinon FindByTenantID via param ?tenant_id=
3. Retourner BrandSettings (champs publics seulement : couleurs, logo, AppName)
```

### 4. Schéma SQL

```sql
-- platform/white_label/migrations/0001_create_brand_settings.up.sql

CREATE TABLE brand_settings (
    tenant_id                  UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    app_name                   VARCHAR(50)  NOT NULL DEFAULT '',
    logo_url                   VARCHAR(500),
    favicon_url                VARCHAR(500),
    primary_color              VARCHAR(7)   NOT NULL DEFAULT '#000000',
    secondary_color            VARCHAR(7)   NOT NULL DEFAULT '#FFFFFF',
    accent_color               VARCHAR(7)   NOT NULL DEFAULT '#0066FF',
    font_family                VARCHAR(20)  NOT NULL DEFAULT 'inter'
                                   CHECK (font_family IN ('inter','roboto','lato','custom')),
    custom_font_url            VARCHAR(500),
    custom_domain              VARCHAR(255),
    custom_domain_status       VARCHAR(20)  NOT NULL DEFAULT 'NOT_SET'
                                   CHECK (custom_domain_status IN ('NOT_SET','PENDING','VERIFIED','FAILED')),
    custom_domain_token        VARCHAR(100),
    custom_domain_verified_at  TIMESTAMPTZ,
    email_footer_html          TEXT,
    email_logo_url             VARCHAR(500),
    terms_url                  VARCHAR(500),
    privacy_url                VARCHAR(500),
    support_email              VARCHAR(255),
    support_phone              VARCHAR(20),
    updated_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Lookup rapide par domaine personnalisé (routing portail)
CREATE UNIQUE INDEX idx_brand_custom_domain
    ON brand_settings(custom_domain)
    WHERE custom_domain IS NOT NULL AND custom_domain_status = 'VERIFIED';
```

> **Migration tenant_settings** : les colonnes `logo_url` et `primary_color` de `tenant_settings` (RFC-0020) sont dépréciées. Une migration transfère les données existantes vers `brand_settings` et retire les colonnes de `tenant_settings` en deux étapes (déploiement zero-downtime).

### 5. API HTTP

```
# White-label management (OWNER/ADMIN)
GET    /api/v1/tenants/{id}/brand          GetBrandSettings
PATCH  /api/v1/tenants/{id}/brand          UpdateBrandSettings
POST   /api/v1/tenants/{id}/brand/verify-domain   VerifyCustomDomain

# Public (pas de JWT — portail customer)
GET    /public/v1/brand                    GetPublicBrand (?tenant_id= ou Host header)
```

### 6. Vérification DNS (stdlib)

```go
// platform/white_label/usecase/verify_domain.go
import "net"

func lookupVerificationToken(domain, expectedToken string) bool {
    records, err := net.LookupTXT("_kernel-verify." + domain)
    if err != nil {
        return false
    }
    for _, r := range records {
        if r == expectedToken {
            return true
        }
    }
    return false
}
```

Pas de dépendance externe — `net.LookupTXT` est stdlib.

---

## **Drawbacks**

1. **Vérification DNS synchrone dans le use case** : `net.LookupTXT` peut être lente (timeout DNS). À lancer dans une goroutine avec timeout court (5s) ou différer dans un worker outbox.
2. **Upload logo** : ce RFC ne gère pas l'upload direct (RAF RFC-0017 presign v2). Le tenant fournit une URL `https://` déjà hébergée.
3. **Migration tenant_settings** : retrait des colonnes `logo_url` / `primary_color` de `tenant_settings` est un breaking change — nécessite déploiement coordonné avec les consommateurs de l'API.

---

## **Alternatives**

### Stocker les settings brand dans tenant_settings (JSONB)

- ✅ Pas de nouvelle table
- ❌ Pas d'index sur custom_domain, pas de contraintes fines par champ
- **Verdict : rejeté** — table dédiée pour indexabilité et clarté

### Déléguer la vérification DNS à un service externe (Cloudflare Workers)

- ✅ Vérification plus rapide et fiable (résolution globale)
- ❌ Dépendance externe, contraire à la politique stdlib-first
- **Verdict : différé** — stdlib suffit pour la v1

---

## **Plan d'adoption**

1. Review Aziz + Lucas
2. Migration `tenant_settings` : ajouter colonne `brand_migrated BOOLEAN DEFAULT FALSE`, puis migrer les données
3. Implémenter `platform/white_label` : migrations → domain → infra → usecase → delivery → tests
4. Retirer colonnes dépréciées de `tenant_settings` (migration séparée, déploiement J+7)
5. ADR post-implémentation

---

## **References**

- [RFC-0033 — Platform end_customers](0033-Platform-End-Customers.md)
- [RFC-0017 — Platform Object Storage](0017-Platform-Object-Storage.md) *(presign upload logo — v2)*
- [RFC-0036 — Platform customer_notifications](0036-Platform-Customer-Notifications.md) *(email_logo_url dans templates)*
- [RFC-0020 — Platform Billing](0020-Platform-Billing-Hybride-Merchant-Mode.md) *(tenant_settings.custom_domain existant)*
- [RFC-0038 — Conventions Package Réutilisable](0038-Conventions-Package-Reutilisable.md)
