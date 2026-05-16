// Package storage is the object-storage port. It defines the abstract
// contract that platforms consume to read/write blobs (logos, exports,
// attachments) without depending on a specific cloud vendor.
//
// Backends live in platform/storage/infrastructure/ — currently a single
// S3-compatible backend that targets Cloudflare R2 in production and
// switches to self-hosted MinIO transparently (same wire protocol).
// Consumers depend on the Storage interface here; the concrete backend
// is injected by main.go.
//
// Aligned with RFC-0017.
package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

// Storage is the abstract port. All implementations are safe for
// concurrent use by multiple goroutines.
type Storage interface {
	// Put streams body under key and returns its post-write metadata.
	// PutOptions controls ContentType, CacheControl, etc.; pass a zero
	// value for sensible defaults.
	Put(ctx context.Context, key string, body io.Reader, size int64, opts PutOptions) (ObjectMeta, error)

	// Get streams the object identified by key. The caller MUST Close()
	// the returned Object when done — failing to close leaks the
	// underlying HTTP response.
	Get(ctx context.Context, key string) (Object, error)

	// Delete removes the object. Deleting a missing key is a no-op
	// (does NOT return ErrNotFound) so callers can be idempotent.
	Delete(ctx context.Context, key string) error

	// Exists returns whether the key resolves to a stored object.
	// Cheaper than Stat — does not fetch metadata.
	Exists(ctx context.Context, key string) (bool, error)

	// Stat returns metadata for the object without reading its body.
	// Returns ErrNotFound when the key does not exist.
	Stat(ctx context.Context, key string) (ObjectMeta, error)

	// List streams keys whose names share the given prefix. The
	// channel is closed when iteration completes or ctx is cancelled.
	// Implementations buffer internally — receivers can be slow without
	// blocking the remote pagination.
	List(ctx context.Context, prefix string) (<-chan ListEntry, error)
}

// Object is a readable handle on a stored blob plus its metadata.
// Body MUST be closed by the caller.
type Object struct {
	Meta ObjectMeta
	Body io.ReadCloser
}

// Close releases the underlying body. Safe to call multiple times.
func (o *Object) Close() error {
	if o == nil || o.Body == nil {
		return nil
	}
	return o.Body.Close()
}

// ObjectMeta is the post-write or post-stat description of an object.
// All times are UTC.
type ObjectMeta struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// ListEntry is one item streamed by List. Err is non-nil only on the
// final entry of a failed iteration; receivers should treat any non-nil
// Err as terminal and stop reading.
type ListEntry struct {
	Meta ObjectMeta
	Err  error
}

// PutOptions are the optional knobs on Put. Leaving fields zero asks
// the backend to apply its defaults.
type PutOptions struct {
	ContentType  string // e.g. "image/png"; backends sniff if empty
	CacheControl string // e.g. "public, max-age=31536000"
	// ContentEncoding is the value of the Content-Encoding header
	// stored alongside the object (e.g. "gzip").
	ContentEncoding string
	// Metadata is the user-defined key/value pairs persisted with the
	// object. Keys are normalised lower-case by S3; do not rely on case.
	Metadata map[string]string
}

// ErrNotFound is returned by Get / Stat when the requested key does
// not exist. Use errors.Is(err, storage.ErrNotFound) to test for it.
var ErrNotFound = errors.New("storage: object not found")
