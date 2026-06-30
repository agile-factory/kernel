// Package cache provides an in-memory cache with TTL support and a GC goroutine
// that periodically evicts expired entries. The Cache interface is minimal and
// intentionally technology-agnostic so callers can swap in Redis or Memcached
// without changing application code.
//
// Usage:
//
//	c := cache.New(cache.Config{GCInterval: 5 * time.Minute})
//	defer c.Stop()
//
//	if err := c.Set(ctx, "my-key", []byte("value"), 30*time.Second); err != nil {
//	    // handle
//	}
//	if v, ok := c.Get(ctx, "my-key"); ok {
//	    fmt.Println(string(v))
//	}
package cache

import (
	"context"
	"sync"
	"time"
)

// Cache is the minimal interface for key/value stores with TTL semantics.
type Cache interface {
	// Get retrieves a value by key. Returns (nil, false) when the key is
	// absent or has expired.
	Get(ctx context.Context, key string) ([]byte, bool)

	// Set stores value under key with the given TTL. A zero TTL means the
	// entry never expires.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete removes the entry for key. It is not an error if the key does
	// not exist.
	Delete(ctx context.Context, key string) error

	// Flush removes all entries from the cache.
	Flush(ctx context.Context) error
}

// Config holds tuning parameters for InMemoryCache.
type Config struct {
	// GCInterval controls how often the background goroutine sweeps the map
	// looking for expired entries. Defaults to 5 minutes when zero.
	GCInterval time.Duration
}

// entry is a single cached value with its expiry timestamp.
type entry struct {
	value     []byte
	expiresAt time.Time // zero means never-expire
}

// expired reports whether the entry has passed its TTL.
func (e entry) expired() bool {
	return !e.expiresAt.IsZero() && time.Now().After(e.expiresAt)
}

// InMemoryCache is a goroutine-safe in-memory cache backed by sync.Map.
// Call Stop() to release the background GC goroutine. Stop is idempotent
// and safe to call multiple times (e.g. via defer).
type InMemoryCache struct {
	m    sync.Map
	once sync.Once
	stop chan struct{}
}

// New returns a running InMemoryCache. The caller MUST call Stop() when the
// cache is no longer needed so the GC goroutine exits cleanly.
func New(cfg Config) *InMemoryCache {
	interval := cfg.GCInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	c := &InMemoryCache{
		stop: make(chan struct{}),
	}
	go c.gc(interval)
	return c
}

// gc runs until Stop() is called, sweeping expired entries every interval.
func (c *InMemoryCache) gc(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.m.Range(func(k, v any) bool {
				if e, ok := v.(entry); ok && e.expired() {
					c.m.Delete(k)
				}
				return true
			})
		case <-c.stop:
			return
		}
	}
}

// Stop shuts down the background GC goroutine. It is idempotent — safe to
// call multiple times or from multiple goroutines.
func (c *InMemoryCache) Stop() {
	c.once.Do(func() { close(c.stop) })
}

// Get implements Cache.
func (c *InMemoryCache) Get(_ context.Context, key string) ([]byte, bool) {
	v, ok := c.m.Load(key)
	if !ok {
		return nil, false
	}
	e := v.(entry)
	if e.expired() {
		c.m.Delete(key)
		return nil, false
	}
	// Return a copy so callers cannot mutate internal state.
	out := make([]byte, len(e.value))
	copy(out, e.value)
	return out, true
}

// Set implements Cache.
func (c *InMemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	e := entry{}
	// Store a defensive copy so callers cannot mutate what we hold.
	e.value = make([]byte, len(value))
	copy(e.value, value)
	if ttl > 0 {
		e.expiresAt = time.Now().Add(ttl)
	}
	c.m.Store(key, e)
	return nil
}

// Delete implements Cache.
func (c *InMemoryCache) Delete(_ context.Context, key string) error {
	c.m.Delete(key)
	return nil
}

// Flush implements Cache. It removes all entries regardless of TTL.
func (c *InMemoryCache) Flush(_ context.Context) error {
	c.m.Range(func(k, _ any) bool {
		c.m.Delete(k)
		return true
	})
	return nil
}
