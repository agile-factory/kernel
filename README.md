# 🏠 Lokatik

**La plateforme de gestion locative n°1 au Sénégal**

---

## Le Problème

Au Sénégal, la majorité des agences immobilières et des propriétaires gèrent encore leurs biens locatifs sur papier ou Excel :

- **Aucune visibilité** sur le taux d'occupation et la rentabilité
- **Loyers impayés** non suivis, relances manuelles inefficaces
- **Contrats de bail** rédigés à la main, non conformes
- **Quittances** absentes ou incomplètes
- **Aucun historique** centralisé des locataires et paiements

## La Solution

Lokatik digitalise le **cycle locatif complet** :

```
Bien → Locataire → Bail → Paiement → Quittance
```

Une seule plateforme pour gérer l'ensemble de votre patrimoine immobilier.

## Fonctionnalités MVP

- **Gestion multi-biens** : appartements, maisons, terrains, locaux commerciaux
- **Dossier locataire complet** : CNI, profession, contact d'urgence
- **Contrats de bail auto-générés** : PDF conforme au droit sénégalais
- **Suivi des paiements** : avec rappels automatiques SMS
- **Quittances de loyer** : PDF auto-générées
- **Dashboard** : taux d'occupation, revenus, impayés, rentabilité
- **Paiements mobile** : intégration Orange Money & Wave
- **Notifications** : SMS et WhatsApp

## Stack Technique

| Couche | Technologie | Justification |
|--------|-------------|---------------|
| **Frontend** | React + Vite + TypeScript | SPA rapide, typage fort, excellent DX |
| **UI** | TailwindCSS | Styling utilitaire, responsive, léger |
| **State** | Zustand | State management minimal et performant |
| **Backend** | Rust + Axum + Tokio | Performance, sécurité mémoire, async natif |
| **ORM** | SQLx | Requêtes vérifiées à la compilation, async |
| **Base de données** | PostgreSQL | Robuste, extensible, excellent pour les données relationnelles |
| **Auth** | JWT (jsonwebtoken) | Stateless, scalable |
| **Infra** | Docker + Docker Compose | Environnement reproductible |

## Installation

### Prérequis

- [Docker](https://docs.docker.com/get-docker/) & Docker Compose
- [Rust](https://rustup.rs/) (pour le développement local)
- [Node.js 20+](https://nodejs.org/) (pour le développement local)

### Démarrage rapide

```bash
# Cloner le repo
git clone https://github.com/Cheikh-Nakamoto/Immobilier_Saas.git
cd Immobilier_Saas

# Configurer l'environnement
cp backend/.env.example backend/.env

# Lancer avec Docker
make dev

# Ou sans Docker :
# Terminal 1 - Backend
cd backend && cargo run

# Terminal 2 - Frontend
cd frontend && npm install && npm run dev
```

### Accès

| Service | URL |
|---------|-----|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:3000 |
| PostgreSQL | localhost:5432 |

## Modèle Économique

| Forfait | Prix/mois | Biens | Fonctionnalités |
|---------|-----------|-------|-----------------|
| **Starter** | 15 000 FCFA | Jusqu'à 20 | Gestion de base |
| **Pro** | 35 000 FCFA | Jusqu'à 100 | + Notifications SMS |
| **Agence** | 50 000 FCFA | Illimité | + Multi-utilisateurs |

## Roadmap

### V1 — MVP (Q1 2026)
- CRUD biens, locataires, baux
- Suivi des paiements
- Dashboard avec KPIs
- Génération PDF (contrats, quittances)

### V2 — Intégrations (Q2 2026)
- Paiements Orange Money & Wave
- Notifications SMS & WhatsApp
- Import/export CSV
- Multi-utilisateurs par agence

### V3 — Scale (Q3-Q4 2026)
- Application mobile (React Native)
- Comptabilité avancée
- Expansion Côte d'Ivoire & Mali
- API publique pour intégrations tierces

## Structure du Projet

```
├── backend/          # API Rust (Axum + SQLx)
├── frontend/         # SPA React (Vite + TypeScript)
├── docs/             # Documentation technique
├── scripts/          # Scripts utilitaires
└── docker-compose.yml
```

## Licence

MIT — voir [LICENSE](LICENSE)
