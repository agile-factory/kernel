# _RFC #0017:_ Platform `storage` — Object Storage Abstraction

**date:** `2026 May 04`

---

## **Status:** `ACCEPTED + IMPL v0.1.0 partial` (S3 backend via minio-go ciblé Cloudflare R2 ; presigned URLs reportés en v2 — cf. `platform/storage/README.md`)

---

## **Summary**

Ce RFC définit la plateforme `storage`, brique transversale du socle Agile Factory qui fournit une **abstraction unifiée pour le stockage d'objets binaires** (fichiers, images, documents, exports) à travers tous les modules métier et plateformes.

La proposition repose sur trois piliers :

1. **Port abstrait `kernel/storage`** indépendant de tout vendor (S3, GCS, local).
2. **Trois backends** : `LocalFS` (dev), `S3Compatible` (prod / MinIO staging), avec sélection par variable d'environnement.
3. **Métadonnées en base** (`storage_objects`) comme source de vérité pour audit, quota et garbage collection.

L'objectif : permettre à n'importe quel module (`billing` pour les factures PDF, `notification` pour les pièces jointes, `user` pour les avatars, futurs modules métier pour leurs assets) d'écrire et lire des objets sans connaître le backend, et préparer l'exposition de cette capacité aux tenants externes (palier BaaS).

---

## **Motivation**

### Cas d'usage transverses déjà identifiés

| Origine | Besoin |
|---|---|
| RFC-0012 (Billing) | Factures PDF émises par le Merchant Mode (`tenant_invoices`) |
| RFC-0005 (Audit) | Exports CSV / JSON des journaux d'audit pour conformité |
| RFC-0009 (User Profiles) | Avatars utilisateurs |
| RFC-0011 (Notifications) | Pièces jointes email (factures, contrats) |
| Modules métier futurs | Documents, photos, vidéos selon la verticale (immobilier, e-commerce, RH) |
| Palier BaaS futur | Endpoint d'upload exposé aux tenants externes via API key |

Sans plateforme commune :
- Chaque module réinvente l'I/O fichier (lock disque vs S3 SDK direct dans les handlers).
- Aucun garde-fou multi-tenant sur les chemins → risque de fuite cross-tenant (cohérent avec les risques RFC-0003).
- Comptabilisation du quota `MaxStorage` (RFC-0012) impossible sans index centralisé.
- Migration du dev (filesystem local) vers la prod (S3) coûte une réécriture par module.

### Pourquoi maintenant

Aucun module métier n'a encore intégré de stockage binaire. Poser le port **avant** la première intégration évite un refactor coûteux. RFC-0002 L1.3 listait déjà `aws-sdk-go-v2` comme dépendance future autorisée — ce RFC formalise son introduction (confinée).

---

## **Proposal**

### 1. Port abstrait (kernel)

```go
// kernel/storage/storage.go
package storage

import (
    "context"
    "io"
    "time"
)

type Storage interface {
    Put(ctx context.Context, key string, body io.Reader, opts PutOptions) (*ObjectRef, error)
    Get(ctx context.Context, key string) (io.ReadCloser, *ObjectRef, error)
    Delete(ctx context.Context, key string) error
    Stat(ctx context.Context, key string) (*ObjectRef, error)
    SignedURL(ctx context.Context, key string, op SignedOp, ttl time.Duration) (string, error)
}

type PutOptions struct {
    ContentType   string
    ContentLength int64
    Metadata      map[string]string // tags applicatifs (module, resource_id)
    Checksum      string            // SHA-256 hex (optionnel, vérifié si présent)
}

type ObjectRef struct {
    Key          string
    Size         int64
    ContentType  string
    ETag         string
    Checksum     string
    LastModified time.Time
}

type SignedOp string
const (
    SignedOpGet SignedOp = "GET"
    SignedOpPut SignedOp = "PUT"
)
```

Les use cases métier consomment `Storage` ; ils n'importent jamais `aws-sdk-go-v2` ni `os.Open` directement.

### 2. Modèle de clé multi-tenant

Toutes les clés suivent le format :

```
tenants/{tenant_id}/{module}/{resource_id}/{filename}
```

Exemples :
- `tenants/3f1c.../billing/inv_a4b8.../invoice.pdf`
- `tenants/3f1c.../user/usr_77.../avatar.png`
- `tenants/3f1c.../audit/exports/2026-05.csv`

**Garde-fou** : un wrapper `TenantScopedStorage` (équivalent des repository wrappers RFC-0003) extrait le `tenant_id` du `context.Context` et préfixe automatiquement la clé. Toute opération sans `tenant_id` en contexte échoue.

### 3. Backends

#### 3.1 `LocalFS` (dev / tests)

Stockage sous `STORAGE_DIR` (défaut `./storage`). Métadonnées (`ContentType`, `Metadata`, `Checksum`) stockées dans un fichier sidecar `.meta.json`. Signed URLs simulées via tokens HMAC (signés avec `STORAGE_SIGNING_KEY`).

#### 3.2 `S3Compatible` (staging MinIO / prod AWS)

Implémentation via `aws-sdk-go-v2/service/s3`. **Confiné dans `platform/storage/infrastructure/s3_storage.go`** — aucun import en dehors. Compatible AWS S3, MinIO, Cloudflare R2, Backblaze B2 (tous parlent S3 API).

Variables d'environnement :
```
STORAGE_DRIVER=s3
STORAGE_S3_ENDPOINT=https://s3.amazonaws.com
STORAGE_S3_REGION=eu-west-3
STORAGE_S3_BUCKET=agile-factory-prod
STORAGE_S3_ACCESS_KEY=...
STORAGE_S3_SECRET_KEY=...
STORAGE_S3_FORCE_PATH_STYLE=false
```

### 4. Wiring (bootstrap)

Sélection dans `internal/bootstrap/container.go` :

```go
switch cfg.Storage.Driver {
case "local":
    storageImpl = localfs.New(cfg.Storage.Dir, cfg.Storage.SigningKey)
case "s3":
    storageImpl = s3storage.New(cfg.Storage.S3)
default:
    return nil, fmt.Errorf("unknown STORAGE_DRIVER=%s", cfg.Storage.Driver)
}
container.Storage = storage.NewTenantScoped(storageImpl)
```

### 5. Signed URLs

Pour les fichiers volumineux, l'upload direct client → S3 évite de transiter par l'API Go.

```
1. Client → POST /api/v1/uploads {filename, size, content_type}
2. API   → SignedURL(PUT, ttl=15min) + insère "PENDING" dans storage_objects
3. Client → PUT vers URL signée (direct S3)
4. Client → POST /api/v1/uploads/{id}/confirm
5. API   → Stat() pour vérifier présence + checksum, marque "READY"
```

TTL par défaut : 15 minutes pour PUT, 5 minutes pour GET. Configurable.

### 6. Modèle de données

```sql
CREATE TABLE storage_objects (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module        VARCHAR(50) NOT NULL,        -- 'billing', 'user', 'audit', ...
    resource_id   VARCHAR(255),                -- id de la ressource métier liée
    storage_key   VARCHAR(1024) NOT NULL UNIQUE,
    size_bytes    BIGINT NOT NULL DEFAULT 0,
    content_type  VARCHAR(255) NOT NULL,
    checksum      VARCHAR(128),                -- SHA-256 hex
    status        VARCHAR(20) NOT NULL DEFAULT 'PENDING', -- PENDING | READY | DELETED
    uploaded_by   UUID REFERENCES users(id),
    metadata      JSONB,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_storage_tenant_module ON storage_objects(tenant_id, module) WHERE deleted_at IS NULL;
CREATE INDEX idx_storage_tenant_size  ON storage_objects(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_storage_pending      ON storage_objects(created_at) WHERE status = 'PENDING';
```

`storage_objects` est la **source de vérité** : le backend physique peut diverger temporairement (objets en transit, GC en retard), la base reste cohérente côté quota et audit.

### 7. Quota & limites

Couplage avec RFC-0012 (`PlanQuotas.MaxStorage`) :

```go
// platform/storage/usecase/upload.go
func (uc *UploadUseCase) Execute(ctx context.Context, p UploadParams) (*ObjectRef, error) {
    used, _ := uc.repo.SumSizeByTenant(ctx, p.TenantID)
    if used + p.Size > uc.quota.MaxStorage(ctx, p.TenantID) {
        return nil, billing.ErrQuotaExceeded
    }
    // ... Put + insert
}
```

Limites complémentaires (configurables, valeurs par défaut) :
- `STORAGE_MAX_OBJECT_SIZE` : 100 MiB par objet
- `STORAGE_ALLOWED_MIME` : whitelist par défaut (`image/*`, `application/pdf`, `text/csv`, `application/json`)
- `STORAGE_RATE_LIMIT_PER_MINUTE` : 60 uploads/minute/tenant (sera durci par le futur RFC Rate Limiting)

### 8. Sécurité

| Aspect | Mesure v1 |
|---|---|
| MIME validation | Détection magic bytes (`net/http.DetectContentType`) + cross-check avec `Content-Type` déclaré |
| Taille | Header `Content-Length` + limite stricte côté handler (streaming + cap) |
| AV scanning | Interface `Scanner` injectable, no-op v1, intégration ClamAV/Lambda v2 (async via event bus RFC-0015) |
| Chiffrement at-rest | S3 SSE-S3 par défaut (prod) ; LocalFS hors scope (dev uniquement) |
| Chiffrement in-transit | HTTPS obligatoire (signed URLs avec scheme `https`) |
| ACL | Tous les objets sont privés ; aucun bucket public ; accès via signed URLs uniquement |
| Path traversal | Sanitization stricte dans le wrapper `TenantScopedStorage` (rejet de `..`, `/`, etc. dans les noms) |

### 9. Suppression et rétention

- **Soft-delete** : `Delete()` met `deleted_at = NOW()` et `status = 'DELETED'` ; l'objet physique reste 30 jours (configurable via `STORAGE_RETENTION_DAYS`).
- **GC** : un job de garbage collection (préfigure le futur RFC Background Jobs) supprime physiquement après expiration.
- **PENDING orphelins** : les objets `PENDING` non confirmés depuis >24h sont supprimés.
- **Hard-delete** : endpoint admin pour suppression immédiate (compliance / GDPR ; sera formalisé dans le futur RFC GDPR).

### 10. Plan de mise en œuvre

| Phase | Contenu | Durée estimée |
|---|---|---|
| 1 | Port `kernel/storage` + backend `LocalFS` + table `storage_objects` + tests unitaires | 3-4 jours |
| 2 | Backend `S3Compatible` + tests d'intégration MinIO (Docker Compose) | 3 jours |
| 3 | Signed URLs + endpoint `/api/v1/uploads` + intégration quota RFC-0012 | 3 jours |
| 4 | Premier consommateur réel : factures PDF (RFC-0012 Merchant Mode) | 2 jours |
| 5 | GC périodique (orphelins PENDING + soft-deleted expirés) | 1 jour |

**Total : ~2 semaines** pour un stockage production-ready.

---

## **Drawbacks**

### Coût opérationnel S3
Un bucket S3 facture stockage + bandwidth + requêtes. Sur des SaaS à fort volume (vidéos, archives), le coût peut dépasser celui de Postgres. Mitigation : tier `S3 Intelligent-Tiering` ou backend Backblaze B2 / Cloudflare R2 (moins chers, S3-compatible).

### Complexité des signed URLs
Le flux upload-direct (3 round-trips client/API) est plus complexe qu'un POST multipart classique. Mitigation : un SDK client (futur RFC SDK Generation) masquera la complexité.

### Validation MIME imparfaite
Magic bytes detection détecte les types courants mais peut être contournée (ex: PDF avec polyglot). Pour des SaaS sensibles, l'AV scanning v2 et un sandbox de prévisualisation sont recommandés.

### GC à orchestrer
Le job de garbage collection doit tourner à intervalle régulier. En v1, simple goroutine périodique ; en v2, intégration avec le futur RFC Background Jobs.

### Cohérence éventuelle base ↔ backend
Si l'insert DB réussit mais le `Put` S3 échoue (ou inverse), `storage_objects` peut diverger du backend. Mitigation : pattern « DB d'abord en PENDING, backend, puis confirm ». Échecs détectés par GC.

### Pas de versioning d'objets en v1
Une réécriture sur la même clé écrase l'ancien contenu. Pour les SaaS qui ont besoin d'historique (contrats juridiques), une stratégie de clés versionnées (`{filename}-v{N}`) est applicative en v1, S3 versioning natif en v2.

---

## **Alternatives**

### Alternative A : BLOB en base Postgres
**Description** : stocker les fichiers en colonnes `BYTEA` directement dans la DB.

✅ **Avantages**
- Transactionnel (atomicité avec les métadonnées)
- Pas d'infrastructure supplémentaire
- Backup unifié

❌ **Inconvénients**
- Ne scale pas au-delà de quelques GB (lent, fragmente la table)
- Backup volumineux (chaque dump contient tout)
- Pas de signed URLs / accès direct
- Coût Postgres > coût S3 à volume égal

**Verdict** : **rejeté**. Adapté uniquement pour de petits objets <100 KB et faibles volumes.

### Alternative B : Upload direct dans les handlers HTTP (sans abstraction)
**Description** : chaque handler reçoit le multipart, écrit directement sur disque ou S3.

✅ **Avantages**
- Implémentation immédiate, zéro infrastructure
- Aucun port à concevoir

❌ **Inconvénients**
- Code dupliqué dans chaque handler
- Couplage fort au backend (changement = N modifications)
- Pas de comptabilisation du quota
- Pas de garde-fou multi-tenant systématique

**Verdict** : **rejeté**. Anti-pattern incompatible avec la Clean Architecture du socle (RFC-0002).

### Alternative C : Service tiers managé (Uploadcare, Cloudinary, Bunny)
**Description** : déléguer entièrement le stockage à un SaaS tiers.

✅ **Avantages**
- CDN inclus
- Transformations d'images / vidéos out-of-the-box
- Pas d'infra à opérer

❌ **Inconvénients**
- Lock-in fort (les fichiers vivent chez le tiers)
- Coût élevé à volume
- Pas adapté pour les documents sensibles (factures, contrats) → souveraineté
- Difficile à exposer en BaaS (tenant ne peut pas BYO bucket)

**Verdict** : **différé**. Le port `Storage` autorise une implémentation `CloudinaryStorage` future si un module spécifique en a besoin (ex: galerie photo). Pas pertinent pour le socle.

### Alternative D : NFS / volumes partagés
**Description** : monter un volume réseau (NFS, EFS) accessible par toutes les instances.

✅ **Avantages**
- Simple côté code (POSIX filesystem)
- Pas de SDK à intégrer

❌ **Inconvénients**
- Single point of failure si non répliqué
- Performances dégradées sur petits fichiers
- Pas de signed URLs (nécessite un proxy applicatif pour servir)
- Pas d'écosystème CDN

**Verdict** : **rejeté**. Anti-cloud-native. Le LocalFS du dev couvre déjà le besoin sans réseau.

---

## **Open Questions**

1. **CDN devant S3** : CloudFront, Bunny CDN, ou Cloudflare ? Pertinent pour les avatars publics, exclu pour les factures privées. À trancher avec le premier module à fort trafic public.
2. **Multipart upload** : v1 (single PUT, max 100 MiB) ou v2 (multipart, fichiers >5 GiB) ? Décision selon les premiers cas d'usage.
3. **Versioning natif S3** : activer le versioning bucket pour les modules « legal » (contrats) ? Coût stockage 2-3× supérieur.
4. **AV scanning sync ou async** : sync = latence d'upload élevée, async = fenêtre où le fichier est exposé avant scan. Le pattern outbox du RFC-0015 amendé permettrait un scan asynchrone fiable.
5. **Quotas par module** : faut-il des sous-quotas (`MaxStorageInvoices`, `MaxStorageAvatars`) ou un seul `MaxStorage` global ? Couplage avec RFC-0012 à arbitrer.
6. **Bucket par tenant vs préfixe partagé** : pour les très gros tenants ou les exigences de compliance, un bucket dédié peut être préférable. Décision à coupler avec un futur palier BaaS « entreprise ».
7. **Exposition BaaS** : doit-on exposer `Storage` directement aux tenants externes (via API key) en v1 du palier BaaS, ou attendre le RFC SDK Generation ?

---

## **References**

- RFC-0002 §1.3 : Architecture & Stack — `aws-sdk-go-v2` listé comme dépendance future
- RFC-0003 : Multi-Tenancy — modèle de clé `tenants/{id}/...` cohérent avec le filtrage applicatif
- RFC-0005 : Platform Audit — consommateur (exports CSV/JSON)
- RFC-0009 : Platform User Profiles — consommateur (avatars)
- RFC-0011 : Platform Notification — consommateur (pièces jointes email)
- RFC-0012 : Platform Billing — quota `MaxStorage` + factures PDF Merchant Mode
- RFC-0015 : Event Bus Interne (amendé v1.1) — support async pour AV scanning, GC, replication
- [AWS S3 SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/sdk-utilities/s3/)
- [MinIO — S3-compatible object storage](https://min.io/)
- [Cloudflare R2 — Zero egress S3-compatible storage](https://www.cloudflare.com/products/r2/)
