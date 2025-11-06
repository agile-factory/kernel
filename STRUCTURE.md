# Immobilier SaaS - Mono-repo Structure

```
immobilier-saas/
├── .github/
│   └── workflows/
│       ├── api-ci.yml
│       ├── web-ci.yml
│       ├── deploy-staging.yml
│       └── deploy-production.yml
├── services/
│   ├── api/                          # Go-Fiber Backend
│   │   ├── cmd/
│   │   │   ├── api/
│   │   │   │   └── main.go
│   │   │   └── worker/
│   │   │       └── main.go
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   │   └── config.go
│   │   │   ├── domain/               # Business entities
│   │   │   │   ├── user.go
│   │   │   │   ├── listing.go
│   │   │   │   ├── booking.go
│   │   │   │   ├── payment.go
│   │   │   │   └── notification.go
│   │   │   ├── repository/           # Data layer
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── listing_repository.go
│   │   │   │   ├── booking_repository.go
│   │   │   │   └── payment_repository.go
│   │   │   ├── service/              # Business logic
│   │   │   │   ├── auth_service.go
│   │   │   │   ├── listing_service.go
│   │   │   │   ├── booking_service.go
│   │   │   │   ├── payment_service.go
│   │   │   │   └── notification_service.go
│   │   │   ├── handler/              # HTTP handlers
│   │   │   │   ├── auth_handler.go
│   │   │   │   ├── listing_handler.go
│   │   │   │   ├── booking_handler.go
│   │   │   │   ├── payment_handler.go
│   │   │   │   └── admin_handler.go
│   │   │   ├── middleware/
│   │   │   │   ├── auth.go
│   │   │   │   ├── rate_limit.go
│   │   │   │   ├── cors.go
│   │   │   │   └── security.go
│   │   │   ├── worker/               # Background jobs
│   │   │   │   ├── email_worker.go
│   │   │   │   ├── sms_worker.go
│   │   │   │   └── payment_worker.go
│   │   │   └── pkg/                  # Internal packages
│   │   │       ├── jwt/
│   │   │       ├── validation/
│   │   │       ├── postgres/
│   │   │       ├── redis/
│   │   │       ├── s3/
│   │   │       └── nats/
│   │   ├── migrations/
│   │   │   ├── 000001_init_schema.up.sql
│   │   │   ├── 000001_init_schema.down.sql
│   │   │   ├── 000002_add_postgis.up.sql
│   │   │   └── 000002_add_postgis.down.sql
│   │   ├── docs/                     # Swagger generated
│   │   │   ├── docs.go
│   │   │   ├── swagger.json
│   │   │   └── swagger.yaml
│   │   ├── tests/
│   │   │   ├── integration/
│   │   │   └── unit/
│   │   ├── .env.example
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── go.sum
│   └── web/                          # Next.js Frontend
│       ├── app/
│       │   ├── (auth)/
│       │   │   ├── login/
│       │   │   │   └── page.tsx
│       │   │   ├── register/
│       │   │   │   └── page.tsx
│       │   │   └── layout.tsx
│       │   ├── (dashboard)/
│       │   │   ├── dashboard/
│       │   │   │   └── page.tsx
│       │   │   ├── listings/
│       │   │   │   ├── page.tsx
│       │   │   │   ├── [id]/
│       │   │   │   │   └── page.tsx
│       │   │   │   └── new/
│       │   │   │       └── page.tsx
│       │   │   ├── bookings/
│       │   │   │   └── page.tsx
│       │   │   └── layout.tsx
│       │   ├── (marketing)/
│       │   │   ├── page.tsx
│       │   │   ├── about/
│       │   │   │   └── page.tsx
│       │   │   ├── pricing/
│       │   │   │   └── page.tsx
│       │   │   └── layout.tsx
│       │   ├── (legal)/
│       │   │   ├── terms/
│       │   │   │   └── page.tsx
│       │   │   ├── privacy/
│       │   │   │   └── page.tsx
│       │   │   └── layout.tsx
│       │   ├── api/
│       │   │   └── webhooks/
│       │   │       └── payment/
│       │   │           └── route.ts
│       │   ├── favicon.ico
│       │   ├── layout.tsx
│       │   └── globals.css
│       ├── components/
│       │   ├── ui/                   # shadcn/ui components
│       │   │   ├── button.tsx
│       │   │   ├── card.tsx
│       │   │   ├── input.tsx
│       │   │   └── ...
│       │   ├── layout/
│       │   │   ├── header.tsx
│       │   │   ├── footer.tsx
│       │   │   └── sidebar.tsx
│       │   ├── listings/
│       │   │   ├── listing-card.tsx
│       │   │   ├── listing-grid.tsx
│       │   │   ├── listing-filters.tsx
│       │   │   └── listing-map.tsx
│       │   ├── booking/
│       │   │   ├── calendar.tsx
│       │   │   └── payment-form.tsx
│       │   └── shared/
│       │       ├── image-upload.tsx
│       │       └── search-box.tsx
│       ├── lib/
│       │   ├── api-client.ts
│       │   ├── auth.ts
│       │   ├── hooks/
│       │   │   ├── use-auth.ts
│       │   │   ├── use-listings.ts
│       │   │   └── use-bookings.ts
│       │   ├── utils.ts
│       │   └── validations.ts
│       ├── public/
│       │   ├── images/
│       │   ├── icons/
│       │   ├── manifest.json
│       │   └── sw.js
│       ├── types/
│       │   ├── api.ts
│       │   └── index.ts
│       ├── .env.example
│       ├── .eslintrc.json
│       ├── Dockerfile
│       ├── next.config.js
│       ├── package.json
│       ├── postcss.config.js
│       ├── tailwind.config.ts
│       └── tsconfig.json
├── shared/
│   ├── contracts/                    # Shared types & schemas
│   │   ├── user.ts
│   │   ├── listing.ts
│   │   ├── booking.ts
│   │   └── payment.ts
│   └── proto/                        # If using gRPC
│       └── api.proto
├── infrastructure/
│   ├── terraform/
│   │   ├── modules/
│   │   │   ├── vpc/
│   │   │   ├── rds/
│   │   │   ├── redis/
│   │   │   ├── s3/
│   │   │   ├── ecs/
│   │   │   └── alb/
│   │   ├── environments/
│   │   │   ├── staging/
│   │   │   │   ├── main.tf
│   │   │   │   ├── variables.tf
│   │   │   │   └── terraform.tfvars
│   │   │   └── production/
│   │   │       ├── main.tf
│   │   │       ├── variables.tf
│   │   │       └── terraform.tfvars
│   │   └── backend.tf
│   ├── kubernetes/
│   │   ├── api/
│   │   │   ├── deployment.yaml
│   │   │   ├── service.yaml
│   │   │   └── hpa.yaml
│   │   └── web/
│   │       ├── deployment.yaml
│   │       └── service.yaml
│   └── monitoring/
│       ├── prometheus/
│       │   └── config.yml
│       ├── grafana/
│       │   └── dashboards/
│       └── loki/
│           └── config.yml
├── scripts/
│   ├── migrate.sh
│   ├── seed.sh
│   ├── backup.sh
│   └── deploy.sh
├── docs/
│   ├── architecture.md
│   ├── api-guide.md
│   ├── deployment.md
│   └── security.md
├── docker-compose.yml
├── docker-compose.prod.yml
├── .gitignore
├── .env.example
├── Makefile
└── README.md
```
