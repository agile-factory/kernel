# Business Plan — Lokatik SaaS

## 1. Resume executif

**Lokatik** est une plateforme SaaS B2B de gestion locative conçue spécifiquement pour le marché immobilier sénégalais. La solution permet aux agences immobilières et aux propriétaires de gérer leurs biens, locataires, baux et paiements de manière centralisée et digitale.

**Proposition de valeur :** Remplacer la gestion papier et les tableurs Excel par une solution cloud moderne, intégrant les moyens de paiement locaux (Orange Money, Wave) et les canaux de communication privilégiés (SMS, WhatsApp).

**Marché cible :** Plus de 2 000 agences immobilières formelles à Dakar et environ 5 000 propriétaires gérant 5 biens ou plus dans la région de Dakar.

**Modèle économique :** Abonnement mensuel SaaS avec trois forfaits adaptés à la taille du portefeuille immobilier.

---

## 2. Analyse du marche

### 2.1 Taille du marché

| Segment | Volume estimé | Potentiel |
|---------|--------------|-----------|
| Agences immobilières formelles (Dakar) | ~2 000 | Cible principale, besoin immédiat de digitalisation |
| Propriétaires avec 5+ biens (Dakar) | ~5 000 | Cible secondaire, sensibles au gain de temps |
| Agences immobilières (hors Dakar) | ~1 500 | Expansion phase 2 (Thiès, Saint-Louis, Mbour) |
| Promoteurs immobiliers | ~200 | Cible premium avec besoins spécifiques |

**Marché adressable total (TAM) :** ~8 700 clients potentiels au Sénégal.

**Marché adressable serviceable (SAM) :** ~3 500 clients (Dakar, agences + grands propriétaires).

**Marché capturable (SOM) :** 400 clients en 24 mois (objectif réaliste de 11% de pénétration sur Dakar).

### 2.2 Problèmes identifiés

| Problème | Impact | Fréquence |
|----------|--------|-----------|
| **Gestion papier/Excel** | Erreurs de suivi, perte de documents, pas de vision consolidée | 80% des agences |
| **Impayés non trackés** | Perte de revenus, pas d'alerte automatique, relance manuelle inefficace | 60% des agences perdent 10-15% de revenus |
| **Pas de visibilité temps réel** | Impossible de connaître la situation financière à un instant T | 90% des agences |
| **Communication locataire inefficace** | Appels téléphoniques manuels, pas de trace écrite | 75% des agences |
| **Pas de quittances systématiques** | Non-conformité, litiges fréquents | 70% des agences |
| **Reporting fiscal laborieux** | Préparation des déclarations chronophage et source d'erreurs | 85% des agences |

### 2.3 Analyse concurrentielle

| Solution | Type | Limites |
|----------|------|---------|
| **Excel / Cahiers** | Manuel | Pas de collaboration, erreurs, pas d'alertes |
| **Logiciels desktop (Sage, etc.)** | Licence locale | Coûteux, pas cloud, pas adapté au marché local |
| **Solutions SaaS françaises (Rentila, etc.)** | Cloud | Pas de Mobile Money, pas de SMS local, interface en français de France |
| **Développements sur mesure** | Custom | Coûteux à maintenir, pas de mises à jour, dépendance au développeur |
| **Lokatik** | SaaS cloud | **Solution complète, locale, intégrée Mobile Money et SMS** |

**Avantages concurrentiels de Lokatik :**
- Intégration native Orange Money et Wave (90%+ des transactions au Sénégal)
- Notifications SMS et WhatsApp en wolof/français
- Interface adaptée aux pratiques locales (montants en FCFA, baux à la sénégalaise)
- Prix accessible en FCFA, pas de conversion de devises
- Support local et formation en présentiel

---

## 3. Produit

### 3.1 Forfaits

| Forfait | Prix mensuel | Biens max | Utilisateurs | Fonctionnalités |
|---------|-------------|-----------|-------------|-----------------|
| **Starter** | 15 000 FCFA/mois | 20 biens | 1 utilisateur | CRUD biens/locataires/baux, paiements manuels, quittances PDF, dashboard basique |
| **Pro** | 35 000 FCFA/mois | 100 biens | 3 utilisateurs | Tout Starter + intégration Mobile Money, alertes SMS automatiques, exports comptables, reporting avancé |
| **Agence** | 50 000 FCFA/mois | Illimité | Illimité | Tout Pro + multi-agences, rôles personnalisés, API d'intégration, support prioritaire, formation incluse |

### 3.2 Fonctionnalités par phase

**Phase 1 — MVP (Mois 1-3) :**
- Inscription et gestion d'organisation
- CRUD complet des biens immobiliers
- Gestion des locataires avec dossier
- Création et suivi des baux
- Enregistrement manuel des paiements
- Génération de quittances PDF
- Dashboard avec KPIs essentiels

**Phase 2 — Croissance (Mois 4-8) :**
- Intégration Orange Money
- Intégration Wave
- Notifications SMS automatiques (rappels, confirmations)
- Détection automatique des impayés
- Export comptable (Excel, CSV)
- Recherche et filtres avancés

**Phase 3 — Scale (Mois 9-18) :**
- Application mobile (React Native)
- Multi-agences et gestion de franchises
- API publique pour intégrations tierces
- Reporting avancé et analytics
- Intégration WhatsApp Business
- Signature électronique des baux

---

## 4. Projections financieres — 24 mois

### 4.1 Hypothèses

| Paramètre | Valeur |
|-----------|--------|
| Répartition forfaits | 50% Starter, 35% Pro, 15% Agence |
| ARPU (revenu moyen par client) | ~30 000 FCFA/mois |
| Churn mensuel | 5% (mois 1-6), 3% (mois 7-12), 2% (mois 13-24) |
| Coût d'acquisition client (CAC) | ~50 000 FCFA |
| Lifetime Value (LTV) | ~720 000 FCFA (24 mois à ARPU moyen) |
| Ratio LTV/CAC | 14.4x |

### 4.2 Croissance par phase

#### Mois 1-3 : Phase pilote gratuit

| Mois | Agences actives | Revenu mensuel | Notes |
|------|----------------|---------------|-------|
| 1 | 5 | 0 FCFA | Recrutement beta testeurs, démarchage terrain |
| 2 | 8 | 0 FCFA | Itérations produit basées sur les retours |
| 3 | 10 | 0 FCFA | Validation du product-market fit, préparation du lancement commercial |

**Objectif :** Valider le produit avec 10 agences beta, collecter les retours, corriger les bugs critiques.

#### Mois 4-6 : Lancement commercial

| Mois | Agences payantes | Revenu mensuel | Notes |
|------|-----------------|---------------|-------|
| 4 | 15 | 450 000 FCFA | Conversion des beta + premiers clients payants |
| 5 | 30 | 900 000 FCFA | Montée en puissance du démarchage |
| 6 | 50 | 1 500 000 FCFA | Objectif break-even atteint |

**Revenu moyen mois 4-6 :** ~1 750 000 FCFA/mois en fin de période.

#### Mois 7-12 : Croissance

| Mois | Agences payantes | Revenu mensuel | Notes |
|------|-----------------|---------------|-------|
| 7 | 70 | 2 100 000 FCFA | Bouche-à-oreille + marketing digital |
| 8 | 90 | 2 700 000 FCFA | Partenariat FNAIM Sénégal |
| 9 | 110 | 3 300 000 FCFA | Intégration Mobile Money → argument de vente fort |
| 10 | 125 | 3 750 000 FCFA | Expansion Thiès et Mbour |
| 11 | 140 | 4 200 000 FCFA | Référencement par les premiers clients satisfaits |
| 12 | 150 | 4 500 000 FCFA | Bilan année 1 |

**Revenu moyen mois 7-12 :** ~5 250 000 FCFA/mois en fin de période.

#### Mois 13-24 : Scale

| Mois | Agences payantes | Revenu mensuel | Notes |
|------|-----------------|---------------|-------|
| 13 | 170 | 5 100 000 FCFA | Lancement app mobile |
| 15 | 220 | 6 600 000 FCFA | Forfait Agence gagne en traction |
| 18 | 300 | 9 000 000 FCFA | Expansion régionale (Saint-Louis, Ziguinchor) |
| 21 | 350 | 10 500 000 FCFA | API publique → intégrations partenaires |
| 24 | 400 | 12 000 000 FCFA | Objectif fin année 2 |

**Revenu cible mois 24 :** ~15 000 000 FCFA/mois (avec augmentation de l'ARPU grâce à la migration vers les forfaits supérieurs).

### 4.3 Récapitulatif annuel

| Indicateur | Année 1 | Année 2 |
|------------|---------|---------|
| Agences payantes (fin de période) | 150 | 400 |
| Revenu annuel | ~25 000 000 FCFA | ~120 000 000 FCFA |
| ARPU mensuel moyen | 30 000 FCFA | 37 500 FCFA |
| Taux de churn | 4% | 2% |

---

## 5. Structure des couts

### 5.1 Coûts mensuels

| Poste | Mois 1-6 | Mois 7-12 | Mois 13-24 |
|-------|----------|-----------|------------|
| **Serveurs (VPS + CDN)** | 100 000 FCFA | 150 000 FCFA | 300 000 FCFA |
| **SMS transactionnels** | 50 000 FCFA | 150 000 FCFA | 400 000 FCFA |
| **Marketing digital** | 200 000 FCFA | 500 000 FCFA | 1 000 000 FCFA |
| **Domaine + SSL + SaaS tools** | 30 000 FCFA | 30 000 FCFA | 50 000 FCFA |
| **Frais Mobile Money (commissions)** | 0 FCFA | 100 000 FCFA | 300 000 FCFA |
| **Support client** | 0 FCFA | 200 000 FCFA | 500 000 FCFA |
| **Total mensuel** | **380 000 FCFA** | **1 130 000 FCFA** | **2 550 000 FCFA** |

### 5.2 Coûts salariaux (équipe)

| Poste | Mois 1-6 | Mois 7-12 | Mois 13-24 |
|-------|----------|-----------|------------|
| Développeur full-stack (fondateur) | 0 FCFA (equity) | 0 FCFA (equity) | 500 000 FCFA |
| Développeur junior | 0 FCFA | 300 000 FCFA | 400 000 FCFA |
| Commercial terrain | 150 000 FCFA | 250 000 FCFA | 350 000 FCFA |
| Support client | 0 FCFA | 150 000 FCFA | 200 000 FCFA |
| **Total salaires** | **150 000 FCFA** | **700 000 FCFA** | **1 450 000 FCFA** |

### 5.3 Coût total vs Revenu

| Période | Coûts totaux mensuels | Revenu mensuel | Résultat |
|---------|----------------------|----------------|----------|
| Mois 1-3 | 530 000 FCFA | 0 FCFA | -530 000 FCFA |
| Mois 4-6 | 530 000 FCFA | 950 000 FCFA (moyenne) | +420 000 FCFA |
| Mois 7-12 | 1 830 000 FCFA | 3 400 000 FCFA (moyenne) | +1 570 000 FCFA |
| Mois 13-24 | 4 000 000 FCFA | 8 500 000 FCFA (moyenne) | +4 500 000 FCFA |

---

## 6. Break-even

**Point mort atteint : Mois 5-6**

Avec un coût fixe mensuel d'environ 530 000 FCFA en phase de lancement et un ARPU de 30 000 FCFA :

- **Nombre de clients pour le break-even :** 530 000 / 30 000 = **18 clients payants**
- **Projection mois 5 :** 30 clients payants → **rentable dès le mois 5**

Le break-even structurel (incluant les salaires) est atteint au **mois 6** avec environ 50 clients payants.

---

## 7. Strategie d'acquisition

### 7.1 Démarchage terrain (Canal principal — Mois 1-6)

| Action | Détails | Coût |
|--------|---------|------|
| Visites agences Dakar | 5-10 visites/semaine dans les quartiers d'affaires (Plateau, Almadies, Mermoz) | Transport + flyers |
| Démonstrations en direct | Présentation live de la plateforme sur tablette | Gratuit |
| Offre d'essai | 1 mois gratuit pour les 50 premières agences | Coût d'opportunité |
| Formation sur site | 2h de formation gratuite à l'inscription | Temps commercial |

### 7.2 Partenariats stratégiques (Mois 4-12)

| Partenaire | Type de partenariat | Bénéfice |
|------------|-------------------|----------|
| **FNAIM Sénégal** | Partenariat officiel | Accès aux membres, crédibilité, co-branding |
| **CNAIM** | Recommandation | Référencement auprès des agences membres |
| **Notaires** | Complémentarité | Recommandation mutuelle pour la gestion des baux |
| **Banques (CBAO, SGBS)** | Intégration | Prélèvements automatiques, prêts immobiliers |

### 7.3 Marketing digital (Mois 3-24)

| Canal | Stratégie | Budget mensuel |
|-------|-----------|---------------|
| **Facebook/Instagram** | Publicités ciblées géo (Dakar), témoignages clients vidéo | 100 000 FCFA |
| **Google Ads** | Mots-clés : "gestion locative Dakar", "logiciel immobilier Sénégal" | 50 000 FCFA |
| **LinkedIn** | Articles thought leadership, connexions avec gérants d'agences | 30 000 FCFA |
| **WhatsApp Business** | Diffusion d'informations produit, support avant-vente | 20 000 FCFA |
| **SEO / Blog** | Articles sur la gestion locative au Sénégal, guides pratiques | Temps rédaction |

### 7.4 Programme de parrainage (Mois 6+)

- Chaque client qui parraine une agence reçoit **1 mois gratuit**
- L'agence parrainée reçoit **50% de réduction le premier mois**
- Objectif : 20% des nouvelles inscriptions via parrainage

---

## 8. Risques et mitigations

| Risque | Probabilité | Impact | Mitigation |
|--------|------------|--------|------------|
| Adoption lente (résistance au changement) | Élevée | Fort | Formation gratuite, interface ultra-simple, support réactif |
| Concurrence d'une solution internationale | Moyenne | Moyen | Avantage local (Mobile Money, prix FCFA, support terrain) |
| Instabilité des API Mobile Money | Moyenne | Fort | Mode dégradé (saisie manuelle), multi-provider |
| Problèmes de connectivité internet | Élevée | Moyen | Mode offline partiel (PWA), interface légère |
| Churn élevé | Moyenne | Fort | Onboarding guidé, check-ins réguliers, valeur démontrée rapidement |
| Réglementation (protection des données) | Faible | Moyen | Conformité CDP (Commission des Données Personnelles du Sénégal) |

---

## 9. Equipe fondatrice

| Rôle | Responsabilités |
|------|----------------|
| **CEO / CTO** | Développement produit, architecture technique, stratégie |
| **Commercial terrain** | Démarchage agences, formations, support client |

**Recrutements prévus :**
- Mois 6 : Développeur junior (maintenance + nouvelles fonctionnalités)
- Mois 9 : Support client dédié
- Mois 12 : Deuxième commercial (expansion régionale)
- Mois 18 : Développeur mobile (application React Native)

---

## 10. Indicateurs cles de performance (KPIs)

| KPI | Objectif Mois 6 | Objectif Mois 12 | Objectif Mois 24 |
|-----|-----------------|------------------|------------------|
| Nombre d'agences payantes | 50 | 150 | 400 |
| MRR (Monthly Recurring Revenue) | 1 500 000 FCFA | 4 500 000 FCFA | 15 000 000 FCFA |
| Churn mensuel | < 5% | < 3% | < 2% |
| NPS (Net Promoter Score) | > 30 | > 50 | > 60 |
| Taux de conversion essai → payant | > 40% | > 50% | > 60% |
| Temps moyen d'onboarding | < 2 jours | < 1 jour | < 4 heures |
| Tickets support / client / mois | < 3 | < 2 | < 1 |
