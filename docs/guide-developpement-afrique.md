# Guide de Développement - Contexte Africain

## Principes de Design pour le Marché Africain

### 1. Mobile-First & 3G-First

**Statistiques Sénégal 2024:**
- 95% des utilisateurs sur mobile
- 70% connexions 3G
- 25% connexions 4G
- 5% WiFi/fixe

**Implications techniques:**

```typescript
// Next.js - Optimisation bundle
// next.config.js
module.exports = {
  // Code splitting agressif
  experimental: {
    optimizeCss: true,
    optimizePackageImports: ['@radix-ui/react-icons'],
  },

  // Compression
  compress: true,

  // Image optimization
  images: {
    formats: ['image/webp'],
    deviceSizes: [640, 750, 828, 1080], // Mobile sizes only
    minimumCacheTTL: 60 * 60 * 24 * 7, // 1 week
  },

  // PWA
  pwa: {
    dest: 'public',
    disable: process.env.NODE_ENV === 'development',
    register: true,
    scope: '/',
    sw: '/sw.js',
  }
}
```

**Performance Budget:**
- JS bundle initial: < 150KB (gzipped)
- CSS initial: < 50KB (gzipped)
- Images: < 100KB each (WebP)
- Fonts: System fonts only (no custom fonts)
- Total page weight: < 500KB

### 2. Offline-First Strategy

**Service Worker Stratégie:**

```javascript
// public/sw.js - Service Worker
const CACHE_NAME = 'immobilier-v1';
const OFFLINE_CACHE = 'immobilier-offline-v1';

// Cache essential assets
const CACHE_URLS = [
  '/',
  '/offline',
  '/manifest.json',
  '/icons/icon-192x192.png',
];

// Install - cache essential resources
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(CACHE_URLS);
    })
  );
});

// Fetch strategy: Network-first, fallback to cache
self.addEventListener('fetch', (event) => {
  const { request } = event;

  // API requests: Network-first
  if (request.url.includes('/api/')) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          // Clone and cache successful responses
          if (response.ok) {
            const responseClone = response.clone();
            caches.open(CACHE_NAME).then((cache) => {
              cache.put(request, responseClone);
            });
          }
          return response;
        })
        .catch(() => {
          // Fallback to cache
          return caches.match(request);
        })
    );
  }

  // Static assets: Cache-first
  else {
    event.respondWith(
      caches.match(request).then((cached) => {
        return cached || fetch(request);
      })
    );
  }
});

// Background sync for offline bookings
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-bookings') {
    event.waitUntil(syncOfflineBookings());
  }
});

async function syncOfflineBookings() {
  const db = await openDB('offline-queue');
  const bookings = await db.getAll('bookings');

  for (const booking of bookings) {
    try {
      await fetch('/api/v1/bookings', {
        method: 'POST',
        body: JSON.stringify(booking),
        headers: { 'Content-Type': 'application/json' },
      });
      await db.delete('bookings', booking.id);
    } catch (err) {
      console.error('Sync failed:', err);
    }
  }
}
```

### 3. Paiement Mobile - Best Practices

**Séquence complète:**

```typescript
// Frontend - Initiate payment
async function initiateWavePayment(bookingId: string, amount: number) {
  // 1. Create booking
  const booking = await apiClient.post('/bookings', {
    listing_id: listingId,
    check_in: checkIn,
    check_out: checkOut,
  });

  // 2. Initiate payment
  const payment = await apiClient.post('/payments', {
    booking_id: booking.id,
    amount: amount,
    payment_method: 'wave',
    phone_number: user.phone,
  });

  // 3. Redirect to Wave
  // Wave will redirect back to return_url
  window.location.href = payment.payment_url;

  // Alternative: Show QR code for mobile app users
  if (isMobileApp) {
    showQRCode(payment.qr_code_url);
  }
}

// Handle return from Wave
async function handlePaymentReturn() {
  const urlParams = new URLSearchParams(window.location.search);
  const bookingId = urlParams.get('booking_id');

  if (!bookingId) return;

  // Poll payment status (webhook may take time)
  const maxAttempts = 10;
  let attempts = 0;

  const pollInterval = setInterval(async () => {
    attempts++;

    const booking = await apiClient.get(`/bookings/${bookingId}`);

    if (booking.status === 'confirmed') {
      clearInterval(pollInterval);
      router.push(`/bookings/${bookingId}/success`);
    }

    if (attempts >= maxAttempts) {
      clearInterval(pollInterval);
      // Show manual verification message
      showMessage('Vérification en cours. Vous recevrez une confirmation par SMS.');
    }
  }, 3000); // Poll every 3 seconds
}
```

**Backend - Webhook handler:**

```go
// Idempotency is CRITICAL
func (h *WebhookHandler) HandleWaveWebhook(c *fiber.Ctx) error {
    // 1. Verify signature (MUST be first)
    signature := c.Get("X-Wave-Signature")
    if !verifySignature(c.Body(), signature) {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid signature"})
    }

    // 2. Parse payload
    var webhook WaveWebhook
    if err := c.BodyParser(&webhook); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid payload"})
    }

    // 3. Check idempotency (transaction_id UNIQUE)
    exists, err := h.db.Exec(`
        INSERT INTO payment_webhooks (provider, transaction_id, payload_hash, raw_payload)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (provider, transaction_id) DO NOTHING
    `, "wave", webhook.TransactionID, hash(c.Body()), c.Body())

    if err != nil {
        // Log but return 200 (prevent retry storm)
        log.Error("Failed to store webhook:", err)
        return c.Status(200).JSON(fiber.Map{"status": "error_logged"})
    }

    if exists.RowsAffected() == 0 {
        // Already processed
        return c.Status(200).JSON(fiber.Map{"status": "already_processed"})
    }

    // 4. RETURN 200 OK IMMEDIATELY (< 2s SLA)
    // Process async
    go h.processWebhookAsync(webhook)

    return c.Status(200).JSON(fiber.Map{"status": "received"})
}
```

### 4. Images & Médias

**Pipeline d'optimisation:**

```go
// Image processing service
type ImageProcessor struct {
    s3Client     *s3.Client
    maxWidth     int
    maxHeight    int
    quality      int
    targetSizeKB int
}

func (p *ImageProcessor) ProcessUpload(ctx context.Context, file multipart.File) (string, error) {
    // 1. Decode image
    img, format, err := image.Decode(file)
    if err != nil {
        return "", fmt.Errorf("decode image: %w", err)
    }

    // 2. Resize to max dimensions (1200x800)
    if img.Bounds().Dx() > p.maxWidth || img.Bounds().Dy() > p.maxHeight {
        img = resize.Thumbnail(
            uint(p.maxWidth),
            uint(p.maxHeight),
            img,
            resize.Lanczos3,
        )
    }

    // 3. Convert to WebP
    var buf bytes.Buffer
    if err := webp.Encode(&buf, img, &webp.Options{
        Quality: float32(p.quality),
    }); err != nil {
        return "", fmt.Errorf("encode webp: %w", err)
    }

    // 4. Check size, reduce quality if needed
    attempts := 0
    targetSize := p.targetSizeKB * 1024 // 100KB default

    for buf.Len() > targetSize && attempts < 3 {
        attempts++
        buf.Reset()
        quality := p.quality - (attempts * 10)
        webp.Encode(&buf, img, &webp.Options{
            Quality: float32(quality),
        })
    }

    // 5. Upload to S3/R2
    key := fmt.Sprintf("listings/%s.webp", uuid.New().String())
    _, err = p.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:      aws.String("immobilier-photos"),
        Key:         aws.String(key),
        Body:        bytes.NewReader(buf.Bytes()),
        ContentType: aws.String("image/webp"),
        CacheControl: aws.String("public, max-age=31536000"), // 1 year
    })

    if err != nil {
        return "", fmt.Errorf("upload to s3: %w", err)
    }

    // 6. Generate thumbnail (300x200 for list view)
    thumbnail := resize.Thumbnail(300, 200, img, resize.Lanczos3)
    var thumbBuf bytes.Buffer
    webp.Encode(&thumbBuf, thumbnail, &webp.Options{Quality: 70})

    thumbKey := fmt.Sprintf("listings/%s-thumb.webp", uuid.New().String())
    p.s3Client.PutObject(ctx, &s3.PutObjectInput{
        Bucket:       aws.String("immobilier-photos"),
        Key:          aws.String(thumbKey),
        Body:         bytes.NewReader(thumbBuf.Bytes()),
        ContentType:  aws.String("image/webp"),
        CacheControl: aws.String("public, max-age=31536000"),
    })

    return key, nil
}
```

**Frontend - Progressive image loading:**

```typescript
// components/OptimizedImage.tsx
import Image from 'next/image';
import { useState } from 'react';

interface Props {
  src: string;
  alt: string;
  width: number;
  height: number;
}

export function OptimizedImage({ src, alt, width, height }: Props) {
  const [isLoading, setIsLoading] = useState(true);

  // Blurhash ou LQIP (Low Quality Image Placeholder)
  const placeholder = generateBlurhash(src);

  return (
    <div className="relative overflow-hidden">
      {/* Placeholder blur */}
      {isLoading && (
        <div
          className="absolute inset-0 bg-gray-200 animate-pulse"
          style={{ filter: 'blur(20px)' }}
        />
      )}

      {/* Actual image */}
      <Image
        src={src}
        alt={alt}
        width={width}
        height={height}
        loading="lazy"
        quality={80}
        onLoadingComplete={() => setIsLoading(false)}
        className={`transition-opacity duration-300 ${
          isLoading ? 'opacity-0' : 'opacity-100'
        }`}
      />
    </div>
  );
}
```

### 5. Search Geo avec PostGIS

**Query optimisée pour recherche par rayon:**

```sql
-- Index GIST déjà créé sur location (geography)
-- CREATE INDEX idx_listings_location ON listings USING GIST(location);

-- Recherche dans un rayon de 5km autour de Dakar
SELECT
    l.id,
    l.title,
    l.price_per_month,
    l.bedrooms,
    l.bathrooms,
    ST_Distance(
        l.location,
        ST_SetSRID(ST_MakePoint(-17.4441, 14.6937), 4326)::geography
    ) / 1000 AS distance_km
FROM listings l
WHERE
    l.status = 'published'
    AND ST_DWithin(
        l.location,
        ST_SetSRID(ST_MakePoint(-17.4441, 14.6937), 4326)::geography,
        5000 -- 5km in meters
    )
    AND l.price_per_month BETWEEN 200000 AND 500000
ORDER BY distance_km ASC
LIMIT 20;
```

**Go implementation:**

```go
type GeoSearchParams struct {
    Latitude    float64
    Longitude   float64
    RadiusKM    float64
    MinPrice    int
    MaxPrice    int
    Limit       int
}

func (r *ListingRepository) SearchByGeo(ctx context.Context, params GeoSearchParams) ([]Listing, error) {
    query := `
        SELECT
            id,
            title,
            price_per_month,
            bedrooms,
            bathrooms,
            ST_X(location::geometry) as longitude,
            ST_Y(location::geometry) as latitude,
            ST_Distance(
                location,
                ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography
            ) / 1000 AS distance_km
        FROM listings
        WHERE
            status = 'published'
            AND deleted_at IS NULL
            AND ST_DWithin(
                location,
                ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
                $3 * 1000
            )
            AND price_per_month BETWEEN $4 AND $5
        ORDER BY distance_km ASC
        LIMIT $6
    `

    rows, err := r.db.QueryContext(
        ctx,
        query,
        params.Longitude, params.Latitude,
        params.RadiusKM,
        params.MinPrice, params.MaxPrice,
        params.Limit,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var listings []Listing
    for rows.Next() {
        var l Listing
        var distanceKM float64

        err := rows.Scan(
            &l.ID,
            &l.Title,
            &l.PricePerMonth,
            &l.Bedrooms,
            &l.Bathrooms,
            &l.Longitude,
            &l.Latitude,
            &distanceKM,
        )
        if err != nil {
            return nil, err
        }

        l.DistanceKM = distanceKM
        listings = append(listings, l)
    }

    return listings, nil
}
```

### 6. Double-Booking Prevention

**Transaction avec SERIALIZABLE isolation:**

```go
func (r *BookingRepository) CreateBooking(ctx context.Context, booking *Booking) error {
    // Start transaction with SERIALIZABLE isolation
    tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
    })
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()

    // Check availability (exclusion constraint will catch overlaps)
    query := `
        INSERT INTO bookings (
            id, listing_id, tenant_id, status,
            check_in, check_out, date_range,
            total_amount, deposit
        )
        VALUES (
            $1, $2, $3, $4,
            $5, $6, daterange($5, $6, '[]'),
            $7, $8
        )
    `

    _, err = tx.ExecContext(
        ctx,
        query,
        booking.ID,
        booking.ListingID,
        booking.TenantID,
        "pending",
        booking.CheckIn,
        booking.CheckOut,
        booking.TotalAmount,
        booking.Deposit,
    )

    if err != nil {
        if isExclusionViolation(err) {
            return fmt.Errorf("dates not available (overlapping booking)")
        }
        return fmt.Errorf("insert booking: %w", err)
    }

    // Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("commit tx: %w", err)
    }

    return nil
}

func isExclusionViolation(err error) bool {
    // PostgreSQL error code 23P01 = exclusion_violation
    return strings.Contains(err.Error(), "23P01") ||
           strings.Contains(err.Error(), "conflicting key value violates exclusion constraint")
}
```

### 7. Testing avec Contraintes Réseau

**Simuler connexion 3G:**

```javascript
// tests/e2e/network.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Performance 3G', () => {
  test.use({
    // Simulate 3G: 400ms RTT, 400kbps download, 400kbps upload
    offline: false,
    downloadThroughput: (400 * 1024) / 8, // 400 Kbps
    uploadThroughput: (400 * 1024) / 8,
    latency: 400, // 400ms
  });

  test('homepage loads in < 5s on 3G', async ({ page }) => {
    const start = Date.now();

    await page.goto('https://immobilier.com');

    // Wait for LCP (Largest Contentful Paint)
    await page.waitForSelector('[data-testid="main-content"]');

    const loadTime = Date.now() - start;

    expect(loadTime).toBeLessThan(5000); // < 5 seconds
  });

  test('search works offline', async ({ page, context }) => {
    // Load page online
    await page.goto('https://immobilier.com');

    // Go offline
    await context.setOffline(true);

    // Search should work from cache
    await page.fill('[data-testid="search-input"]', 'Dakar');
    await page.click('[data-testid="search-button"]');

    // Should show cached results
    await expect(page.locator('[data-testid="listing-card"]')).toHaveCount(20);
  });
});
```

---

**Version:** 1.0
**Dernière mise à jour:** 7 novembre 2025
**Public:** Équipe de développement Immobilier-SaaS
