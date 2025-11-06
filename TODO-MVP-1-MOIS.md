# 📋 TODO MVP - Plan de Codage 1 Mois (4 Semaines)

**Objectif:** MVP Production-Ready en 4 semaines (au lieu de 6)
**Stratégie:** Focus features essentielles, pas de nice-to-have

---

## 🎯 Scope MVP Minimal (1 Mois)

### ✅ Features INCLUSES
- Auth (register, login, logout, JWT)
- Listings CRUD (create, read, update, delete)
- Search basique (ville, prix, surface)
- Booking simple (dates, paiement Wave uniquement)
- Admin basique (modération manuelle)
- PWA offline listings

### ❌ Features EXCLUES (Phase 2)
- Orange Money, Free Money (seulement Wave)
- KYC automatisé (validation manuelle)
- Modération IA photos (validation manuelle)
- Prix dynamique (prix fixe)
- QR code check-in (SMS code simple)
- Compare listings
- Notifications push
- Reviews & ratings

---

## 📅 Planning Détaillé

### Semaine 1: Setup & Auth (5 jours)

#### Jour 1: Setup Environnement (6h)
- [ ] **Setup local** (2h)
  ```bash
  git clone <repo>
  make setup
  make docker-up
  make migrate-up
  ```
  - [ ] Vérifier tous services démarrent
  - [ ] Tester connexion DB
  - [ ] Tester connexion Redis

- [ ] **Config environnement** (2h)
  - [ ] Générer JWT keys (RS256)
  - [ ] Configurer .env API
  - [ ] Configurer .env Web
  - [ ] Tester hot-reload API
  - [ ] Tester hot-reload Web

- [ ] **Setup IDE** (2h)
  - [ ] Extensions Go (gopls, golangci-lint)
  - [ ] Extensions TypeScript (ESLint, Prettier)
  - [ ] Debug configuration
  - [ ] Git hooks (pre-commit)

#### Jour 2-3: Backend Auth (12h)
- [ ] **User Repository** (3h)
  ```go
  // services/api/internal/repository/user_repository.go
  - [ ] Create(user *User) error
  - [ ] GetByID(id uuid.UUID) (*User, error)
  - [ ] GetByEmail(email string) (*User, error)
  - [ ] Update(user *User) error
  - [ ] UpdateVerificationStatus(id uuid.UUID) error
  ```

- [ ] **Auth Service** (4h)
  ```go
  // services/api/internal/service/auth_service.go
  - [ ] Register(req *RegisterRequest) (*User, error)
    - Validate email unique
    - Hash password (Argon2)
    - Create user
    - Generate verification token
  - [ ] Login(req *LoginRequest) (*LoginResponse, error)
    - Check password
    - Generate JWT access token (15min)
    - Generate refresh token (7 days)
    - Store refresh token in Redis
  - [ ] RefreshToken(token string) (string, error)
    - Validate refresh token
    - Check Redis
    - Generate new access token
  - [ ] Logout(userID uuid.UUID) error
    - Blacklist token in Redis
  ```

- [ ] **Auth Handlers** (3h)
  ```go
  // services/api/internal/handler/auth_handler.go
  - [ ] Register() POST /api/v1/auth/register
  - [ ] Login() POST /api/v1/auth/login
  - [ ] Logout() POST /api/v1/auth/logout
  - [ ] RefreshToken() POST /api/v1/auth/refresh
  ```

- [ ] **Tests Auth** (2h)
  - [ ] Test Register success
  - [ ] Test Register duplicate email
  - [ ] Test Login success
  - [ ] Test Login wrong password
  - [ ] Test JWT validation

#### Jour 4-5: Frontend Auth (12h)
- [ ] **Auth Store** (2h)
  ```typescript
  // services/web/lib/stores/auth-store.ts
  - [ ] useAuthStore (Zustand)
    - user state
    - isAuthenticated
    - login()
    - logout()
    - register()
  ```

- [ ] **Auth API Client** (2h)
  ```typescript
  // services/web/lib/api-client.ts (déjà créé, améliorer)
  - [ ] Tester register()
  - [ ] Tester login()
  - [ ] Tester token refresh automatique
  - [ ] Tester logout()
  ```

- [ ] **UI Auth Pages** (6h)
  ```typescript
  // services/web/app/(auth)/
  - [ ] login/page.tsx
    - Email input
    - Password input
    - Submit button
    - Link vers register
    - Error handling
  - [ ] register/page.tsx
    - Form avec react-hook-form
    - Email, password, firstName, lastName, phone
    - Role selector (tenant/landlord)
    - Validation
    - Success redirect
  ```

- [ ] **Auth Components** (2h)
  ```typescript
  // services/web/components/auth/
  - [ ] login-form.tsx
  - [ ] register-form.tsx
  - [ ] auth-guard.tsx (HOC protected routes)
  ```

---

### Semaine 2: Listings CRUD (5 jours)

#### Jour 6-7: Backend Listings (12h)
- [ ] **Listing Repository** (4h)
  ```go
  // services/api/internal/repository/listing_repository.go
  - [ ] Create(listing *Listing) error
  - [ ] GetByID(id uuid.UUID) (*Listing, error)
  - [ ] GetByLandlordID(id uuid.UUID) ([]Listing, error)
  - [ ] Search(params SearchParams) ([]Listing, int, error)
    - Filter by city
    - Filter by price range
    - Filter by surface
    - Filter by bedrooms
    - Pagination (limit/offset)
    - ORDER BY created_at DESC
  - [ ] Update(listing *Listing) error
  - [ ] SoftDelete(id uuid.UUID) error
  ```

- [ ] **Image Service** (4h)
  ```go
  // services/api/internal/service/image_service.go
  - [ ] UploadImage(file multipart.File) (string, error)
    - Resize max 1200x800
    - Convert to WebP
    - Quality 80
    - Target < 100KB
    - Upload to S3/R2
    - Return URL
  - [ ] DeleteImage(key string) error
  ```

- [ ] **Listing Service** (4h)
  ```go
  // services/api/internal/service/listing_service.go
  - [ ] CreateListing(landlordID uuid.UUID, req *CreateListingRequest) (*Listing, error)
    - Validate landlord is authenticated
    - Validate all fields
    - Create listing (status=draft)
  - [ ] UpdateListing(id uuid.UUID, req *UpdateListingRequest) error
    - Check ownership
    - Update fields
  - [ ] DeleteListing(id uuid.UUID) error
    - Check ownership
    - Soft delete
  - [ ] GetListing(id uuid.UUID) (*Listing, error)
    - Increment view count (async)
  - [ ] SearchListings(params *SearchParams) (*SearchResponse, error)
    - Cache in Redis (60s TTL)
  ```

#### Jour 8-9: Backend Listings Handlers (12h)
- [ ] **Listing Handlers** (6h)
  ```go
  // services/api/internal/handler/listing_handler.go
  - [ ] CreateListing() POST /api/v1/listings
    - Require auth middleware
    - Require landlord role
    - Parse request
    - Call service
    - Return 201
  - [ ] GetListings() GET /api/v1/listings
    - Parse query params
    - Call service with cache
    - Return paginated response
  - [ ] GetListingByID() GET /api/v1/listings/:id
    - Parse ID
    - Call service
    - Return listing
  - [ ] UpdateListing() PUT /api/v1/listings/:id
    - Require auth
    - Check ownership
    - Call service
  - [ ] DeleteListing() DELETE /api/v1/listings/:id
    - Require auth
    - Check ownership
    - Soft delete
  ```

- [ ] **Upload Handler** (3h)
  ```go
  // services/api/internal/handler/upload_handler.go
  - [ ] UploadPhoto() POST /api/v1/listings/:id/photos
    - Require auth
    - Check ownership
    - Parse multipart file
    - Max 5MB
    - Call image service
    - Store in listing_photos table
  ```

- [ ] **Tests Listings** (3h)
  - [ ] Test Create listing
  - [ ] Test Search listings
  - [ ] Test Update ownership check
  - [ ] Test Image upload
  - [ ] Test Pagination

#### Jour 10: Frontend Listings - Liste (6h)
- [ ] **Listing Components** (4h)
  ```typescript
  // services/web/components/listings/
  - [ ] listing-card.tsx
    - Image thumbnail
    - Title, price, city
    - Bedrooms, bathrooms, surface
    - Link to detail
  - [ ] listing-grid.tsx
    - Grid layout (3 cols desktop, 1 col mobile)
    - Skeleton loading
  - [ ] listing-filters.tsx
    - City select
    - Price range slider
    - Surface range
    - Bedrooms select
  ```

- [ ] **Listings Page** (2h)
  ```typescript
  // services/web/app/(dashboard)/listings/page.tsx
  - [ ] Fetch listings with TanStack Query
  - [ ] Display grid
  - [ ] Filters sidebar
  - [ ] Pagination
  - [ ] Loading states
  ```

---

### Semaine 3: Listings Detail & Booking (5 jours)

#### Jour 11: Frontend Listing Detail (6h)
- [ ] **Listing Detail Page** (6h)
  ```typescript
  // services/web/app/(dashboard)/listings/[id]/page.tsx
  - [ ] Image gallery (carousel)
  - [ ] Title, description
  - [ ] Price per month, deposit
  - [ ] Details (bedrooms, bathrooms, surface, floor)
  - [ ] Amenities list
  - [ ] Map with Leaflet (lat/lng)
  - [ ] Landlord info
  - [ ] Booking form (sidebar)
  ```

#### Jour 12: Frontend Create Listing (6h)
- [ ] **Create Listing Page** (6h)
  ```typescript
  // services/web/app/(dashboard)/listings/new/page.tsx
  - [ ] Multi-step form (react-hook-form)
    - Step 1: Basic info (title, description, type)
    - Step 2: Address & location (city, address, map picker)
    - Step 3: Details (bedrooms, bathrooms, surface, floor)
    - Step 4: Price (monthly, deposit)
    - Step 5: Amenities (checkboxes)
    - Step 6: Photos (drag & drop upload)
  - [ ] Image upload with preview
  - [ ] Validation
  - [ ] Submit
  ```

#### Jour 13-14: Backend Booking (12h)
- [ ] **Booking Repository** (4h)
  ```go
  // services/api/internal/repository/booking_repository.go
  - [ ] Create(booking *Booking) error
    - Transaction SERIALIZABLE
    - Insert with daterange
    - Exclusion constraint catch
  - [ ] GetByID(id uuid.UUID) (*Booking, error)
  - [ ] GetByUserID(userID uuid.UUID) ([]Booking, error)
  - [ ] GetByListingID(listingID uuid.UUID) ([]Booking, error)
  - [ ] CheckAvailability(listingID uuid.UUID, start, end time.Time) (bool, error)
  - [ ] UpdateStatus(id uuid.UUID, status string) error
  ```

- [ ] **Booking Service** (4h)
  ```go
  // services/api/internal/service/booking_service.go
  - [ ] CreateBooking(tenantID uuid.UUID, req *CreateBookingRequest) (*Booking, error)
    - Check availability
    - Calculate total (price * months)
    - Create booking (status=pending)
    - Return booking with payment URL
  - [ ] GetBooking(id uuid.UUID, userID uuid.UUID) (*Booking, error)
    - Check ownership (tenant or landlord)
  - [ ] GetUserBookings(userID uuid.UUID) ([]Booking, error)
  - [ ] CancelBooking(id uuid.UUID, userID uuid.UUID) error
    - Check ownership
    - Check status (only pending/confirmed)
    - Update status=cancelled
  ```

- [ ] **Booking Handlers** (4h)
  ```go
  // services/api/internal/handler/booking_handler.go
  - [ ] CreateBooking() POST /api/v1/bookings
  - [ ] GetBooking() GET /api/v1/bookings/:id
  - [ ] GetUserBookings() GET /api/v1/bookings
  - [ ] CancelBooking() POST /api/v1/bookings/:id/cancel
  - [ ] GetAvailability() GET /api/v1/listings/:id/availability
  ```

#### Jour 15: Frontend Booking (6h)
- [ ] **Booking Components** (4h)
  ```typescript
  // services/web/components/booking/
  - [ ] booking-calendar.tsx
    - Date range picker
    - Disable unavailable dates
    - Calculate total
  - [ ] booking-form.tsx
    - Check-in, check-out
    - Total summary
    - Submit button
  ```

- [ ] **Bookings Page** (2h)
  ```typescript
  // services/web/app/(dashboard)/bookings/page.tsx
  - [ ] List user bookings
  - [ ] Status badges
  - [ ] Cancel button
  ```

---

### Semaine 4: Paiement Wave & Deploy (5 jours)

#### Jour 16-17: Backend Payment Wave (12h)
- [ ] **Payment Service** (6h)
  ```go
  // services/api/internal/service/payment_service.go
  - [ ] CreatePayment(bookingID uuid.UUID, req *CreatePaymentRequest) (*PaymentResponse, error)
    - Get booking
    - Call Wave service (déjà créé)
    - Store payment (status=pending)
    - Return payment URL
  - [ ] ProcessWebhook(provider string, payload []byte, signature string) error
    - Verify signature (déjà fait dans webhook_handler.go)
    - Check idempotency
    - Update payment status
    - Update booking status
    - Publish NATS event (ou skip pour MVP)
  ```

- [ ] **Payment Handlers** (3h)
  ```go
  // services/api/internal/handler/payment_handler.go
  - [ ] CreatePayment() POST /api/v1/payments
  - [ ] GetPayment() GET /api/v1/payments/:id
  - [ ] WaveWebhook() POST /api/v1/webhooks/wave (déjà créé, tester)
  ```

- [ ] **Tests Payment** (3h)
  - [ ] Test create payment
  - [ ] Test webhook idempotency
  - [ ] Test booking status update
  - [ ] Test webhook signature verification

#### Jour 18: Frontend Payment (6h)
- [ ] **Payment Components** (4h)
  ```typescript
  // services/web/components/payment/
  - [ ] payment-modal.tsx
    - Show Wave payment URL
    - QR code display
    - Instructions
    - Polling status (every 3s)
  ```

- [ ] **Payment Flow** (2h)
  - [ ] In booking form: Submit → Create payment → Redirect/QR
  - [ ] Handle return from Wave
  - [ ] Poll booking status
  - [ ] Show success page

#### Jour 19: Admin Dashboard (6h)
- [ ] **Admin Pages** (6h)
  ```typescript
  // services/web/app/(dashboard)/admin/
  - [ ] dashboard/page.tsx
    - Stats cards (total listings, bookings, revenue)
    - Recent bookings table
  - [ ] listings/page.tsx
    - List all listings
    - Approve/Reject buttons
    - Change status (draft/published)
  - [ ] users/page.tsx
    - List users
    - View details
    - Ban/Unban
  ```

#### Jour 20: Deploy Production (6h)
- [ ] **Setup Fly.io** (2h)
  ```bash
  - [ ] Create account Fly.io
  - [ ] Install flyctl
  - [ ] Create app API
  - [ ] Create Postgres
  - [ ] Configure secrets
  ```

- [ ] **Deploy API** (2h)
  ```bash
  - [ ] Test build local
  - [ ] Deploy Fly.io
  - [ ] Run migrations
  - [ ] Test health endpoint
  ```

- [ ] **Deploy Web** (1h)
  ```bash
  - [ ] Build Next.js
  - [ ] Deploy Fly.io
  - [ ] Test app loads
  ```

- [ ] **Configure Domain & SSL** (1h)
  ```bash
  - [ ] Point DNS to Fly.io
  - [ ] Setup SSL certificates
  - [ ] Test HTTPS
  ```

---

## 📦 Checklist Pré-Deploy

### Backend
- [ ] Toutes les migrations passent
- [ ] Tous les tests passent (make test)
- [ ] Lint sans erreurs (make lint)
- [ ] JWT keys en production (secrets)
- [ ] Wave API keys configurées
- [ ] Database connection string production
- [ ] Redis connection string production
- [ ] CORS origins production

### Frontend
- [ ] Build Next.js réussit
- [ ] Lighthouse score > 80
- [ ] Bundle size < 150KB
- [ ] Images < 100KB
- [ ] PWA manifest configuré
- [ ] Service Worker fonctionne
- [ ] API_URL pointe vers prod

### Infrastructure
- [ ] Fly.io apps créées (api + web)
- [ ] Postgres provisionné
- [ ] Redis Upstash configuré
- [ ] S3/R2 bucket créé
- [ ] Domain DNS configuré
- [ ] SSL certificates installés

### Monitoring
- [ ] Sentry DSN configuré
- [ ] Better Uptime monitor créé
- [ ] Health check fonctionne
- [ ] Logs accessibles (fly logs)

---

## 🚀 Commandes Rapides

### Développement
```bash
# Jour 1: Setup
make setup && make docker-up && make migrate-up

# Chaque jour: Start
make dev  # API + Web hot reload

# Tests
make test-api  # Backend tests
make test-web  # Frontend tests

# Lint
make lint  # All linters
```

### Semaine 4: Deploy
```bash
# API
cd services/api
fly deploy --app immobilier-api-mvp

# Web
cd services/web
fly deploy --app immobilier-web-mvp

# Migrations
fly ssh console --app immobilier-api-mvp
migrate -path /app/migrations -database $DATABASE_URL up
```

---

## 📊 Métriques de Succès MVP

### Fonctionnel
- [ ] User peut s'inscrire et se connecter
- [ ] Landlord peut créer un listing avec photos
- [ ] Tenant peut chercher et voir listings
- [ ] Tenant peut réserver et payer (Wave)
- [ ] Admin peut modérer listings

### Performance
- [ ] API p99 < 300ms (acceptable MVP)
- [ ] Page load < 5s sur 3G
- [ ] Images < 100KB

### Technique
- [ ] Tests coverage > 60% (MVP)
- [ ] Zero critical security issues
- [ ] Deploy automatisé (GitHub Actions)
- [ ] Monitoring actif

---

## 🎯 Priorités si Retard

### Must Have (Blocker)
1. Auth (register, login)
2. Listings CRUD
3. Booking création
4. Payment Wave (sans webhook = polling)
5. Deploy production

### Nice to Have (Skip si retard)
- Webhook idempotent (manuel first)
- Admin dashboard (CLI first)
- PWA offline (Phase 2)
- Image optimization < 100KB (accepter 200KB)
- Tests > 60% (accepter 40%)

---

## 📞 Support Quotidien

### Blocages Techniques
- **Database:** Consulter migrations SQL existantes
- **Auth JWT:** Voir `internal/middleware/auth.go`
- **Wave:** Voir `internal/service/payment/wave_service.go`
- **Frontend:** Voir `lib/api-client.ts`

### Guides
- **Architecture:** `docs/architecture-africaine.md`
- **Développement:** `docs/guide-developpement-afrique.md`
- **Deployment:** `docs/deployment-mvp-budget.md`

---

## ✅ Daily Checklist

Chaque jour:
- [ ] Standup 9h: Plan du jour
- [ ] Code 9h30-12h30 (3h focus)
- [ ] Pause déjeuner
- [ ] Code 14h-18h (4h focus)
- [ ] Tests + commit 18h-18h30
- [ ] Review progress 18h30

Chaque semaine:
- [ ] Vendredi 17h: Demo interne
- [ ] Vendredi 18h: Rétrospective
- [ ] Dimanche: Plan semaine suivante

---

**🎯 Objectif: MVP Production Samedi 7 Décembre 2025**

**💪 Tu peux le faire ! Focus + Discipline = Success**

---

**Version:** 1.0
**Créé:** 7 novembre 2025
**Deadline:** 7 décembre 2025 (4 semaines)
