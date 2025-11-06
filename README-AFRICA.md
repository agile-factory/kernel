# 🌍 Immobilier-SaaS - Édition Afrique

> **Plateforme SaaS de location immobilière optimisée pour l'Afrique**
> Sénégal 🇸🇳 • Côte d'Ivoire 🇨🇮 • Maroc 🇲🇦

[![Version](https://img.shields.io/badge/version-1.0.0--mvp-blue.svg)](https://github.com/username/immobilier-saas)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Budget](https://img.shields.io/badge/cost-$25%2Fmonth-success.svg)](docs/deployment-mvp-budget.md)

## 🎯 Vision

Devenir la **référence SaaS africaine** pour la location courte et moyenne durée, avec :
- ⚡ **Time-to-market:** 6 semaines pour le MVP
- 📈 **Scalabilité:** 1k → 100k listings sans refactor
- 🚀 **Performance:** < 200ms p99 même sur 3G
- 💰 **Budget:** $25/mois pour démarrer

## 🌟 Spécificités Afrique

### Optimisations Réseau 3G
- **Bundle Next.js:** < 150KB (gzipped)
- **Images WebP:** < 100KB chacune
- **ISR:** 60s pour SEO + cache agressif
- **PWA:** Offline-first avec Service Worker
- **CDN:** Cloudflare (gratuit, illimité)

### Paiements Mobile Money
- 💸 **Wave** (65% market share Sénégal)
- 🍊 **Orange Money** (25%)
- 🆓 **Free Money** (10%)
- ✅ **Webhooks idempotents** (prévention duplications)
- 🔄 **Réconciliation quotidienne** automatique

### Contraintes Terrain
- Connexions intermittentes → Offline-first PWA
- Pas de cartes bancaires → Mobile money uniquement
- Signatures papier → E-signature DocuSign intégrée
- Fraude SIM-swap → 2FA email obligatoire

## 📊 Performance Targets

| Métrique | Target 3G | Target 4G | Mesure |
|----------|-----------|-----------|--------|
| **LCP** | < 3.5s | < 2.5s | Lighthouse |
| **API p99** | < 200ms | < 150ms | Prometheus |
| **Bundle size** | < 150KB | < 150KB | webpack-bundle-analyzer |
| **Image load** | < 3s | < 2s | Lighthouse |
| **Uptime** | 99.9% | 99.9% | Better Uptime |

## 🏗 Architecture Simplifiée

```
┌─────────────────────────────────────────────────────┐
│          Cloudflare CDN + WAF (Gratuit)             │
│     WebP < 100KB • Brotli • Cache 1 an              │
└─────────────────┬───────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────┐
│         Fly.io Multi-Region ($10/mois)              │
│  Johannesburg (primary) • Paris (backup)            │
│         Go API 256MB • Auto-scale 1-2               │
└─────────────────┬───────────────────────────────────┘
                  │
        ┌─────────┴──────────┐
        ▼                    ▼
┌───────────────┐    ┌────────────────┐
│  PostgreSQL   │    │ Upstash Redis  │
│  + PostGIS    │    │  (Gratuit)     │
│  ($15/mois)   │    │ 10k cmds/jour  │
└───────────────┘    └────────────────┘
        │
        ├─► Wave API (Paiements)
        ├─► Orange Money API
        ├─► Free Money API
        ├─► Twilio (SMS)
        ├─► Brevo (Email 300/jour)
        └─► AWS Rekognition (Modération photos)
```

## 🚀 Quick Start MVP (Budget $25/mois)

### Prérequis
- Docker Desktop
- Fly.io CLI (`curl -L https://fly.io/install.sh | sh`)
- Node.js 20+
- Go 1.22+

### 1. Setup Local

```bash
# Clone
git clone https://github.com/username/Immobilier_Saas.git
cd Immobilier_Saas

# Setup automatique
make setup

# Démarrer services locaux
make docker-up

# Migrations
make migrate-up

# Accéder à l'app
open http://localhost:3000
```

### 2. Deploy Production (30min)

Suivre le guide détaillé: [docs/deployment-mvp-budget.md](docs/deployment-mvp-budget.md)

**Résumé:**
```bash
# 1. Generate JWT keys
make setup

# 2. Create Fly.io app
fly launch --name immobilier-api-mvp

# 3. Set secrets
fly secrets set DATABASE_URL=postgres://... --app immobilier-api-mvp

# 4. Deploy
fly deploy --app immobilier-api-mvp

# 5. Run migrations
fly ssh console --app immobilier-api-mvp
migrate -path /app/migrations -database $DATABASE_URL up
```

**Coût total:** $25/mois (+ $25 marge)

## 💳 Configuration Paiements Mobile

### Wave (Sénégal)

```bash
# 1. Créer compte marchand Wave
# https://wave.com/en/business

# 2. Obtenir API key
# Dashboard > Developers > API Keys

# 3. Configure webhook
fly secrets set \
  WAVE_API_KEY=wave_live_sk_xxxxx \
  WAVE_WEBHOOK_SECRET=whsec_xxxxx \
  --app immobilier-api-mvp

# 4. Set webhook URL dans Wave dashboard
# https://immobilier-api-mvp.fly.dev/api/v1/webhooks/wave
```

### Orange Money (multi-pays)

```bash
fly secrets set \
  ORANGE_API_KEY=orange_xxxxx \
  ORANGE_WEBHOOK_SECRET=orange_whsec_xxxxx \
  --app immobilier-api-mvp
```

### Free Money (Sénégal)

```bash
fly secrets set \
  FREE_API_KEY=free_xxxxx \
  FREE_WEBHOOK_SECRET=free_whsec_xxxxx \
  --app immobilier-api-mvp
```

## 📱 Features MVP (Semaines 1-6)

### Core
- [x] Auth JWT + refresh token
- [x] CRUD listings avec PostGIS
- [x] Search géo + filtres
- [x] Booking avec prévention double-booking
- [x] Paiement Wave (idempotent webhooks)
- [x] PWA offline-first
- [x] Images WebP < 100KB

### Admin
- [x] Modération listings
- [x] Dashboard stats
- [x] Gestion users
- [x] Réconciliation paiements

### À venir (Semaines 7-12)
- [ ] Orange Money + Free Money
- [ ] KYC automatisé (OCR CNI)
- [ ] Prix dynamique
- [ ] QR code check-in
- [ ] Modération IA photos (Rekognition)
- [ ] Push notifications

## 🧪 Testing avec Contraintes 3G

```bash
# Run tests with 3G simulation
cd services/web
npm run test:e2e -- --network=3g

# Lighthouse audit
npm run lighthouse -- --preset=3g

# Bundle size check
npm run analyze
```

**Budgets:**
- JS bundle: < 150KB ✅
- CSS bundle: < 50KB ✅
- Images: < 100KB ✅
- LCP: < 3.5s on 3G ✅

## 📊 Monitoring

### Uptime
- **Better Uptime:** https://status.immobilier.com (gratuit)
- **Check interval:** 3min
- **Alert:** Email + SMS si > 5min downtime

### Errors
- **Sentry:** https://sentry.io/immobilier-saas (5k events/mois)
- **Sample rate:** 50% (économiser quota)

### Performance
- **Prometheus metrics:** https://api.immobilier.com/metrics
- **Grafana:** (M3+ quand budget > $150/mois)

## 💰 Coûts par Phase

| Phase | Timeline | Listings | Users | Coût/mois | Services |
|-------|----------|----------|-------|-----------|----------|
| **MVP** | S1-6 | 1,000 | 500 | **$25** | Fly.io + PG + Free tier |
| **Beta** | M3 | 5,000 | 2,500 | $60 | + Upstash Pro |
| **Launch** | M6 | 20,000 | 10,000 | $150 | + Multi-region |
| **Scale** | M12 | 100,000 | 100,000 | $400 | + Read replicas |

## 🔒 Sécurité

### Checklist MVP
- [x] HTTPS enforced (Cloudflare)
- [x] JWT RS256 (asymmetric)
- [x] Webhook signature verification (HMAC SHA256)
- [x] Rate limiting (100 req/min/IP)
- [x] SQL injection protection (parameterized queries)
- [x] XSS protection (CSP headers)
- [x] CORS restricted
- [x] Secrets in Fly secrets
- [x] 2FA email pour changements sensibles

### Fraude Prevention
- Velocity limits (max 3 paiements/heure/user)
- Blacklist Redis
- Device fingerprinting
- Fraud score (rule-based)

## 📚 Documentation

### Guides Essentiels
- [Architecture Africaine](docs/architecture-africaine.md) - Diagrammes, décisions, contraintes
- [Deployment MVP](docs/deployment-mvp-budget.md) - Guide 30min, budget $25/mois
- [Guide Développement](docs/guide-developpement-afrique.md) - Best practices 3G
- [Security Checklist](docs/security-checklist.md) - OWASP + GDPR

### API
- **Swagger:** https://api.immobilier.com/swagger/index.html
- **Postman:** [Collection](docs/postman_collection.json)

## 🤝 Contribution

```bash
# Fork & clone
git clone https://github.com/yourusername/Immobilier_Saas.git

# Create branch
git checkout -b feature/amazing-feature

# Make changes, test
make test

# Commit (conventional commits)
git commit -m "feat: add amazing feature"

# Push & PR
git push origin feature/amazing-feature
```

### Conventions
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/)
- **Code Go:** `gofmt` + `golangci-lint`
- **Code TS:** ESLint + Prettier
- **Tests:** 80%+ coverage obligatoire

## 🗺 Roadmap

### Phase 1: MVP Sénégal (S1-6) ✅
- Auth, listings, search, booking, paiement Wave
- Deploy Fly.io mono-region
- PWA offline-first

### Phase 2: Beta Multi-pays (S7-12)
- Orange Money, Free Money
- KYC automatisé
- Prix dynamique
- Côte d'Ivoire expansion

### Phase 3: Production Scale (M3-6)
- 20k listings
- Multi-region (Dakar + Paris)
- Modération IA
- ML pricing
- Mobile app PWA

### Phase 4: Expansion Régionale (M6-12)
- Maroc expansion
- 100k listings
- B2B API pour agences
- Programme fidélité
- Assurance loyer

## 👥 Équipe

- **Tech Lead:** [Your Name]
- **Full-stack:** 1 dev (S1-6)
- **Designer:** 1 (M3+)
- **DevOps:** Partagé (M6+)

## 📞 Support

- **Email:** support@immobilier.com
- **WhatsApp:** +221 XX XXX XXXX
- **Docs:** https://docs.immobilier.com
- **Status:** https://status.immobilier.com

## 📄 License

MIT License - voir [LICENSE](LICENSE)

---

**🌍 Construit avec ❤️ pour l'Afrique**

*Performance 3G • Mobile Money • Offline-first • Budget-optimisé*
