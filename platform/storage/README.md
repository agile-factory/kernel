# `platform/storage/` — Object storage

> ✅ **Implémenté (v0.1.0 partial)** — RFC-0017. Port abstrait + backend S3-compatible (Cloudflare R2 / MinIO / AWS S3 / B2 / DO Spaces) via `minio-go`. URLs présignées en v2.

Brique transverse de stockage d'objets binaires (logos, exports, attachments). Le **port** vit dans [`kernel/storage`](../../kernel/storage/storage.go) ; le **backend S3** est ici. Les modules métier dépendent de l'interface `Storage`, pas de minio-go directement → swap de backend sans refactor.

---

## Wiring

```go
import (
    "context"

    "github.com/AgileFactory/kernel/platform/storage"
)

storageMod, err := storage.New(context.Background(), storage.Config{
    S3: storage.S3Config{
        Endpoint:  cfg.StorageEndpoint, // ex: "<account>.r2.cloudflarestorage.com"
        Region:    "auto",              // R2 default; AWS = "eu-west-3" etc.
        Bucket:    cfg.StorageBucket,
        AccessKey: cfg.StorageAccessKey,
        SecretKey: cfg.StorageSecretKey,
        UseSSL:    true,
        PathStyle: false,               // true seulement pour MinIO local
    },
    Logger: log,
})
if err != nil { log.Error("storage init", "error", err); os.Exit(1) }

// Inject dans n'importe quel module consommateur :
brandingMod := branding.New(branding.Config{Storage: storageMod.Storage(), ...})
```

⚠️ `storage.New` exécute un `BucketExists` round-trip — toute mauvaise credential / mauvais bucket plante au boot, pas en runtime.

---

## Config

| Champ S3Config | Requis | Cloudflare R2 | AWS S3 | MinIO local |
|---|---|---|---|---|
| `Endpoint` | ✅ | `<account>.r2.cloudflarestorage.com` | `s3.<region>.amazonaws.com` | `localhost:9000` |
| `Region` | ⏳ | `auto` (default) | `eu-west-3` etc. | n'importe quoi |
| `Bucket` | ✅ | bucket R2 | bucket S3 | bucket MinIO |
| `AccessKey` | ✅ | R2 API token | AWS IAM key | MinIO key |
| `SecretKey` | ✅ | R2 secret | AWS IAM secret | MinIO secret |
| `UseSSL` | ✅ | `true` | `true` | `false` typiquement |
| `PathStyle` | ⏳ | `false` | `false` | `true` (obligatoire) |

---

## Surface API

Interface `kernel/storage.Storage` (ré-exportée comme `storage.Storage`) :

```go
type Storage interface {
    Put(ctx, key, body, size int64, opts PutOptions) (ObjectMeta, error)
    Get(ctx, key) (Object, error)                 // !! caller MUST Close()
    Delete(ctx, key) error                         // idempotent (404 = no-op)
    Exists(ctx, key) (bool, error)
    Stat(ctx, key) (ObjectMeta, error)             // ErrNotFound sur absent
    List(ctx, prefix) (<-chan ListEntry, error)    // streaming, buffer 16
}
```

Exemple — upload d'un logo de tenant :

```go
meta, err := s.Put(ctx, "tenants/"+tenantID+"/logo.png", file, fileSize, storage.PutOptions{
    ContentType:  "image/png",
    CacheControl: "public, max-age=31536000",
})
if err != nil { return err }
return s.tenantSettingsRepo.UpdateLogoURL(ctx, tenantID, meta.Key) // ou une URL publique
```

Exemple — lecture :

```go
obj, err := s.Get(ctx, key)
if errors.Is(err, storage.ErrNotFound) { return errors.NotFound("Logo missing") }
if err != nil { return err }
defer obj.Close()                                // ⚠️ obligatoire
return io.Copy(w, obj.Body)
```

---

## Erreurs

| Erreur | Quand | Test |
|---|---|---|
| `storage.ErrNotFound` | `Get` / `Stat` sur clé inexistante | `errors.Is(err, storage.ErrNotFound)` |
| Wrap `fmt.Errorf("storage/s3: ...")` | Toute autre erreur backend | Logger / surfacer comme `kernel/errors.InternalServerError` |

`Delete` sur clé inexistante = succès (idempotence).

---

## Hors-scope v0.1.0

| Feature | Pourquoi reportée | Cible |
|---|---|---|
| **URLs présignées** (`Presign`) | Pas de besoin v0.1.0 (uploads serveur < 50 MB acceptables) | v2 — RFC-0017 phase 2 |
| **Backend LocalFS** | Cloudflare R2 utilisé partout, dev avec un bucket dédié | Réviser si dev sans réseau devient pénible |
| **Backend GCS / Azure Blob** | Non S3-API natif | À ouvrir via une variante de l'interface si demandé |
| **Versioning + lifecycle policies** | Configurés côté bucket (R2 / S3 console), pas dans le code Go | Documenter dans runbook |

---

## Migrations

Le module **n'expose pas** de `Migrations` embed.FS (contrairement aux autres modules `platform/*`) — il ne possède aucune table SQL. Ne pas inclure storage dans le slice passé à `migrator.Run`.

---

## Références

- [RFC-0017](../../docs/architecture/rfcs/0017-Platform-Object-Storage.md) — spec.
- [`kernel/storage/storage.go`](../../kernel/storage/storage.go) — port abstrait.
- [`infrastructure/s3_storage.go`](infrastructure/s3_storage.go) — backend minio-go.
- [minio-go SDK](https://github.com/minio/minio-go) — driver (works against R2, MinIO, AWS S3, B2, Spaces).
