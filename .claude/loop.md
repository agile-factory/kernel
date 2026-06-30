# Kernel Loop — État actuel (mis à jour 2026-06-29)

## Branche active : wbsitedoc
## PR active : agile-factory/kernel#61 (docs + perf + RS256)

## Statut P1 — DONE
- PR #58 (feat/p1-v0.2.0) : mergée ✓
- Modules P1bis livrés : end_customers, white_label, customer_auth, customer_notifications ✓
- Issues fermées ce jour : #51 #46 #38 #43 #41 #39 #54 ✓

## Chaque itération, dans l'ordre :

### 1. Surveiller PR #61
- `gh api repos/agile-factory/kernel/pulls/61` → vérifier state, mergeable, review_comments
- Si commentaire de review → adresser et push
- Si CI fail → diagnostiquer et corriger
- Si prête → noter pour merge (ne pas merger sans instruction explicite)

### 2. Issues ouvertes restantes
Issues v0.2.0 encore ouvertes :
- #48 RFC-0021 SSO OIDC+SAML Enterprise (complexe, dépend sign-off)
- #47 RFC-0020 Billing Hybride Stripe + Mobile Money (Wave stub en place)

Issues DEFERRED P2+ (ne pas toucher) :
- #53 RFC-0026 Usage Metering
- #52 RFC-0025 BaaS Versioning
- #50 RFC-0023 Platform Webhooks
- #49 RFC-0022 APIKey + PAT

Milestones (tracking seulement) : #18 #19 #20

### 3. Cleanup si rien d'autre
- go vet ./... → corriger tout warning
- staticcheck ./... → U1000 = hard fail
- go test ./... → signaler tout FAIL

## Règles
- Ne pas implémenter #48 (SSO) sans RFC sign-off explicite de l'utilisateur
- Ne pas merger PR sans instruction explicite
- Chaque .up.sql doit avoir son .down.sql
