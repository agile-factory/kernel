# Deployment Runbook - Immobilier SaaS

This guide will walk you through deploying the Immobilier SaaS platform to production in approximately 30 minutes.

## Prerequisites

Before starting, ensure you have:

- [ ] AWS account with appropriate permissions
- [ ] Terraform >= 1.5 installed
- [ ] AWS CLI configured (`aws configure`)
- [ ] Fly.io CLI installed (`flyctl`)
- [ ] Docker installed
- [ ] Go 1.22+ installed
- [ ] Node.js 20+ installed
- [ ] Domain name configured (e.g., immobilier.com)

## Step 1: Generate JWT Keys (2 minutes)

```bash
# Generate RS256 key pair for JWT
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# Display keys (you'll need these for environment variables)
cat private.pem
cat public.pem
```

**Save these keys securely!** Store them in AWS Secrets Manager or your secrets management tool.

## Step 2: Configure AWS Infrastructure (10 minutes)

### 2.1 Create S3 Backend for Terraform State

```bash
# Create S3 bucket for Terraform state
aws s3 mb s3://immobilier-saas-terraform-state --region us-east-1

# Enable versioning
aws s3api put-bucket-versioning \
  --bucket immobilier-saas-terraform-state \
  --versioning-configuration Status=Enabled

# Create DynamoDB table for state locking
aws dynamodb create-table \
  --table-name immobilier-saas-terraform-lock \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

### 2.2 Store Secrets in AWS Secrets Manager

```bash
# Store JWT keys
aws secretsmanager create-secret \
  --name immobilier-saas/production/jwt-private-key \
  --secret-string file://private.pem \
  --region us-east-1

aws secretsmanager create-secret \
  --name immobilier-saas/production/jwt-public-key \
  --secret-string file://public.pem \
  --region us-east-1

# Store payment provider keys
aws secretsmanager create-secret \
  --name immobilier-saas/production/wave-api-key \
  --secret-string "wave_live_sk_xxxxxxxxxx" \
  --region us-east-1

aws secretsmanager create-secret \
  --name immobilier-saas/production/twilio-auth-token \
  --secret-string "your_twilio_auth_token" \
  --region us-east-1
```

### 2.3 Deploy Infrastructure with Terraform

```bash
cd infrastructure/terraform/environments/production

# Create terraform.tfvars
cat > terraform.tfvars <<EOF
aws_region                  = "us-east-1"
db_username                 = "postgres"
db_password                 = "$(openssl rand -base64 32)"
api_image                   = "registry.fly.io/immobilier-api-prod:latest"
acm_certificate_arn         = "arn:aws:acm:us-east-1:xxxxx:certificate/xxxxx"
alert_email                 = "devops@immobilier.com"
jwt_private_key_secret_arn  = "arn:aws:secretsmanager:us-east-1:xxxxx:secret:immobilier-saas/production/jwt-private-key"
jwt_public_key_secret_arn   = "arn:aws:secretsmanager:us-east-1:xxxxx:secret:immobilier-saas/production/jwt-public-key"
wave_api_key_secret_arn     = "arn:aws:secretsmanager:us-east-1:xxxxx:secret:immobilier-saas/production/wave-api-key"
twilio_auth_token_secret_arn = "arn:aws:secretsmanager:us-east-1:xxxxx:secret:immobilier-saas/production/twilio-auth-token"
EOF

# Initialize Terraform
terraform init

# Plan infrastructure
terraform plan -out=tfplan

# Apply infrastructure
terraform apply tfplan
```

**Save the outputs!** You'll need the database endpoint, Redis endpoint, and S3 bucket name.

## Step 3: Deploy API to Fly.io (8 minutes)

### 3.1 Create Fly.io Application

```bash
cd services/api

# Login to Fly.io
flyctl auth login

# Create app
flyctl apps create immobilier-api-prod --region cdg

# Set secrets
flyctl secrets set \
  DATABASE_URL="postgres://postgres:PASSWORD@rds-endpoint:5432/immobilier_saas?sslmode=require" \
  REDIS_URL="rediss://redis-endpoint:6379/0" \
  JWT_PRIVATE_KEY="$(cat private.pem)" \
  JWT_PUBLIC_KEY="$(cat public.pem)" \
  WAVE_API_KEY="wave_live_sk_xxxxxxxxxx" \
  TWILIO_AUTH_TOKEN="your_twilio_auth_token" \
  --app immobilier-api-prod
```

### 3.2 Create fly.toml

```bash
cat > fly.toml <<EOF
app = "immobilier-api-prod"
primary_region = "cdg"

[build]
  dockerfile = "Dockerfile"

[env]
  ENVIRONMENT = "production"
  PORT = "8080"
  APP_URL = "https://api.immobilier.com"
  CORS_ORIGINS = "https://www.immobilier.com,https://app.immobilier.com"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = false
  auto_start_machines = true
  min_machines_running = 2
  processes = ["app"]

  [http_service.concurrency]
    type = "requests"
    hard_limit = 250
    soft_limit = 200

[[services.ports]]
  port = 80
  handlers = ["http"]

[[services.ports]]
  port = 443
  handlers = ["tls", "http"]

[checks]
  [checks.health]
    grace_period = "10s"
    interval = "30s"
    method = "get"
    path = "/health"
    timeout = "5s"

[[vm]]
  cpu_kind = "shared"
  cpus = 2
  memory_mb = 2048
EOF
```

### 3.3 Deploy API

```bash
# Deploy
flyctl deploy --remote-only --app immobilier-api-prod

# Verify deployment
flyctl status --app immobilier-api-prod

# Check logs
flyctl logs --app immobilier-api-prod
```

### 3.4 Run Database Migrations

```bash
# SSH into the container and run migrations
flyctl ssh console --app immobilier-api-prod

# Inside the container:
migrate -path /app/migrations -database "$DATABASE_URL" up
exit
```

## Step 4: Deploy Web to Fly.io (5 minutes)

```bash
cd ../web

# Create app
flyctl apps create immobilier-web-prod --region cdg

# Set secrets
flyctl secrets set \
  NEXT_PUBLIC_API_URL="https://api.immobilier.com" \
  API_URL="http://immobilier-api-prod.internal:8080" \
  --app immobilier-web-prod

# Create fly.toml
cat > fly.toml <<EOF
app = "immobilier-web-prod"
primary_region = "cdg"

[build]
  dockerfile = "Dockerfile"
  [build.args]
    NEXT_PUBLIC_APP_URL = "https://www.immobilier.com"
    NEXT_PUBLIC_API_URL = "https://api.immobilier.com"

[env]
  NODE_ENV = "production"
  PORT = "3000"

[http_service]
  internal_port = 3000
  force_https = true
  auto_stop_machines = false
  auto_start_machines = true
  min_machines_running = 2

[[vm]]
  cpu_kind = "shared"
  cpus = 2
  memory_mb = 2048
EOF

# Deploy
flyctl deploy --remote-only --app immobilier-web-prod

# Verify
flyctl status --app immobilier-web-prod
```

## Step 5: Configure DNS (3 minutes)

Add these DNS records to your domain:

```
# API subdomain
api.immobilier.com   CNAME   immobilier-api-prod.fly.dev

# Web application
www.immobilier.com   CNAME   immobilier-web-prod.fly.dev
immobilier.com       CNAME   immobilier-web-prod.fly.dev
```

## Step 6: Configure SSL Certificates (2 minutes)

```bash
# Add custom domain to API
flyctl certs create api.immobilier.com --app immobilier-api-prod

# Add custom domain to Web
flyctl certs create www.immobilier.com --app immobilier-web-prod
flyctl certs create immobilier.com --app immobilier-web-prod

# Check certificate status
flyctl certs show api.immobilier.com --app immobilier-api-prod
flyctl certs show www.immobilier.com --app immobilier-web-prod
```

## Step 7: Smoke Tests (2 minutes)

```bash
# Test API health
curl -f https://api.immobilier.com/health
# Expected: {"status":"ok","time":"2024-01-15T10:30:00Z"}

# Test API endpoints
curl -f https://api.immobilier.com/api/v1/listings?limit=5
# Expected: {"listings":[],"total_count":0,"page":1,"limit":5,"total_pages":0}

# Test Web
curl -f https://www.immobilier.com/
# Expected: HTML response

# Test Swagger docs
open https://api.immobilier.com/swagger/index.html
```

## Step 8: Configure Monitoring (Optional, 5 minutes)

### 8.1 Set up Sentry

```bash
# Install Sentry CLI
npm install -g @sentry/cli

# Create Sentry project
sentry-cli login

# Set Sentry DSN in environment variables
flyctl secrets set SENTRY_DSN="https://xxxxx@sentry.io/xxxxx" --app immobilier-api-prod
flyctl secrets set NEXT_PUBLIC_SENTRY_DSN="https://xxxxx@sentry.io/xxxxx" --app immobilier-web-prod
```

### 8.2 Set up Prometheus + Grafana (if using AWS)

```bash
# Prometheus is already configured to scrape /metrics endpoint
# Access Grafana dashboard
open http://$(terraform output -raw alb_dns_name):3001
# Default credentials: admin/admin
```

## Rollback Procedure

If something goes wrong:

```bash
# Rollback API
flyctl releases --app immobilier-api-prod
flyctl releases rollback <version> --app immobilier-api-prod

# Rollback Web
flyctl releases --app immobilier-web-prod
flyctl releases rollback <version> --app immobilier-web-prod

# Rollback database migration
flyctl ssh console --app immobilier-api-prod
migrate -path /app/migrations -database "$DATABASE_URL" down 1
```

## Post-Deployment Checklist

- [ ] API health check passes
- [ ] Web application loads successfully
- [ ] Can register a new user
- [ ] Can login with credentials
- [ ] Can create a listing (as landlord)
- [ ] Can view listings (public)
- [ ] Database backups are enabled
- [ ] Monitoring alerts are configured
- [ ] SSL certificates are valid
- [ ] CORS is configured correctly
- [ ] Rate limiting is working
- [ ] Error tracking is active (Sentry)

## Scaling

### Horizontal Scaling

```bash
# Scale API
flyctl scale count 5 --app immobilier-api-prod

# Scale Web
flyctl scale count 3 --app immobilier-web-prod
```

### Vertical Scaling

```bash
# Scale API VM size
flyctl scale vm shared-cpu-4x --memory 4096 --app immobilier-api-prod

# Scale Web VM size
flyctl scale vm shared-cpu-2x --memory 2048 --app immobilier-web-prod
```

### Database Scaling

```bash
# Modify RDS instance class
cd infrastructure/terraform/environments/production
terraform apply -var="rds_instance_class=db.r5.large"
```

## Troubleshooting

### API not responding

```bash
# Check logs
flyctl logs --app immobilier-api-prod

# Check status
flyctl status --app immobilier-api-prod

# SSH into container
flyctl ssh console --app immobilier-api-prod
```

### Database connection issues

```bash
# Test database connectivity
flyctl ssh console --app immobilier-api-prod
psql "$DATABASE_URL"
```

### High CPU/Memory usage

```bash
# Check metrics
flyctl metrics --app immobilier-api-prod

# Scale up
flyctl scale vm shared-cpu-4x --memory 4096 --app immobilier-api-prod
```

## Support

For issues or questions:
- Email: devops@immobilier.com
- Slack: #immobilier-devops
- On-call: PagerDuty rotation

## Estimated Timeline

| Step | Duration |
|------|----------|
| Generate JWT keys | 2 min |
| Configure AWS infrastructure | 10 min |
| Deploy API to Fly.io | 8 min |
| Deploy Web to Fly.io | 5 min |
| Configure DNS | 3 min |
| Configure SSL | 2 min |
| Smoke tests | 2 min |
| **Total** | **~32 minutes** |

---

**Last Updated:** 2024-01-15
**Version:** 1.0.0
