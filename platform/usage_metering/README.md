# `platform/usage_metering/` — *réservé, non implémenté*

> ⚠️ **DEFERRED — Implémentation P2+** (profil BaaS + billing usage-based).

Ce package est intentionnellement vide. Il **réserve l'emplacement** pour la future plateforme `usage_metering` qui comptabilisera l'usage par tenant pour le billing usage-based (Stripe Metered Billing).

## Quand sera-t-il implémenté ?

P2+ du [plan de spécialisation](../../docs/plan.txt), couplé à l'arrivée du profil BaaS et de la facturation à l'usage (par requête API, par bandwidth, par storage).

L'outbox transactionnel (P0-8) servira aussi de canal d'ingestion pour les events d'usage.

## Spécification

Voir [RFC-0026 — Platform `usage_metering` — Comptage d'Usage par Tenant](../../docs/architecture/rfcs/0026-Platform-Usage-Metering.md).

Résumé : interface `Meter.Record(ctx, metric, value)`. Tables `usage_events` (event-level) + `usage_aggregates_daily` (rollups). Couplage [RFC-0020 (billing)](../../docs/architecture/rfcs/0020-Platform-Billing-Hybride-Merchant-Mode.md) pour Stripe Metered Billing.

## Pourquoi un README dans un package vide ?

Pour signaler explicitement aux contributeurs que cet emplacement est **réservé**.
