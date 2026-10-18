// Package ratelimit implements a per-key token-bucket rate limiter and an HTTP
// middleware that enforces it. The implementation relies only on the standard
// library — no external dependencies are introduced.
//
// Usage:
//
//	limiter := ratelimit.New(10, 20) // 10 tokens/s, burst of 20
//	mux.Handle("/api/", ratelimit.Middleware(limiter, ratelimit.IPKeyFn)(mux))
//
// Behind a reverse proxy, use TrustedProxyIPKeyFn: IPKeyFn deliberately
// ignores X-Forwarded-For, which any client can forge.
package ratelimit

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// sweepInterval is how often Allow prunes idle buckets.
const sweepInterval = time.Minute

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
	rate      float64 // tokens per second
	burst     float64 // maximum bucket capacity
	m         sync.Map
	lastSweep atomic.Int64 // unix nanos of the last idle-bucket sweep
}

// New returns a TokenBucketLimiter that allows rate tokens per second with a
// maximum burst of burst tokens. It panics unless both values are positive.
func New(rate, burst int) *TokenBucketLimiter {
	return NewRate(float64(rate), burst)
}

// NewRate is New with a fractional rate, for limits slower than one request
// per second: NewRate(10.0/60, 10) allows 10 requests per minute.
func NewRate(perSecond float64, burst int) *TokenBucketLimiter {
	if perSecond <= 0 || burst <= 0 {
		panic(fmt.Sprintf("ratelimit: rate and burst must be positive (got %v, %d)", perSecond, burst))
	}
	l := &TokenBucketLimiter{
		rate:  perSecond,
		burst: float64(burst),
	}
	l.lastSweep.Store(time.Now().UnixNano())
	return l
}

// Allow implements Limiter. It is safe to call concurrently from multiple
// goroutines with different or identical keys.
func (l *TokenBucketLimiter) Allow(key string) bool {
	now := time.Now()
	l.maybeSweep(now)

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

// maybeSweep drops buckets idle long enough to have refilled completely.
// Such a bucket is indistinguishable from a fresh one, so removing it never
// changes a decision — it only bounds memory when keys are unbounded (one
// per client IP).
func (l *TokenBucketLimiter) maybeSweep(now time.Time) {
	last := l.lastSweep.Load()
	if now.UnixNano()-last < int64(sweepInterval) || !l.lastSweep.CompareAndSwap(last, now.UnixNano()) {
		return
	}
	idle := time.Duration(l.burst / l.rate * float64(time.Second))
	l.m.Range(func(k, v any) bool {
		b := v.(*bucket)
		b.mu.Lock()
		if now.Sub(b.lastSeen) > idle {
			l.m.Delete(k)
		}
		b.mu.Unlock()
		return true
	})
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

// IPKeyFn keys on the TCP peer address (RemoteAddr). X-Forwarded-For is
// ignored on purpose: it is client-controlled, so trusting it would let
// anyone pick a fresh key per request. Behind a proxy, use
// TrustedProxyIPKeyFn instead.
func IPKeyFn(r *http.Request) string {
	return remoteIP(r)
}

// TrustedProxyIPKeyFn returns a key function for deployments behind reverse
// proxies. X-Forwarded-For is only honoured when the TCP peer is one of the
// trusted prefixes; the client is then the right-most address that is not
// itself a trusted proxy (entries further left are client-supplied).
func TrustedProxyIPKeyFn(trusted []netip.Prefix) func(*http.Request) string {
	isTrusted := func(a netip.Addr) bool {
		for _, p := range trusted {
			if p.Contains(a.Unmap()) {
				return true
			}
		}
		return false
	}
	return func(r *http.Request) string {
		peer := remoteIP(r)
		peerAddr, err := netip.ParseAddr(peer)
		if err != nil || !isTrusted(peerAddr) {
			return peer
		}
		hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
		for i := len(hops) - 1; i >= 0; i-- {
			a, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
			if err != nil {
				break // garbage from here on is client-controlled
			}
			if !isTrusted(a) {
				return a.Unmap().String()
			}
		}
		return peer
	}
}

// remoteIP returns the host part of r.RemoteAddr (IPv6-safe).
func remoteIP(r *http.Request) string {
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// noTenantKey is the shared bucket for requests outside /api/v1/tenants/.
const noTenantKey = "_no_tenant"

// TenantKeyFn extracts the tenant_id segment from paths of the form
// /api/v1/tenants/{tenantId}/... and returns it as the rate-limit key.
// If the path does not match the expected shape, a single shared key is
// returned so the request is still rate limited (coarsely) without letting
// arbitrary paths mint unbounded keys.
func TenantKeyFn(r *http.Request) string {
	// Expected prefix: /api/v1/tenants/
	const prefix = "/api/v1/tenants/"
	path := r.URL.Path
	if !strings.HasPrefix(path, prefix) {
		return noTenantKey
	}
	rest := path[len(prefix):]
	if i := strings.Index(rest, "/"); i != -1 {
		return rest[:i]
	}
	// Path ends at the tenant segment with no trailing slash.
	return rest
}
