# 📖 Guide d'Utilisation - Immobilier-SaaS

**Comment utiliser tous les livrables de ce projet**

---

## 🎯 Vous êtes...

### 👨‍💻 Développeur Full-Stack

**Objectif:** Implémenter le MVP en 6 semaines

**Par où commencer:**

1. **Lisez d'abord:**
   - `README-AFRICA.md` - Vue d'ensemble contexte africain
   - `QUICKSTART.md` - Setup environnement (10min)
   - `docs/architecture-africaine.md` - Comprendre les décisions

2. **Setup local:**
   ```bash
   # Clone & setup
   git clone <repo>
   make setup

   # Démarrer services
   make docker-up
   make migrate-up

   # Accéder
   open http://localhost:3000
   ```

3. **Développer:**
   - **Backend:** `docs/guide-developpement-afrique.md` - Best practices Go
   - **Frontend:** Section Next.js du même guide
   - **Database:** `services/api/migrations/` - Schema SQL
   - **API:** http://localhost:8080/swagger/index.html

4. **Tests:**
   ```bash
   # Backend
   cd services/api
   make test

   # Frontend
   cd services/web
   npm test
   npm run test:e2e -- --network=3g
   ```

5. **Références rapides:**
   - Paiement Wave: `services/api/internal/service/payment/wave_service.go`
   - Webhook idempotent: `services/api/internal/handler/webhook_handler.go`
   - PWA offline: `services/web/public/sw.js`
   - Image optimization: Guide développement section 4

---

### 🚀 DevOps / SRE

**Objectif:** Déployer en production budget $25/mois

**Par où commencer:**

1. **Lisez d'abord:**
   - `docs/deployment-mvp-budget.md` - Guide complet $25/mois
   - `docs/security-checklist.md` - Checklist avant deploy
   - `docs/deployment.md` - Runbook détaillé

2. **Setup Fly.io (10min):**
   ```bash
   # Install CLI
   curl -L https://fly.io/install.sh | sh

   # Login
   fly auth login

   # Create app
   fly launch --name immobilier-api-mvp
   ```

3. **Configure secrets:**
   ```bash
   # Generate JWT keys
   cd services/api
   openssl genrsa -out private.pem 2048
   openssl rsa -in private.pem -pubout -out public.pem

   # Set secrets
   fly secrets set \
     DATABASE_URL=postgres://... \
     JWT_PRIVATE_KEY="$(cat private.pem)" \
     JWT_PUBLIC_KEY="$(cat public.pem)" \
     WAVE_API_KEY=wave_live_sk_xxx \
     --app immobilier-api-mvp
   ```

4. **Deploy:**
   ```bash
   # API
   cd services/api
   fly deploy --app immobilier-api-mvp

   # Web
   cd services/web
   fly deploy --app immobilier-web-mvp
   ```

5. **Monitoring:**
   - Better Uptime: Setup gratuit (1 monitor)
   - Sentry: Configure DSN (5k events/mois)
   - Health: https://api.immobilier.com/health

6. **Scaling (si besoin):**
   ```bash
   # Horizontal
   fly scale count 2 --app immobilier-api-mvp

   # Vertical
   fly scale vm shared-cpu-2x --memory 512 --app immobilier-api-mvp
   ```

**Références:**
- Budget breakdown: `docs/deployment-mvp-budget.md` section "Total Cost Breakdown"
- Terraform (futur): `infrastructure/terraform/`
- CI/CD: `.github/workflows/deploy-production.yml`

---

### 🏗️ Architecte / Tech Lead

**Objectif:** Comprendre & valider l'architecture

**Par où commencer:**

1. **Documents clés:**
   - `SUMMARY.md` - Vue d'ensemble complète projet
   - `docs/architecture-africaine.md` - Architecture détaillée
   - `LIVRABLES-AFRIQUE.md` - Liste tous les livrables

2. **Diagrammes:**
   - Architecture globale: `docs/architecture-africaine.md` section "Architecture Globale"
   - Workflows métier: Mermaid diagrams dans même doc
   - Database schema: `services/api/migrations/000001_init_schema.up.sql`

3. **Décisions architecturales:**
   - Contraintes 3G: Section "Contraintes & Décisions"
   - Paiement mobile: Section "Paiement Mobile Dominant"
   - Double-booking: Section "Double-Booking Prevention"
   - Budget: `docs/deployment-mvp-budget.md`

4. **Performance:**
   - Targets: `docs/architecture-africaine.md` section "Performance Targets"
   - Optimisations: `docs/guide-developpement-afrique.md` sections 1-6
   - Cache strategy: Architecture doc section "Cache Strategy"

5. **Security:**
   - Checklist: `docs/security-checklist.md` (200+ checks)
   - Fraud prevention: Migration 000002 table `fraud_checks`
   - Webhook verification: `webhook_handler.go` signature HMAC

6. **Scaling:**
   - Capacity planning: Architecture doc section "Capacity Planning"
   - Cost projection: Deployment budget section "Upgrade Path"
   - Roadmap: `README-AFRICA.md` section "Roadmap"

---

### 💼 Product Owner / Business

**Objectif:** Comprendre le produit & le ROI

**Par où commencer:**

1. **Vision & Features:**
   - `README-AFRICA.md` - Vue d'ensemble produit
   - Features MVP: Section "Features MVP (Semaines 1-6)"
   - Roadmap 12 mois: Section "Roadmap"

2. **Budget & ROI:**
   - Coût par phase: `README-AFRICA.md` section "Coûts par Phase"
   - MVP $25/mois: `docs/deployment-mvp-budget.md`
   - Scaling costs: Deployment budget section "Scaling Plan"

3. **Time-to-market:**
   - MVP: 6 semaines
   - Beta: 12 semaines totales
   - Production: 3-4 mois
   - Scale: 6-12 mois

4. **Metrics:**
   - Performance: < 200ms p99 sur 3G
   - Uptime: 99.9% SLA
   - Capacity: 1k → 100k listings
   - Conversion: > 8% recherche → réservation

5. **Marché:**
   - Phase 1: Sénégal (Dakar, Saint-Louis, Saly)
   - Phase 2: Côte d'Ivoire (Abidjan)
   - Phase 3: Maroc (Casablanca, Marrakech)

6. **Concurrence:**
   - USP: Optimisé 3G, mobile money natif, budget accessible
   - Positionnement: SaaS africain vs solutions occidentales inadaptées

---

## 📂 Navigation Rapide par Besoin

### "Je veux comprendre l'architecture"
→ `docs/architecture-africaine.md` (40 pages, diagrammes Mermaid)

### "Je veux déployer en prod rapidement"
→ `docs/deployment-mvp-budget.md` (guide 30min, $25/mois)

### "Je veux coder les paiements mobile"
→ `services/api/internal/service/payment/wave_service.go`
→ `services/api/internal/handler/webhook_handler.go`

### "Je veux optimiser pour 3G"
→ `docs/guide-developpement-afrique.md` sections 1-4

### "Je veux voir le schema database"
→ `services/api/migrations/000001_init_schema.up.sql` (base)
→ `services/api/migrations/000002_africa_specific.up.sql` (Afrique)

### "Je veux setup mon environnement local"
→ `QUICKSTART.md` (10 minutes, copy-paste ready)

### "Je veux voir la checklist sécurité"
→ `docs/security-checklist.md` (200+ checks OWASP + GDPR)

### "Je veux connaître le budget"
→ `README-AFRICA.md` section "Coûts par Phase"
→ `docs/deployment-mvp-budget.md` section "Total Cost Breakdown"

### "Je veux voir les tests E2E 3G"
→ `docs/guide-developpement-afrique.md` section 7

---

## 🎓 Parcours d'Apprentissage

### Jour 1: Découverte
1. Lire `README-AFRICA.md` (15min)
2. Lire `SUMMARY.md` (10min)
3. Parcourir `docs/architecture-africaine.md` (30min)

**Total:** 1 heure

### Jour 2: Setup Local
1. Suivre `QUICKSTART.md` (10min setup)
2. Explorer http://localhost:3000
3. Tester API http://localhost:8080/swagger/index.html
4. Lire migrations SQL (30min)

**Total:** 2 heures

### Jour 3: Backend
1. Lire `docs/guide-developpement-afrique.md` sections Go (1h)
2. Étudier `wave_service.go` (30min)
3. Comprendre webhook idempotent (30min)
4. Écrire premier test (1h)

**Total:** 3 heures

### Jour 4: Frontend
1. Lire guide développement sections Next.js (1h)
2. Créer composant listing card (1h)
3. Implémenter recherche geo (1h)
4. Tester PWA offline (30min)

**Total:** 3.5 heures

### Jour 5: Déploiement
1. Lire `docs/deployment-mvp-budget.md` (30min)
2. Setup Fly.io compte (10min)
3. Deploy staging (30min)
4. Configure monitoring (20min)

**Total:** 1.5 heures

---

## 🔧 Commandes Essentielles

### Développement Local

```bash
# Setup initial
make setup                    # Generate JWT keys, env files

# Start services
make docker-up               # All services
make dev                     # With logs

# Database
make migrate-up              # Run migrations
make migrate-down            # Rollback
make db-shell                # PostgreSQL shell
make seed                    # Sample data

# Tests
make test                    # All tests
make test-api                # Backend only
make test-web                # Frontend only

# Code quality
make lint                    # Lint all
make format                  # Format all

# Build
make build                   # Production builds
make swagger                 # Generate API docs

# Cleanup
make clean                   # Remove artifacts
make docker-down-volumes     # Reset all
```

### Production

```bash
# Deploy
make deploy-prod             # Production (avec confirmation)
make deploy-staging          # Staging

# Monitoring
make health                  # Check all services
make logs                    # View logs

# Backup
make backup-db               # Backup PostgreSQL
make restore-db FILE=backup.sql  # Restore

# Utilities
make check-env               # Verify env config
make version                 # Show versions
```

---

## 🐛 Troubleshooting

### "Services won't start"

```bash
# Check ports
lsof -i :8080 :3000 :5432

# Reset everything
make clean-all
make docker-up
make migrate-up
```

### "Database migrations fail"

```bash
# Check connection
make db-shell

# Force version
make migrate-force VERSION=1
make migrate-up
```

### "JWT auth not working"

Vérifier:
1. Keys générées: `ls services/api/*.pem`
2. Keys dans .env: `cat services/api/.env | grep JWT`
3. Format correct (avec BEGIN/END)

### "Payment webhook fails"

Vérifier:
1. Signature verification: `webhook_handler.go:45`
2. Idempotency check: Migration 000002 table `payment_webhooks`
3. Logs: `make logs-api | grep webhook`

### "Images too large"

Vérifier:
1. Pipeline optimization: `guide-developpement-afrique.md` section 4
2. WebP conversion active
3. Target < 100KB respecté

---

## 📞 Obtenir de l'Aide

### Documentation
1. Chercher dans ce fichier d'abord
2. Consulter `SUMMARY.md` pour vue d'ensemble
3. Lire le guide spécifique à votre besoin

### Code
1. Chercher dans `services/api/` (Go) ou `services/web/` (Next.js)
2. Consulter Swagger: http://localhost:8080/swagger/index.html
3. Lire les tests: `tests/` directories

### Support
- **Email:** dev@immobilier.com
- **Slack:** #immobilier-dev
- **Docs:** https://docs.immobilier.com

---

**🎉 Vous êtes prêt !**

Choisissez votre profil ci-dessus et suivez le guide approprié.

**Version:** 1.0.0
**Dernière mise à jour:** 7 novembre 2025
