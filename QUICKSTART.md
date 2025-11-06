# 🚀 Quick Start Guide - Immobilier SaaS

Get up and running with the Immobilier SaaS platform in under 10 minutes!

## Prerequisites

Make sure you have installed:
- **Docker Desktop** (https://www.docker.com/products/docker-desktop)
- **Make** (usually pre-installed on Mac/Linux, Windows users can use WSL)

That's it! Docker will handle all other dependencies.

## Step 1: Clone the Repository

```bash
git clone https://github.com/yourusername/Immobilier_Saas.git
cd Immobilier_Saas
```

## Step 2: Initial Setup

Run the automated setup command:

```bash
make setup
```

This will:
- Create environment files from templates
- Generate JWT RSA keys for authentication
- Display next steps

## Step 3: Configure Environment

Edit `services/api/.env` and add the generated JWT keys:

```bash
# Copy the contents of private.pem
JWT_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
... paste private key content here ...
-----END PRIVATE KEY-----"

# Copy the contents of public.pem
JWT_PUBLIC_KEY="-----BEGIN PUBLIC KEY-----
... paste public key content here ...
-----END PUBLIC KEY-----"
```

**Tip:** The keys were generated in Step 2 and saved to `services/api/private.pem` and `services/api/public.pem`

## Step 4: Start All Services

```bash
make docker-up
```

Wait for all services to start (about 30-60 seconds). You'll see a success message with all service URLs.

## Step 5: Initialize Database

```bash
make migrate-up
```

This creates all database tables with PostGIS support.

## Step 6: Access the Platform

Open your browser and visit:

| Service | URL | Credentials |
|---------|-----|-------------|
| 🌐 Web App | http://localhost:3000 | Register new account |
| 🔧 API | http://localhost:8080 | - |
| 📚 Swagger Docs | http://localhost:8080/swagger/index.html | - |
| 📧 MailHog (Email) | http://localhost:8025 | - |
| 📊 Grafana | http://localhost:3001 | admin / admin |
| 💾 MinIO Console | http://localhost:9001 | minioadmin / minioadmin |

## Step 7: Create Your First User

### Option A: Via Web UI
1. Go to http://localhost:3000
2. Click "Register"
3. Fill in the form:
   - Email: your@email.com
   - Password: (min 8 characters)
   - First Name: Your Name
   - Last Name: Your Last Name
   - Role: Select "Landlord" or "Tenant"
4. Click "Register"
5. Check MailHog at http://localhost:8025 for verification email

### Option B: Via API (using curl)

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "landlord@example.com",
    "password": "SecurePass123",
    "first_name": "John",
    "last_name": "Doe",
    "role": "landlord"
  }'
```

## Step 8: Create Your First Listing (Landlord)

After logging in as a landlord:

```bash
# Login first
LOGIN_RESPONSE=$(curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "landlord@example.com",
    "password": "SecurePass123"
  }')

# Extract token
TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.access_token')

# Create listing
curl -X POST http://localhost:8080/api/v1/listings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "title": "Beautiful 2BR Apartment in Dakar",
    "description": "Spacious apartment with ocean view in the heart of Dakar. Modern amenities, secure building, close to amenities.",
    "property_type": "apartment",
    "address": "Rue de la République",
    "city": "Dakar",
    "postal_code": "12500",
    "latitude": 14.6937,
    "longitude": -17.4441,
    "price_per_month": 350000,
    "deposit_amount": 700000,
    "surface_area": 85,
    "bedrooms": 2,
    "bathrooms": 1,
    "floor": 3,
    "total_floors": 5,
    "amenities": ["wifi", "parking", "air_conditioning", "balcony"],
    "available_from": "2024-02-01",
    "min_lease_months": 6,
    "max_lease_months": 24
  }'
```

## Common Commands

```bash
# View logs
make logs

# Stop all services
make docker-down

# Restart services
make docker-restart

# Run tests
make test

# Access PostgreSQL
make db-shell

# Access Redis CLI
make redis-cli

# Check service health
make health

# View all available commands
make help
```

## Development Workflow

### Backend Development (Go)
```bash
# Work on API in hot-reload mode
cd services/api
make dev

# Run tests
make test

# Generate Swagger docs
make swagger
```

### Frontend Development (Next.js)
```bash
# Work on web in hot-reload mode
cd services/web
npm run dev

# Run tests
npm test

# Type checking
npm run type-check
```

## Troubleshooting

### Services won't start

```bash
# Check if ports are already in use
lsof -i :8080  # API
lsof -i :3000  # Web
lsof -i :5432  # PostgreSQL

# Clean everything and start fresh
make clean-all
make docker-up
make migrate-up
```

### Database migrations fail

```bash
# Force migration to specific version
make migrate-force VERSION=1

# Then run migrations again
make migrate-up
```

### JWT authentication not working

Make sure you:
1. Generated the keys with `make setup`
2. Added both private and public keys to `services/api/.env`
3. Keys are properly formatted (include BEGIN/END markers)
4. No extra quotes or whitespace

### Can't connect to services

```bash
# Check if all containers are running
make ps

# Check logs for specific service
make logs-api
make logs-web

# Restart specific service
docker-compose restart api
docker-compose restart web
```

## Next Steps

1. **Read the Documentation**
   - [Full README](README.md)
   - [Architecture Overview](STRUCTURE.md)
   - [Security Checklist](docs/security-checklist.md)
   - [Deployment Guide](docs/deployment.md)

2. **Explore the API**
   - Visit Swagger UI: http://localhost:8080/swagger/index.html
   - Try different endpoints
   - Test authentication flow

3. **Customize the Platform**
   - Modify database schema in `services/api/migrations/`
   - Update API handlers in `services/api/internal/handler/`
   - Customize UI components in `services/web/components/`

4. **Deploy to Production**
   - Follow the [30-minute deployment guide](docs/deployment.md)
   - Deploy to Fly.io, AWS, or your preferred cloud

## Getting Help

- **Documentation:** Check [README.md](README.md) and [docs/](docs/)
- **Issues:** Open an issue on GitHub
- **Email:** support@immobilier.com

## Sample Data

Want to populate the database with sample listings for testing?

```bash
# Run seed script (coming soon)
make seed
```

This will create:
- 5 landlord accounts
- 10 tenant accounts
- 25 property listings
- Sample bookings and payments

---

**🎉 Congratulations!** You now have a fully functional real-estate SaaS platform running locally!

Explore the features, build amazing experiences, and deploy to production when you're ready.

Happy coding! 🚀
