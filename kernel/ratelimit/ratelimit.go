// Package ratelimit implements a per-key token-bucket rate limiter and an HTTP
// middleware that enforces it. The implementation relies only on the standard
// library — no external dependencies are introduced.
//
// Usage:
//
//	limiter := ratelimit.New(10, 20) // 10 tokens/s, burst of 20
//	mux.Handle("/api/", ratelimit.Middleware(limiter, ratelimit.IPKeyFn)(mux))
package ratelimit

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Limiter is the minimal interface for rate-limiting decisions.
type Limiter interface {
	// Allow returns true when the caller identified by key is permitted to
	// proceed. It returns false when the bucket is empty (rate limit exceeded).
	Allow(key string) bool
}

// bucket holds the mutable state for a single key in the token bucket.
type bucket struct {
	mu       sync.Mutex
	tokens   float64
	lastSeen time.Time
}

// TokenBucketLimiter is a per-key token bucket limiter. Tokens refill at
// rate tokens/second up to a maximum of burst tokens.
type TokenBucketLimiter struct {
	rate  float64 // tokens per second
	burst float64 // maximum bucket capacity
	m     sync.Map
}

// New returns a TokenBucketLimiter that allows rate tokens per second with a
// maximum burst of burst tokens. Both values must be positive.
func New(rate, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:  float64(rate),
		burst: float64(burst),
	}
}

// Allow implements Limiter. It is safe to call concurrently from multiple
// goroutines with different or identical keys.
func (l *TokenBucketLimiter) Allow(key string) bool {
	now := time.Now()

	v, _ := l.m.LoadOrStore(key, &bucket{
		tokens:   l.burst, // new key starts with a full bucket
		lastSeen: now,
	})
	b := v.(*bucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens proportional to elapsed time.
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// rateLimitExceededBody is the JSON body returned on 429 responses.
type rateLimitExceededBody struct {
	Error string `json:"error"`
}

// Middleware returns an HTTP middleware that calls l.Allow(keyFn(r)) for every
// request. When the limiter denies the request a 429 JSON response is sent and
// the next handler is NOT called.
func Middleware(l Limiter, keyFn func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if !l.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(rateLimitExceededBody{Error: "rate_limit_exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IPKeyFn extracts the client IP from X-Forwarded-For (first entry) or falls
// back to RemoteAddr. It is suitable as the keyFn argument to Middleware.
func IPKeyFn(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain a comma-separated list; the leftmost
		// entry is the originating client.
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	// RemoteAddr is "host:port"; strip the port.
	addr := r.RemoteAddr
	if i := strings.LastIndex(addr, ":"); i != -1 {
		return addr[:i]
	}
	return addr
}

// TenantKeyFn extracts the tenant_id segment from paths of the form
// /api/v1/tenants/{tenantId}/... and returns it as the rate-limit key.
// If the path does not match the expected shape, the raw path is returned so
// the request is still subject to rate limiting (under a coarse key).
func TenantKeyFn(r *http.Request) string {
	// Expected prefix: /api/v1/tenants/
	const prefix = "/api/v1/tenants/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return path
	}
	rest := path[len(prefix):]
	if i := strings.Index(rest, "/"); i != -1 {
		return rest[:i]
	}
	// Path ends at the tenant segment with no trailing slash.
	return rest
}
