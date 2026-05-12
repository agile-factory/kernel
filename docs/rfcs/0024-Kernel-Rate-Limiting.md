# _RFC #0024:_ Kernel `ratelimit` — Rate Limiting & Quotas API

**date:** `2026 May 12`

> ⚠️ **DEFERRED pour la surface BaaS (`/public/v1/*`).** Le `kernel/ratelimit/` est **implémenté en P1** pour protéger `/customer/v1/auth/*` (brute-force end-customers). Sa réutilisation pour le BaaS (par api_key) est prévue en P2+.

---

## **Status:** `DRAFT`

---

## **Summary**

Ce RFC définit le package `kernel/ratelimit/` : port abstrait `Limiter` + deux implémentations (in-memory token bucket en v1, Redis sliding window en v2), et le middleware HTTP `WithRateLimit` paramétrable par extracteur de clé.

---

## **Motivation**

Trois surfaces nécessitent du rate-limiting :

| Surface | Priorité | Raison |
|---|---|---|
| `/customer/v1/auth/*` | **P1 — implémenté** | Protection brute-force login end-customers |
| `/api/v1/*` | P1 — souple | Fairness entre tenants (100 rpm/user) |
| `/public/v1/*` (BaaS) | **P2+** | Quotas par plan, par api_key |

---

## **Proposal**

### 1. Port abstrait `Limiter`

```go
// kernel/ratelimit/limiter.go
package ratelimit

import "context"

type Limiter interface {
    // Allow retourne true si la requête est autorisée.
    // key = identifiant de la bucket (user_id, tenant_id, ip, api_key_id...)
    // cost = poids de la requête (généralement 1)
    Allow(ctx context.Context, key string, cost int) (bool, error)

    // Status retourne les compteurs courants (pour les headers X-RateLimit-*)
    Status(ctx context.Context, key string) (LimitStatus, error)
}

type LimitStatus struct {
    Limit     int
    Remaining int
    ResetAt   time.Time
}
```

### 2. Implémentation v1 — In-Memory Token Bucket

```go
// kernel/ratelimit/memory_limiter.go
type MemoryLimiter struct {
    rate     int           // tokens ajoutés par seconde
    burst    int           // capacité maximale de la bucket
    buckets  sync.Map      // key → *tokenBucket
    mu       sync.Mutex
}

func NewMemoryLimiter(rate, burst int) *MemoryLimiter
```

Adapté pour : instances single-process, tests, `/customer/v1/auth/*` (5 rpm par IP).

**Limitation** : non partagé entre répliques. En multi-instance, les limites sont par-processus (moins strict). Acceptable pour P1.

### 3. Implémentation v2 — Redis Sliding Window

```go
// kernel/ratelimit/redis_limiter.go
type RedisLimiter struct {
    client  redis.UniversalClient
    window  time.Duration
    limit   int
}

func NewRedisLimiter(client redis.UniversalClient, window time.Duration, limit int) *RedisLimiter
```

Script Lua atomique : `ZADD` + `ZREMRANGEBYSCORE` + `ZCARD` dans la même opération. Partagé entre toutes les répliques.

### 4. Middleware HTTP `WithRateLimit`

```go
// delivery/http/middleware/ratelimit.go
package middleware

type KeyExtractor func(r *http.Request) string

// Extracteurs fournis
var (
    ExtractByIP       KeyExtractor = func(r *http.Request) string { return realIP(r) }
    ExtractByUser     KeyExtractor = func(r *http.Request) string { return userIDFromCtx(r.Context()) }
    ExtractByTenant   KeyExtractor = func(r *http.Request) string { return tenantIDFromCtx(r.Context()) }
    ExtractByAPIKey   KeyExtractor = func(r *http.Request) string { return apiKeyIDFromCtx(r.Context()) }
)

func WithRateLimit(limiter ratelimit.Limiter, extract KeyExtractor) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := extract(r)
            ok, _ := limiter.Allow(r.Context(), key, 1)
            if !ok {
                status, _ := limiter.Status(r.Context(), key)
                w.Header().Set("X-RateLimit-Limit", strconv.Itoa(status.Limit))
                w.Header().Set("X-RateLimit-Remaining", "0")
                w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(status.ResetAt.Unix(), 10))
                w.Header().Set("Retry-After", strconv.FormatInt(time.Until(status.ResetAt).Seconds(), 10))
                http.Error(w, `{"error":"rate_limit_exceeded"}`, http.StatusTooManyRequests)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### 5. Usage P1 — protection `/customer/v1/auth/*`

```go
// Dans router B2B2C (activé si B2B2C_ENABLED=true)
authLimiter := ratelimit.NewMemoryLimiter(rate: 5, burst: 5)  // 5 rpm par IP
customerMux.Handle("POST /customer/v1/auth/login",
    middleware.WithRateLimit(authLimiter, middleware.ExtractByIP)(loginHandler))
customerMux.Handle("POST /customer/v1/auth/otp/request",
    middleware.WithRateLimit(authLimiter, middleware.ExtractByIP)(otpHandler))
```

### 6. Usage P2+ — quotas BaaS par API Key

```go
// Terrain préparé — activé avec BAAS_ENABLED=true
planLimiter := ratelimit.NewRedisLimiter(redisClient, time.Minute, planRPM)
publicMux.Handle("/public/v1/",
    middleware.WithAPIKey(apiKeyRepo)(
        middleware.WithRateLimit(planLimiter, middleware.ExtractByAPIKey)(publicHandler)))
```

---

## **Références**

- [RFC-0022 — Platform API Keys](0022-Platform-APIKey-PAT-Scopes.md)
- [RFC-0025 — Surface Publique BaaS](0025-Surface-Publique-BaaS-Versioning.md)
- [GitHub API Rate Limiting — headers convention](https://docs.github.com/en/rest/overview/rate-limits-for-the-rest-api)
