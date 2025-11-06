# 🏠 Immobilier SaaS - Real Estate Platform

Production-grade, scalable real estate SaaS platform for Senegal built with Go, Next.js, PostgreSQL, and deployed on cloud infrastructure.

[![API CI](https://github.com/username/immobilier-saas/actions/workflows/api-ci.yml/badge.svg)](https://github.com/username/immobilier-saas/actions/workflows/api-ci.yml)
[![Web CI](https://github.com/username/immobilier-saas/actions/workflows/web-ci.yml/badge.svg)](https://github.com/username/immobilier-saas/actions/workflows/web-ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## 📋 Table of Contents

- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Architecture](#-architecture)
- [Getting Started](#-getting-started)
- [Development](#-development)
- [Deployment](#-deployment)
- [Documentation](#-documentation)
- [Contributing](#-contributing)
- [License](#-license)

## ✨ Features

### Core Features
- 🔐 **Authentication & Authorization** - JWT RS256 with refresh tokens
- 🏘️ **Property Listings** - CRUD operations with photos, amenities, and geo-location
- 🔍 **Advanced Search** - Filter by price, surface, location, amenities with geo-radius search
- 📅 **Booking System** - Calendar availability with 2-step payment (deposit + balance)
- 💳 **Payment Integration** - Wave, Orange Money, Free Money with webhook handling
- ⭐ **Favorites & Compare** - Save listings and compare side-by-side
- 🔔 **Notifications** - Email & SMS for bookings, payments, and reminders
- 📊 **Admin Dashboard** - Analytics, user management, and listing moderation
- 🌍 **SEO Optimized** - SSR/ISR pages, dynamic sitemap, OpenGraph images
- 📱 **PWA Support** - Offline cache, install prompt, push notifications
- 📝 **E-Signature** - DocuSign integration for rental contracts
- 🔒 **GDPR Compliant** - Data export, deletion, and privacy controls

### Technical Features
- Horizontal scaling with stateless services
- < 200ms p99 API response time (cached)
- 99.9% monthly SLA
- Zero-downtime deployments (blue/green)
- Automated daily backups with 7-day PITR
- Infrastructure as Code (Terraform)
- Comprehensive observability (Prometheus, Grafana, Loki, Sentry)

## 🛠 Tech Stack

### Backend
- **Language:** Go 1.22
- **Framework:** Fiber (high-performance HTTP)
- **Database:** PostgreSQL 15 + PostGIS
- **Cache:** Redis 7
- **Queue:** NATS JetStream
- **Storage:** S3-compatible (AWS S3 / MinIO)
- **Auth:** JWT RS256 + HTTP-only refresh cookies

### Frontend
- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript
- **Styling:** Tailwind CSS + shadcn/ui
- **State:** Zustand + TanStack Query
- **Maps:** Leaflet (OpenStreetMap)

### Infrastructure
- **Cloud:** AWS / Fly.io
- **Container:** Docker
- **Orchestration:** ECS / Kubernetes
- **CDN:** CloudFront / Cloudflare
- **IaC:** Terraform
- **CI/CD:** GitHub Actions

### Observability
- **Metrics:** Prometheus + Grafana
- **Logs:** Loki
- **Errors:** Sentry
- **Tracing:** OpenTelemetry (planned)

## 🏗 Architecture

```
┌─────────────────┐
│   Next.js PWA   │ ← Users
└────────┬────────┘
         │ HTTPS
         ↓
┌─────────────────┐
│  Load Balancer  │
│   + WAF + CDN   │
└────────┬────────┘
         │
    ┌────┴────┐
    ↓         ↓
┌───────┐ ┌───────┐
│  API  │ │  API  │ (Auto-scaled)
│ Go+Fiber │
└───┬───┘ └───┬───┘
    │         │
    └────┬────┘
         │
    ┌────┴────┐
    ↓         ↓
┌──────┐  ┌──────┐
│ Redis│  │ NATS │
└──────┘  └──────┘
    │         │
    └────┬────┘
         ↓
    ┌─────────┐
    │PostgreSQL│
    │ + PostGIS│
    └─────────┘
```

See [STRUCTURE.md](STRUCTURE.md) for detailed mono-repo structure.

## 🚀 Getting Started

### Prerequisites

- Docker & Docker Compose
- Go 1.22+ (for local API development)
- Node.js 20+ (for local web development)
- Make (optional, for convenience commands)

### Quick Start (Docker)

```bash
# Clone the repository
git clone https://github.com/username/immobilier-saas.git
cd immobilier-saas

# Copy environment files
cp services/api/.env.example services/api/.env
cp services/web/.env.example services/web/.env

# Generate JWT keys
openssl genrsa -out services/api/private.pem 2048
openssl rsa -in services/api/private.pem -pubout -out services/api/public.pem

# Add keys to services/api/.env (see .env.example for format)

# Start all services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f
```

Services will be available at:
- **API:** http://localhost:8080
- **Web:** http://localhost:3000
- **Swagger Docs:** http://localhost:8080/swagger/index.html
- **PostgreSQL:** localhost:5432
- **Redis:** localhost:6379
- **MinIO Console:** http://localhost:9001
- **MailHog:** http://localhost:8025
- **Grafana:** http://localhost:3001

### Database Setup

```bash
# Run migrations
docker-compose exec api migrate -path /app/migrations -database "$DATABASE_URL" up

# Seed database (optional)
docker-compose exec api go run scripts/seed.go
```

## 💻 Development

### API Development

```bash
cd services/api

# Install dependencies
go mod download

# Install tools
make install

# Run with hot reload
make dev

# Run tests
make test

# Run linter
make lint

# Generate Swagger docs
make swagger
```

### Web Development

```bash
cd services/web

# Install dependencies
npm install

# Run development server
npm run dev

# Run tests
npm test

# Run type check
npm run type-check

# Build for production
npm run build
```

### Running Migrations

```bash
# Create new migration
cd services/api
make migrate-create name=add_reviews_table

# Run migrations up
make migrate-up

# Rollback migration
make migrate-down

# Force migration version
make migrate-force version=1
```

## 📦 Deployment

### Production Deployment (30 minutes)

Follow the comprehensive [Deployment Runbook](docs/deployment.md) for step-by-step instructions.

**Quick summary:**

1. **Generate JWT keys** (2 min)
2. **Configure AWS infrastructure** with Terraform (10 min)
3. **Deploy API** to Fly.io (8 min)
4. **Deploy Web** to Fly.io (5 min)
5. **Configure DNS** (3 min)
6. **Configure SSL** (2 min)

```bash
# Deploy to production
cd services/api
flyctl deploy --app immobilier-api-prod

cd ../web
flyctl deploy --app immobilier-web-prod
```

### Scaling

```bash
# Horizontal scaling
flyctl scale count 5 --app immobilier-api-prod

# Vertical scaling
flyctl scale vm shared-cpu-4x --memory 4096 --app immobilier-api-prod
```

## 📚 Documentation

- [Architecture Overview](STRUCTURE.md)
- [Deployment Runbook](docs/deployment.md)
- [Security Checklist](docs/security-checklist.md)
- [API Documentation](http://localhost:8080/swagger/index.html)
- [Database Schema](services/api/migrations/000001_init_schema.up.sql)

## 🔒 Security

This project follows security best practices:

- ✅ OWASP Top 10 protection
- ✅ GDPR compliant
- ✅ JWT RS256 authentication
- ✅ Rate limiting
- ✅ Input validation
- ✅ SQL injection prevention
- ✅ XSS protection
- ✅ CSRF protection
- ✅ Encryption at rest and in transit

See [Security Checklist](docs/security-checklist.md) for complete details.

## 🧪 Testing

```bash
# Backend tests
cd services/api
make test
make test-coverage

# Frontend tests
cd services/web
npm test
npm run test:e2e
```

## 📊 Monitoring

### Prometheus Metrics

- API request duration
- Request count by endpoint
- Error rates
- Database query performance
- Redis cache hit ratio

### Grafana Dashboards

- System metrics (CPU, Memory, Disk)
- Application metrics (requests, errors)
- Database metrics (connections, queries)
- Business metrics (bookings, revenue)

### Sentry Error Tracking

All errors are automatically tracked in Sentry with context:
- User information
- Request details
- Stack traces
- Breadcrumbs

## 🤝 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- **Go:** Follow [Effective Go](https://golang.org/doc/effective_go) and use `gofmt`
- **TypeScript:** Follow [TypeScript Style Guide](https://google.github.io/styleguide/tsguide.html)
- **Commits:** Use [Conventional Commits](https://www.conventionalcommits.org/)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👥 Team

- **Product Owner:** [Name]
- **Tech Lead:** [Name]
- **Backend Developer:** [Name]
- **Frontend Developer:** [Name]
- **DevOps Engineer:** [Name]

## 📞 Support

- **Email:** support@immobilier.com
- **Slack:** #immobilier-support
- **Documentation:** https://docs.immobilier.com

## 🗺 Roadmap

### Phase 1 (Current)
- [x] Core authentication
- [x] Listing CRUD
- [x] Search & filters
- [x] Booking system
- [x] Payment integration

### Phase 2 (Q2 2024)
- [ ] Mobile app (React Native)
- [ ] Video tours
- [ ] Virtual reality tours
- [ ] AI-powered recommendations
- [ ] Advanced analytics

### Phase 3 (Q3 2024)
- [ ] Multi-language support
- [ ] Multi-currency support
- [ ] Agent dashboard
- [ ] Property management tools
- [ ] Tenant screening

## 🙏 Acknowledgments

- [Fiber](https://gofiber.io/) - Fast HTTP framework for Go
- [Next.js](https://nextjs.org/) - React framework
- [shadcn/ui](https://ui.shadcn.com/) - Beautiful UI components
- [PostGIS](https://postgis.net/) - Spatial database extension
- [Fly.io](https://fly.io/) - Application hosting

---

**Built with ❤️ for the Senegalese real estate market**