# Kernel Loop — Mandatory Engineering Workflow

Working directory: /home/zone01/kernel
Branch de travail: feat/p1-v0.2.0 (puis main après merge PR #58)

## Workflow obligatoire

```
RFC PROPOSED → Review Aziz + Lucas → RFC ACCEPTED → Implement → ADR → Review Lucas → Merge
```

## Chaque itération, dans l'ordre :

### 1. CI Watch
- Vérifier PR #58 (agile-factory/kernel) : `gh pr view 58 --json statusCheckRollup`
- Si un check échoue → diagnostiquer et corriger immédiatement (staticcheck U1000 = hard fail)
- Si tous les checks sont verts et qu'il n'y a pas de review bloquante → continuer

### 2. RFC Review (Aziz + Lucas)
- Lister les RFCs en statut PROPOSED dans docs/architecture/rfcs/
- Pour chaque RFC PROPOSED sans review : lancer agents Aziz ET Lucas EN PARALLÈLE
  - Agent Aziz : reviewer architecture et cohérence système
  - Agent Lucas : reviewer sécurité, Go idioms, testabilité
- Synthétiser leurs feedbacks dans le RFC
- Si feedback mineur → passer à ACCEPTED + noter les remarques dans le RFC
- Si feedback majeur → amender le RFC avant ACCEPTED

### 3. Implémentation (RFC ACCEPTED uniquement)
Ordre d'implémentation P1bis :
  1. RFC-0033 platform/end_customers (fondation, pas de dépendances)
  2. RFC-0035 platform/white_label (dépend : tenant_settings existant)
  3. RFC-0034 platform/customer_auth (dépend : end_customers)
  4. RFC-0036 platform/customer_notifications (dépend : end_customers + white_label)

Pour chaque module :
  - migrations SQL (up + down pair obligatoire)
  - domain/ (entités + repository port + erreurs)
  - infrastructure/ (postgres-repo + query.go)
  - usecase/ (un fichier par use case)
  - delivery/http/ (routes.go + handlers)
  - tests (unit + integration stubs)
  - go test ./platform/<module>/... → PASS obligatoire
  - staticcheck ./platform/<module>/... → 0 erreur obligatoire

### 4. ADR
- Après chaque module implémenté → créer docs/architecture/adrs/NNNN-<titre>.md
- Documenter : décision, alternatives rejetées, conséquences

### 5. Code Review Lucas
- Lancer agent Lucas en mode code review sur les fichiers modifiés
- Adresser tous les commentaires pertinents
- Commit des corrections

### 6. Commit + Push
- Un commit par étape logique (RFC update, implémentation, ADR, fixes review)
- Push vers feat/p1-v0.2.0

## Règles invariantes
- NE JAMAIS implémenter avant que le RFC soit ACCEPTED
- Chaque .up.sql doit avoir son .down.sql (lint-migrations.sh = hard fail)
- staticcheck U1000 = corriger immédiatement
- go test doit passer avant tout commit d'implémentation
- Pas de dépendances externes non approuvées (stdlib-first)
