# ✅ TODO - Implémentation Restante

**Ce qui est fait vs ce qui reste à implémenter**

---

## ✅ FAIT (Architecture & Documentation Complète)

### Architecture & Design
- [x] Architecture générique (Mermaid)
- [x] Architecture optimisée Afrique (Mermaid + détails)
- [x] Schéma database complet (24 tables, PostGIS)
- [x] Décisions architecturales documentées
- [x] Performance targets définis
- [x] Capacity planning 12 mois
- [x] Workflows métier (séquence diagrams)

### Documentation
- [x] README.md principal
- [x] README-AFRICA.md spécifique
- [x] QUICKSTART.md (10min)
- [x] SUMMARY.md (vue d'ensemble)
- [x] LIVRABLES-AFRIQUE.md (liste complète)
- [x] GUIDE-UTILISATION.md (navigation)
- [x] Architecture africaine (40 pages)
- [x] Deployment MVP budget (15 pages)
- [x] Guide développement Afrique (12 pages)
- [x] Security checklist (20 pages)
- [x] Deployment runbook (15 pages)

### Infrastructure
- [x] Docker-compose (10 services)
- [x] Makefile root (40 commands)
- [x] Makefile API (15 commands)
- [x] .gitignore complet
- [x] .env.example (API + Web)
- [x] Dockerfile API (multi-stage)
- [x] Dockerfile Web (multi-stage)
- [x] fly.toml examples

### Database
- [x] Migration 000001: Schema base (14 tables)
- [x] Migration 000002: Extensions Afrique (10 tables)
- [x] Indexes optimisés (60+)
- [x] Triggers (5)
- [x] Functions (3)
- [x] Exclusion constraints (double-booking)

### Backend Go - Structure
- [x] Main API entry point
- [x] Config management
- [x] Domain models (User, Listing, Booking, Payment)
- [x] Middleware (Auth JWT, rate limit, CORS)
- [x] Wave service complet (800 lignes)
- [x] Webhook handler idempotent (400 lignes)

### Frontend Next.js - Structure
- [x] Project structure (app router)
- [x] package.json complet
- [x] next.config.js optimisé
- [x] API client avec token refresh
- [x] Layout root
- [x] PWA configuration

### CI/CD
- [x] Workflow API CI (lint, test, build)
- [x] Workflow Web CI (lint, test, Lighthouse)
- [x] Workflow Deploy production

### Terraform
- [x] Module VPC (networking complet)
- [x] Module RDS (PostgreSQL + PostGIS)
- [x] Structure modules (Redis, S3, ECS, ALB)
- [x] Environment production

---

## ⏳ À IMPLÉMENTER (Code Fonctionnel)

### Backend Go - Handlers

```bash
# À créer dans services/api/internal/handler/

# 1. auth_handler.go - PARTIEL (structure existe)
# Manquent:
- [ ] Register() implementation complète
- [ ] Login() avec password hashing Argon2
- [ ] Logout() avec Redis blacklist token
- [ ] RefreshToken() complet
- [ ] ForgotPassword() avec email
- [ ] ResetPassword() avec token validation
- [ ] VerifyEmail() avec token

# 2. listing_handler.go - PARTIEL
# Manquent:
- [ ] GetListings() avec pagination cursor
- [ ] SearchListings() avec PostGIS < 200ms
- [ ] CreateListing() avec image upload
- [ ] UpdateListing() avec ownership check
- [ ] DeleteListing() soft delete

# 3. booking_handler.go - À CRÉER
- [ ] GetUserBookings()
- [ ] GetBookingByID() avec ownership check
- [ ] CreateBooking() avec exclusion constraint
- [ ] CancelBooking() avec refund logic
- [ ] GetAvailability() pour calendrier

# 4. payment_handler.go - À CRÉER
- [ ] CreatePayment() Wave/OM/Free
- [ ] GetPaymentByID()
- [ ] InitiateRefund()
- [ ] GetPaymentStatus()

# 5. admin_handler.go - À CRÉER
- [ ] GetDashboardStats()
- [ ] ModerateListings()
- [ ] ManageUsers()
- [ ] ReconcilePayments()

# 6. kyc_handler.go - À CRÉER
- [ ] UploadDocument()
- [ ] GetKYCStatus()
- [ ] ReviewDocument() (admin)
```

### Backend Go - Services

```bash
# À créer dans services/api/internal/service/

# 1. auth_service.go - À CRÉER
- [ ] Register() avec password hashing
- [ ] Login() avec JWT generation
- [ ] RefreshToken() validation
- [ ] ForgotPassword() avec email queue
- [ ] ResetPassword() avec token check
- [ ] VerifyEmail() update status

# 2. listing_service.go - À CRÉER
- [ ] CreateListing() validation + S3 upload
- [ ] UpdateListing() avec moderation
- [ ] DeleteListing() soft delete
- [ ] SearchListings() PostGIS query
- [ ] GetListingByID() avec cache
- [ ] IncrementViewCount() async

# 3. booking_service.go - À CRÉER
- [ ] CreateBooking() avec transaction
- [ ] GetAvailability() exclusion check
- [ ] CancelBooking() avec business logic
- [ ] CompleteCheckout() trigger payout

# 4. payment_service.go - À CRÉER
- [ ] CreatePayment() Wave integration
- [ ] ProcessWebhook() idempotency
- [ ] CreateRefund()
- [ ] ReconcileDaily()

# 5. image_service.go - À CRÉER
- [ ] OptimizeUpload() resize + WebP
- [ ] GenerateThumbnail() 300x200
- [ ] ModerateWithAI() Rekognition
- [ ] DeleteImage() S3 cleanup

# 6. kyc_service.go - À CRÉER
- [ ] ProcessDocument() OCR
- [ ] VerifyFaceMatch()
- [ ] UpdateUserStatus()

# 7. notification_service.go - À CRÉER
- [ ] SendEmail() via Brevo
- [ ] SendSMS() via Twilio
- [ ] QueueNotification() NATS
```

### Backend Go - Repositories

```bash
# À créer dans services/api/internal/repository/

# 1. user_repository.go - À CRÉER
- [ ] Create()
- [ ] GetByID()
- [ ] GetByEmail()
- [ ] Update()
- [ ] UpdateKYCStatus()

# 2. listing_repository.go - À CRÉER
- [ ] Create()
- [ ] GetByID()
- [ ] Search() avec PostGIS
- [ ] Update()
- [ ] SoftDelete()
- [ ] IncrementViewCount()

# 3. booking_repository.go - À CRÉER
- [ ] Create() avec transaction SERIALIZABLE
- [ ] GetByID()
- [ ] GetByUser()
- [ ] GetByListing()
- [ ] CheckAvailability() exclusion
- [ ] Update()

# 4. payment_repository.go - À CRÉER
- [ ] Create()
- [ ] GetByID()
- [ ] GetByBooking()
- [ ] StoreWebhook()
- [ ] CheckWebhookProcessed()
- [ ] UpdateStatus()

# 5. payout_repository.go - À CRÉER
- [ ] Create()
- [ ] GetPending()
- [ ] UpdateStatus()
- [ ] GetByLandlord()
```

### Frontend Next.js - Pages

```bash
# À créer dans services/web/app/

# 1. (marketing)/page.tsx - Landing
- [ ] Hero section
- [ ] Search bar
- [ ] Featured listings
- [ ] How it works
- [ ] Testimonials

# 2. (auth)/login/page.tsx
- [ ] Login form
- [ ] OAuth buttons (optional)
- [ ] Forgot password link

# 3. (auth)/register/page.tsx
- [ ] Register form
- [ ] Role selection (tenant/landlord)
- [ ] Terms acceptance

# 4. (dashboard)/listings/page.tsx
- [ ] Listing grid
- [ ] Filters sidebar
- [ ] Search + geo map
- [ ] Pagination

# 5. (dashboard)/listings/[id]/page.tsx
- [ ] Image gallery
- [ ] Details + amenities
- [ ] Map location
- [ ] Calendar availability
- [ ] Booking form
- [ ] Reviews

# 6. (dashboard)/listings/new/page.tsx (landlord)
- [ ] Multi-step form
- [ ] Image upload + crop
- [ ] Map picker
- [ ] Amenities selector
- [ ] Pricing + availability

# 7. (dashboard)/bookings/page.tsx
- [ ] User bookings list
- [ ] Status badges
- [ ] Payment status
- [ ] Cancel button

# 8. (dashboard)/dashboard/page.tsx (landlord)
- [ ] Stats cards
- [ ] Revenue chart
- [ ] Occupancy rate
- [ ] Recent bookings
```

### Frontend Next.js - Components

```bash
# À créer dans services/web/components/

# UI (shadcn/ui)
- [ ] button.tsx
- [ ] card.tsx
- [ ] input.tsx
- [ ] dialog.tsx
- [ ] toast.tsx
- [ ] calendar.tsx
- [ ] select.tsx

# Layout
- [ ] header.tsx - Navigation + auth
- [ ] footer.tsx
- [ ] sidebar.tsx - Dashboard nav

# Listings
- [ ] listing-card.tsx - Thumbnail + details
- [ ] listing-grid.tsx - Grid layout
- [ ] listing-filters.tsx - Search filters
- [ ] listing-map.tsx - Leaflet map
- [ ] image-gallery.tsx - Lightbox

# Booking
- [ ] calendar.tsx - Availability picker
- [ ] payment-form.tsx - Wave/OM/Free
- [ ] booking-summary.tsx

# Shared
- [ ] image-upload.tsx - Drag & drop
- [ ] search-box.tsx - Geo autocomplete
- [ ] loading-spinner.tsx
- [ ] error-boundary.tsx
```

### Tests

```bash
# Backend
services/api/tests/
- [ ] unit/ - Handler tests
- [ ] integration/ - DB + API tests
- [ ] load/ - k6 scripts (3G simulation)

# Frontend
services/web/
- [ ] __tests__/unit/ - Component tests
- [ ] __tests__/integration/ - API tests
- [ ] e2e/ - Playwright tests (3G network)
```

---

## 📋 Plan d'Implémentation (6 Semaines)

### Semaine 1: Auth & Setup
- [ ] Setup local environment
- [ ] Auth handlers (register, login, logout)
- [ ] Auth service (JWT, password hashing)
- [ ] User repository
- [ ] Frontend: Login/Register pages
- [ ] Tests auth flow

### Semaine 2: Listings CRUD
- [ ] Listing handlers (CRUD)
- [ ] Listing service
- [ ] Listing repository avec PostGIS
- [ ] Image service (optimize, upload S3)
- [ ] Frontend: Listing pages
- [ ] Tests listings

### Semaine 3: Search & Filters
- [ ] Search handler avec geo-radius
- [ ] PostGIS queries < 200ms
- [ ] Redis cache listings
- [ ] Frontend: Search + filters
- [ ] Frontend: Map Leaflet
- [ ] Tests search performance

### Semaine 4: Booking
- [ ] Booking handlers
- [ ] Booking service
- [ ] Booking repository (exclusion constraint)
- [ ] Frontend: Calendar + booking form
- [ ] Tests double-booking prevention

### Semaine 5: Payments
- [ ] Payment handlers
- [ ] Wave service integration
- [ ] Webhook idempotent (déjà fait)
- [ ] Frontend: Payment flow
- [ ] Tests paiement end-to-end

### Semaine 6: Deploy & Polish
- [ ] Deploy Fly.io production
- [ ] Monitoring (Sentry, Better Uptime)
- [ ] Load tests k6
- [ ] Bug fixes
- [ ] Documentation updates

---

## 🚀 Commandes Génération Code

### Backend

```bash
# Générer handler boilerplate
cd services/api
cat > internal/handler/booking_handler.go << 'EOF'
package handler

type BookingHandler struct {
    bookingService *service.BookingService
}

func NewBookingHandler(svc *service.BookingService) *BookingHandler {
    return &BookingHandler{bookingService: svc}
}

// TODO: Implement handlers
EOF

# Générer service boilerplate
cat > internal/service/booking_service.go << 'EOF'
package service

type BookingService struct {
    repo *repository.BookingRepository
}

func NewBookingService(repo *repository.BookingRepository) *BookingService {
    return &BookingService{repo: repo}
}

// TODO: Implement business logic
EOF

# Générer repository boilerplate
cat > internal/repository/booking_repository.go << 'EOF'
package repository

import "database/sql"

type BookingRepository struct {
    db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
    return &BookingRepository{db: db}
}

// TODO: Implement data access
EOF
```

### Frontend

```bash
# Générer page boilerplate
cd services/web
mkdir -p app/\(dashboard\)/listings
cat > app/\(dashboard\)/listings/page.tsx << 'EOF'
import { ListingGrid } from '@/components/listings/listing-grid';

export default async function ListingsPage() {
  // TODO: Fetch listings from API
  return (
    <div className="container mx-auto py-8">
      <h1 className="text-3xl font-bold mb-8">Listings</h1>
      <ListingGrid listings={[]} />
    </div>
  );
}
EOF

# Générer component boilerplate
cat > components/listings/listing-card.tsx << 'EOF'
import Image from 'next/image';
import { Card } from '@/components/ui/card';

interface Props {
  listing: any; // TODO: Type from API
}

export function ListingCard({ listing }: Props) {
  return (
    <Card>
      {/* TODO: Implement card UI */}
    </Card>
  );
}
EOF
```

---

## 📚 Références pour Implémentation

### Backend
- **Auth:** `docs/guide-developpement-afrique.md` pas d'exemple mais standard JWT RS256
- **Paiements:** `services/api/internal/service/payment/wave_service.go` (complet)
- **Webhooks:** `services/api/internal/handler/webhook_handler.go` (complet)
- **PostGIS:** Guide développement section 5
- **Double-booking:** Guide développement section 6

### Frontend
- **PWA:** Guide développement section 2
- **Images:** Guide développement section 4
- **3G optimization:** Guide développement section 1
- **Tests E2E:** Guide développement section 7

### Database
- **Schema:** `services/api/migrations/` (complet)
- **Indexes:** Déjà optimisés dans migrations
- **Queries:** Examples dans guide développement

---

**⚠️ Priorité: Semaines 1-6 pour MVP production-ready**

Tout le reste (ML pricing, modération IA, etc.) = Phase 2+

**✅ Architecture & Documentation: 100% complètes**
**⏳ Implémentation code: À faire (6 semaines)**

---

**Version:** 1.0.0
**Dernière mise à jour:** 7 novembre 2025
