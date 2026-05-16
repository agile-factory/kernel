package infrastructure

import (
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestStripScheme(t *testing.T) {
	cases := []struct{ in, want string }{
		{"localhost:9000", "localhost:9000"},
		{"https://localhost:9000", "localhost:9000"},
		{"http://localhost:9000", "localhost:9000"},
		{"https://acme.r2.cloudflarestorage.com", "acme.r2.cloudflarestorage.com"},
		{"s3.eu-west-3.amazonaws.com", "s3.eu-west-3.amazonaws.com"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := stripScheme(c.in); got != c.want {
				t.Errorf("stripScheme(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestBucketLookup(t *testing.T) {
	if got := bucketLookup(true); got != minio.BucketLookupPath {
		t.Errorf("bucketLookup(true) = %v, want BucketLookupPath", got)
	}
	if got := bucketLookup(false); got != minio.BucketLookupAuto {
		t.Errorf("bucketLookup(false) = %v, want BucketLookupAuto", got)
	}
}

func TestIsNoSuchKey(t *testing.T) {
	if isNoSuchKey(nil) {
		t.Error("isNoSuchKey(nil) should be false")
	}
	if isNoSuchKey(errors.New("random")) {
		t.Error("isNoSuchKey(random) should be false")
	}
	if !isNoSuchKey(minio.ErrorResponse{Code: "NoSuchKey"}) {
		t.Error("isNoSuchKey(NoSuchKey) should be true")
	}
	if !isNoSuchKey(minio.ErrorResponse{StatusCode: 404}) {
		t.Error("isNoSuchKey(404) should be true")
	}
	if isNoSuchKey(minio.ErrorResponse{Code: "AccessDenied", StatusCode: 403}) {
		t.Error("isNoSuchKey(403 AccessDenied) should be false")
	}
}
