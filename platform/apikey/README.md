# `platform/apikey/` — *réservé, non implémenté*

> ⚠️ **DEFERRED — Implémentation P2+** (profil BaaS).

Ce package est intentionnellement vide. Il **réserve l'emplacement** pour la future plateforme `apikey` qui gérera les API Keys et Personal Access Tokens scopés, consommés par la surface publique `/public/v1/*` (profil BaaS).

## Quand sera-t-il implémenté ?

Quand le flag `BAAS_ENABLED=true` deviendra actif (P2+ du [plan de spécialisation](../../docs/plan.txt)). Le terrain est déjà préparé en P0/P1 :
- Middleware d'auth pluggable (`WithJWT` / `WithCustomerJWT` existent ; `WithAPIKey` s'y ajoutera sans refactor).
- `kernel/ratelimit/` (implémenté en P1) sera réutilisé par cle API key.
- Outbox transactionnel (P0-8) sera consommé par un futur `WebhookDispatcher` (RFC-0023).

## Spécification

Voir [RFC-0022 — Platform `apikey` — API Keys & PAT Scopés](../../docs/architecture/rfcs/0022-Platform-APIKey-PAT-Scopes.md).

Résumé : `pk_live_xxxx` / `pk_test_xxxx`, hash bcrypt + prefix visible, scopes alignés RBAC (RFC-0010), rotation, expiration optionnelle, environnement live/test, métriques `api_key_requests_total{tenant_id, api_key_id}`.

## Pourquoi un README dans un package vide ?

Pour signaler explicitement aux contributeurs que cet emplacement est **réservé** et que la vision est figée. Évite la création accidentelle d'un package concurrent ou d'un découpage ad-hoc.
