package cache

import (
	"context"
	"time"
)

// NoopCache is a Cache implementation that silently discards all writes and
// returns "miss" for every read. It is useful in tests and in optional module
// paths where caching is not required.
type NoopCache struct{}

// Get always returns (nil, false).
func (NoopCache) Get(_ context.Context, _ string) ([]byte, bool) { return nil, false }

// Set discards the value and returns nil.
func (NoopCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error { return nil }

// Delete is a no-op.
func (NoopCache) Delete(_ context.Context, _ string) error { return nil }

// Flush is a no-op.
func (NoopCache) Flush(_ context.Context) error { return nil }
