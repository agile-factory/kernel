// Package infrastructure hosts the storage backends. The S3 backend
// here implements kernel/storage.Storage via the minio-go client; it
// targets any service speaking the S3 API (Cloudflare R2, MinIO, AWS S3,
// Backblaze B2, DigitalOcean Spaces).
package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/AgileFactory/kernel/kernel/storage"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config captures everything needed to reach an S3-compatible endpoint.
// Endpoint, AccessKey, SecretKey and Bucket are required; the rest takes
// sensible defaults appropriate for Cloudflare R2 (the v0.1.0 target).
type S3Config struct {
	// Endpoint is the S3 API host without scheme. Examples:
	//   Cloudflare R2 : "<account-id>.r2.cloudflarestorage.com"
	//   AWS S3        : "s3.<region>.amazonaws.com"
	//   MinIO local   : "localhost:9000"
	Endpoint string

	// Region is sent in the AWS signature. R2 uses "auto", AWS uses
	// "eu-west-3" etc., MinIO ignores it. Defaults to "auto".
	Region string

	Bucket    string
	AccessKey string
	SecretKey string

	// UseSSL toggles TLS to the endpoint. Defaults to true (production
	// path); set to false only for local MinIO over plain HTTP.
	UseSSL bool

	// PathStyle forces "https://endpoint/bucket/key" instead of the
	// virtual-host style "https://bucket.endpoint/key". Required by
	// MinIO; harmless on R2 / AWS.
	PathStyle bool
}

// S3Storage implements storage.Storage by delegating to a minio-go
// client. Construct it once at boot and share across goroutines —
// minio.Client is safe for concurrent use.
type S3Storage struct {
	client *minio.Client
	bucket string
}

// NewS3Storage builds the backend and validates that the bucket is
// reachable. Returns an error if credentials are wrong, the endpoint
// is unreachable, or the bucket does not exist.
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("storage/s3: Endpoint is required")
	}
	if cfg.Bucket == "" {
		return nil, errors.New("storage/s3: Bucket is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("storage/s3: AccessKey and SecretKey are required")
	}

	if cfg.Region == "" {
		cfg.Region = "auto" // R2's expected value; AWS rejects this but caller would have set it
	}

	endpoint := stripScheme(cfg.Endpoint)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.UseSSL,
		Region:       cfg.Region,
		BucketLookup: bucketLookup(cfg.PathStyle),
	})
	if err != nil {
		return nil, fmt.Errorf("storage/s3: client init: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("storage/s3: bucket check: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("storage/s3: bucket %q does not exist or is not accessible with the provided credentials", cfg.Bucket)
	}

	return &S3Storage{client: client, bucket: cfg.Bucket}, nil
}

// Put uploads body under key. size is the exact content length; pass
// -1 only when the size is genuinely unknown — minio-go will then
// fall back to multipart with an in-memory buffer per part.
func (s *S3Storage) Put(ctx context.Context, key string, body io.Reader, size int64, opts storage.PutOptions) (storage.ObjectMeta, error) {
	info, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType:     opts.ContentType,
		CacheControl:    opts.CacheControl,
		ContentEncoding: opts.ContentEncoding,
		UserMetadata:    opts.Metadata,
	})
	if err != nil {
		return storage.ObjectMeta{}, fmt.Errorf("storage/s3: put %q: %w", key, err)
	}
	return storage.ObjectMeta{
		Key:          key,
		Size:         info.Size,
		ContentType:  opts.ContentType,
		ETag:         strings.Trim(info.ETag, `"`),
		LastModified: info.LastModified,
	}, nil
}

// Get opens a read stream. Caller MUST Close the returned Object.
func (s *S3Storage) Get(ctx context.Context, key string) (storage.Object, error) {
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return storage.Object{}, fmt.Errorf("storage/s3: get %q: %w", key, err)
	}
	// minio's GetObject is lazy — Stat triggers the real HEAD and gives
	// us a chance to surface a 404 as ErrNotFound BEFORE the caller
	// starts reading.
	info, statErr := obj.Stat()
	if statErr != nil {
		_ = obj.Close()
		if isNoSuchKey(statErr) {
			return storage.Object{}, fmt.Errorf("%w: %s", storage.ErrNotFound, key)
		}
		return storage.Object{}, fmt.Errorf("storage/s3: get %q: %w", key, statErr)
	}
	return storage.Object{
		Meta: storage.ObjectMeta{
			Key:          key,
			Size:         info.Size,
			ContentType:  info.ContentType,
			ETag:         strings.Trim(info.ETag, `"`),
			LastModified: info.LastModified,
		},
		Body: obj,
	}, nil
}

// Delete removes the key. Missing keys are silently OK.
func (s *S3Storage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil && !isNoSuchKey(err) {
		return fmt.Errorf("storage/s3: delete %q: %w", key, err)
	}
	return nil
}

// Exists is a cheap HEAD that swallows 404 into a clean bool.
func (s *S3Storage) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return true, nil
	}
	if isNoSuchKey(err) {
		return false, nil
	}
	return false, fmt.Errorf("storage/s3: exists %q: %w", key, err)
}

// Stat returns metadata for the key, ErrNotFound when absent.
func (s *S3Storage) Stat(ctx context.Context, key string) (storage.ObjectMeta, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if isNoSuchKey(err) {
			return storage.ObjectMeta{}, fmt.Errorf("%w: %s", storage.ErrNotFound, key)
		}
		return storage.ObjectMeta{}, fmt.Errorf("storage/s3: stat %q: %w", key, err)
	}
	return storage.ObjectMeta{
		Key:          key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         strings.Trim(info.ETag, `"`),
		LastModified: info.LastModified,
	}, nil
}

// List converts minio-go's iterator into the storage.ListEntry channel
// shape. A buffer of 16 absorbs slow consumers without blocking pagination.
func (s *S3Storage) List(ctx context.Context, prefix string) (<-chan storage.ListEntry, error) {
	out := make(chan storage.ListEntry, 16)

	go func() {
		defer close(out)
		objects := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		})
		for info := range objects {
			if info.Err != nil {
				select {
				case out <- storage.ListEntry{Err: fmt.Errorf("storage/s3: list %q: %w", prefix, info.Err)}:
				case <-ctx.Done():
				}
				return
			}
			entry := storage.ListEntry{
				Meta: storage.ObjectMeta{
					Key:          info.Key,
					Size:         info.Size,
					ContentType:  info.ContentType,
					ETag:         strings.Trim(info.ETag, `"`),
					LastModified: info.LastModified,
				},
			}
			select {
			case out <- entry:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// ─── helpers ────────────────────────────────────────────────────────────────

// stripScheme drops the http:// or https:// prefix because minio-go's
// New() wants the bare host[:port]. Leaving the scheme in fails with
// "Endpoint url cannot have fully qualified paths".
func stripScheme(endpoint string) string {
	if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
		return u.Host
	}
	return strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
}

func bucketLookup(pathStyle bool) minio.BucketLookupType {
	if pathStyle {
		return minio.BucketLookupPath
	}
	return minio.BucketLookupAuto
}

// isNoSuchKey detects the various "object missing" responses across
// S3-API implementations (AWS uses "NoSuchKey", R2 uses HTTP 404, MinIO
// matches AWS). minio-go exposes this via minio.ErrorResponse.Code.
func isNoSuchKey(err error) bool {
	if err == nil {
		return false
	}
	var resp minio.ErrorResponse
	if errors.As(err, &resp) {
		return resp.Code == "NoSuchKey" || resp.StatusCode == 404
	}
	return false
}
