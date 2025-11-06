# Déploiement MVP - Budget $50/mois

## Objectif: Production-Ready en 6 Semaines

**Budget total:** $25/mois (marge $25 pour scaling)

**Performance targets:**
- < 200ms p99 latency (3G)
- 1,000 listings
- 500 users actifs
- 99.9% uptime

## Stack Infrastructure MVP

### 1. Fly.io (Compute) - $10/mois

**Configuration:**
- 1x shared-cpu-1x (256MB RAM)
- Region: Johannesburg (closest to Dakar)
- Hot-reload disabled en prod
- Autoscaling: 1 min, 2 max

```bash
# fly.toml configuration
app = "immobilier-api-mvp"
primary_region = "jnb" # Johannesburg

[[services]]
  http_checks = []
  internal_port = 8080
  protocol = "tcp"
  script_checks = []

  [services.concurrency]
    hard_limit = 25
    soft_limit = 20
    type = "connections"

  [[services.ports]]
    force_https = true
    handlers = ["http"]
    port = 80

  [[services.ports]]
    handlers = ["tls", "http"]
    port = 443

  [[services.tcp_checks]]
    grace_period = "1s"
    interval = "15s"
    restart_limit = 0
    timeout = "2s"

[mounts]
  destination = "/data"
  source = "data_volume"

[[vm]]
  cpu_kind = "shared"
  cpus = 1
  memory_mb = 256
```

**Deploy:**
```bash
fly deploy --ha=false --vm-size shared-cpu-1x --vm-memory 256
```

### 2. Fly.io Postgres - $15/mois

**Configuration:**
- 1x shared-cpu-1x (256MB RAM)
- 10GB storage
- Daily automated backups
- No read replica (yet)

```bash
fly postgres create \
  --name immobilier-db-mvp \
  --initial-cluster-size 1 \
  --vm-size shared-cpu-1x \
  --volume-size 10 \
  --region jnb
```

**Optimizations:**
```sql
-- PostgreSQL tuning for 256MB RAM
ALTER SYSTEM SET shared_buffers = '64MB';
ALTER SYSTEM SET effective_cache_size = '192MB';
ALTER SYSTEM SET maintenance_work_mem = '16MB';
ALTER SYSTEM SET work_mem = '2MB';
ALTER SYSTEM SET max_connections = 20;

-- Enable query logging (queries > 500ms)
ALTER SYSTEM SET log_min_duration_statement = 500;

-- Restart required
SELECT pg_reload_conf();
```

### 3. Upstash Redis - $0/mois (Free Tier)

**Limits:**
- 10,000 commands/day
- 256MB RAM
- Global replication

**Usage strategy:**
- Cache listings (TTL 60s)
- Rate limiting
- Session storage

```bash
# .env
REDIS_URL=rediss://:password@global.upstash.io:6379
```

**Cache keys:**
```
listings:search:{hash}    TTL: 60s
user:session:{user_id}    TTL: 15min
rate:limit:{ip}           TTL: 60s
```

### 4. Cloudflare R2 - $0/mois (Free Tier)

**Limits:**
- 10GB storage/month
- Class A: 1M requests/month
- Class B: 10M requests/month
- Egress: Free

**Configuration:**
```bash
# R2 bucket
wrangler r2 bucket create immobilier-photos-mvp

# Public access via custom domain
# photos.immobilier.com → R2
```

**Image optimization:**
- Resize server-side before upload
- WebP conversion
- Target: < 100KB per image
- Progressive JPEG fallback

### 5. Cloudflare CDN - $0/mois (Free)

**Features:**
- Unlimited bandwidth
- DDoS protection
- Web Application Firewall (WAF)
- Page Rules (3 free)

**Configuration:**
```nginx
# Cache everything
Cache-Control: public, max-age=3600, s-maxage=3600
```

**Page Rules:**
1. `*.immobilier.com/api/*` - Bypass cache
2. `*.immobilier.com/static/*` - Cache everything (1 year)
3. `*.immobilier.com/_next/*` - Cache everything (1 year)

### 6. Brevo (ex-SendinBlue) - $0/mois

**Limits:**
- 300 emails/day
- Transactional only

**Usage:**
- Welcome emails
- Booking confirmations
- Payment receipts
- Password resets

### 7. Better Uptime - $0/mois

**Features:**
- 1 monitor
- 3-minute checks
- Public status page
- Email alerts

**Monitor:**
- URL: https://api.immobilier.com/health
- Interval: 3min
- Alert: > 5min downtime

### 8. Sentry - $0/mois (Free)

**Limits:**
- 5,000 events/month
- 1 project

**Configuration:**
```go
sentry.Init(sentry.ClientOptions{
    Dsn: os.Getenv("SENTRY_DSN"),
    Environment: "production",
    SampleRate: 0.5, // 50% sampling to stay under limit
    TracesSampleRate: 0.1, // 10% traces
})
```

## Total Cost Breakdown

| Service | Plan | Cost/Month | Key Limits |
|---------|------|------------|------------|
| Fly.io API | shared-cpu-1x | $10 | 256MB RAM, 1 vCPU |
| Fly.io Postgres | shared-cpu-1x | $15 | 256MB RAM, 10GB |
| Upstash Redis | Free | $0 | 10k cmds/day |
| Cloudflare R2 | Free | $0 | 10GB storage |
| Cloudflare CDN | Free | $0 | Unlimited |
| Brevo Email | Free | $0 | 300/day |
| Better Uptime | Free | $0 | 1 monitor |
| Sentry | Free | $0 | 5k events/month |
| **TOTAL** | | **$25** | |

## Performance Optimizations

### 1. Database Queries

**Problème:** Queries lentes dépassent 200ms target

**Solutions:**
```sql
-- Index covering pour search
CREATE INDEX idx_listings_search
ON listings (city, status, price_per_month)
WHERE deleted_at IS NULL;

-- Pagination cursor-based (pas d'OFFSET)
SELECT * FROM listings
WHERE id > $1 AND status = 'published'
ORDER BY id
LIMIT 20;

-- Materialized view pour stats admin
CREATE MATERIALIZED VIEW admin_stats AS
SELECT
  COUNT(*) as total_listings,
  COUNT(*) FILTER (WHERE status = 'published') as published,
  AVG(price_per_month) as avg_price
FROM listings;

REFRESH MATERIALIZED VIEW CONCURRENTLY admin_stats;
```

### 2. API Response Size

**Problème:** Payloads trop gros pour 3G

**Solutions:**
```go
// Pagination stricte
type ListingsResponse struct {
    Listings   []ListingSummary `json:"listings"` // pas de Listing complet
    Cursor     string           `json:"cursor"`
    HasMore    bool             `json:"has_more"`
}

// Field selection (GraphQL-like)
// GET /listings?fields=id,title,price,images.primary
func selectFields(fields []string) {
    // Return only requested fields
}

// Compression Brotli niveau 6
app.Use(compress.New(compress.Config{
    Level: compress.LevelBestSpeed,
}))
```

### 3. Image Serving

**Problème:** Images trop lourdes (> 500KB)

**Solutions:**
```go
// Resize + WebP conversion avant upload
func optimizeImage(img image.Image) ([]byte, error) {
    // Resize to max 1200x800
    resized := resize.Thumbnail(1200, 800, img, resize.Lanczos3)

    // Convert to WebP (quality 80)
    var buf bytes.Buffer
    if err := webp.Encode(&buf, resized, &webp.Options{
        Quality: 80,
    }); err != nil {
        return nil, err
    }

    // Ensure < 100KB
    if buf.Len() > 100*1024 {
        // Re-encode with lower quality
        buf.Reset()
        webp.Encode(&buf, resized, &webp.Options{Quality: 60})
    }

    return buf.Bytes(), nil
}
```

### 4. Cache Strategy

**Problème:** Upstash limit 10k cmds/day

**Solutions:**
```go
// Cache ONLY hot paths
// 1. Listing search results (60s TTL)
// 2. User sessions (15min TTL)
// 3. Rate limiting (1min TTL)

// Cache key structure
type CacheKey struct {
    Prefix string
    ID     string
    TTL    time.Duration
}

var CacheKeys = struct {
    ListingSearch CacheKey
    UserSession   CacheKey
    RateLimit     CacheKey
}{
    ListingSearch: {Prefix: "search", TTL: 60 * time.Second},
    UserSession:   {Prefix: "session", TTL: 15 * time.Minute},
    RateLimit:     {Prefix: "rate", TTL: 60 * time.Second},
}

// Estimate: 5k searches/day = 5k GET + 5k SET = 10k cmds
```

### 5. Background Jobs

**Problème:** Pas de budget pour queue dédiée

**Solutions:**
```go
// Utiliser Fly.io volumes pour queue simple
type JobQueue struct {
    mu   sync.Mutex
    jobs []Job
}

// Ou: Postgres comme queue (pas optimal mais gratuit)
CREATE TABLE job_queue (
    id UUID PRIMARY KEY,
    job_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    run_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    attempts INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_jobs_pending
ON job_queue (run_at)
WHERE status = 'pending';
```

## Monitoring & Alerts

### 1. Health Checks

**Endpoint:**
```go
// GET /health
func HealthCheck(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
    defer cancel()

    health := fiber.Map{
        "status": "ok",
        "timestamp": time.Now().Unix(),
    }

    // Check database
    if err := db.PingContext(ctx); err != nil {
        health["database"] = "unhealthy"
        health["status"] = "degraded"
    } else {
        health["database"] = "ok"
    }

    // Check Redis
    if _, err := redis.Ping(ctx).Result(); err != nil {
        health["cache"] = "unhealthy"
    } else {
        health["cache"] = "ok"
    }

    return c.JSON(health)
}
```

### 2. Error Tracking

**Sentry Integration:**
```go
// Capture errors (50% sampling)
sentry.CaptureException(err)

// Breadcrumbs
sentry.AddBreadcrumb(&sentry.Breadcrumb{
    Category: "query",
    Message:  "Searching listings",
    Level:    sentry.LevelInfo,
})
```

### 3. Performance Monitoring

**Custom Metrics (Prometheus format):**
```go
// Expose /metrics endpoint
import "github.com/prometheus/client_golang/prometheus"

var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Buckets: []float64{0.1, 0.2, 0.5, 1.0, 2.0},
        },
        []string{"method", "endpoint", "status"},
    )
)

// Track in middleware
start := time.Now()
// ... handle request ...
duration := time.Since(start).Seconds()
requestDuration.WithLabelValues(method, path, status).Observe(duration)
```

## Scaling Plan

### Week 1-2: MVP (Current)
- 1 VM, 1 DB, Free tier everything
- Cost: $25/month

### Week 3-6: Beta Launch
- 2 VMs (auto-scale)
- Upstash Pro ($20/month)
- Cost: $60/month

### Month 3-4: Production
- 3 VMs multi-region
- Dedicated Postgres (1GB RAM)
- R2 paid tier
- Cost: $150/month

### Month 6+: Scale
- 5+ VMs
- Read replicas
- Dedicated Redis
- Cost: $400/month

## Backup & Recovery

### Database Backups

**Fly.io automated:**
- Daily snapshots (retained 7 days)
- Restore time: < 5 minutes

**Manual backup:**
```bash
# Export to S3
fly postgres connect -a immobilier-db-mvp
pg_dump immobilier_saas | gzip > backup_$(date +%Y%m%d).sql.gz
aws s3 cp backup_*.sql.gz s3://backups/
```

### Disaster Recovery

**RTO:** 15 minutes (Recovery Time Objective)
**RPO:** 24 hours (Recovery Point Objective)

**Procedure:**
1. Restore latest Fly.io snapshot
2. Re-deploy API from GitHub
3. Update DNS if needed
4. Verify /health endpoint
5. Monitor errors in Sentry

## Security Checklist

- [ ] HTTPS enforced (Cloudflare)
- [ ] JWT RS256 keys in env secrets
- [ ] Rate limiting (100 req/min/IP)
- [ ] CORS restricted to domain
- [ ] SQL injection protected (parameterized queries)
- [ ] XSS protection (CSP headers)
- [ ] Webhook signature verification
- [ ] Secrets in Fly secrets (not in git)
- [ ] Database not publicly accessible
- [ ] Regular dependency updates (Dependabot)

---

**Version:** 1.0 - MVP Budget
**Dernière mise à jour:** 7 novembre 2025
**Prochaine révision:** After 500 active users
