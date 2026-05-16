// Package storage is the object-storage platform module. It packages
// an S3-compatible backend (Cloudflare R2 / MinIO / AWS S3 / B2) so a
// consumer can wire it in one line and inject the resulting
// storage.Storage into modules that need to read/write blobs (logos,
// exports, attachments).
//
// The module has no HTTP routes, no migrations and no DB tables in
// v0.1.0 — RegisterRoutes is a no-op. Future versions (RFC-0017 phase 2)
// will add presigned URL endpoints; the Storage interface in
// kernel/storage will grow a Presign method at that point.
//
// Aligned with RFC-0017 + RFC-0038.
package storage

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	kstorage "github.com/agile-factory/kernel/kernel/storage"
	"github.com/agile-factory/kernel/platform/storage/infrastructure"
)

// Note: unlike other platform modules, storage does NOT export a
// Migrations FS — the module owns no SQL tables. Consumers wiring
// migrator.Run should simply omit storage from the source slice.

// Storage re-exports the port so consumers can depend on this single
// package rather than importing kernel/storage separately.
type Storage = kstorage.Storage

// Object re-export.
type Object = kstorage.Object

// ObjectMeta re-export.
type ObjectMeta = kstorage.ObjectMeta

// ListEntry re-export.
type ListEntry = kstorage.ListEntry

// PutOptions re-export.
type PutOptions = kstorage.PutOptions

// ErrNotFound re-export — callers can use errors.Is(err, storage.ErrNotFound).
var ErrNotFound = kstorage.ErrNotFound

// S3Config re-exports the infrastructure-level config so the consumer
// doesn't have to import the infrastructure package.
type S3Config = infrastructure.S3Config

// Config captures the dependencies of the storage Module. Only the S3
// backend exists in v0.1.0; future LocalFS / GCS backends will land as
// alternative fields with a Backend selector.
type Config struct {
	S3     S3Config
	Logger *slog.Logger
}

// Module exposes a ready-to-use storage.Storage. Build it once at boot,
// inject the returned Storage into any module that needs object I/O.
type Module struct {
	cfg     Config
	backend Storage
}

// New constructs the module and validates that the backend is reachable
// (BucketExists round-trip). Surfacing the error here turns "credentials
// or bucket wrong" into a boot-time failure instead of a runtime 5xx.
func New(ctx context.Context, cfg Config) (*Module, error) {
	if cfg.Logger == nil {
		return nil, errors.New("storage.Config: Logger is required")
	}

	backend, err := infrastructure.NewS3Storage(ctx, cfg.S3)
	if err != nil {
		return nil, err
	}

	cfg.Logger.Info("storage: S3 backend ready",
		"endpoint", cfg.S3.Endpoint,
		"bucket", cfg.S3.Bucket,
		"region", cfg.S3.Region,
	)

	return &Module{cfg: cfg, backend: backend}, nil
}

// Storage returns the backend handle. Inject this into the Config of
// any consumer module that needs to put/get/delete objects.
func (m *Module) Storage() Storage {
	return m.backend
}

// RegisterRoutes is a no-op in v0.1.0. Kept for RFC-0038 symmetry so the
// consumer's main.go can call it uniformly across all modules.
func (m *Module) RegisterRoutes(_ *http.ServeMux) {}
