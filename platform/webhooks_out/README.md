# `platform/webhooks_out/` — *réservé, non implémenté*

> ⚠️ **DEFERRED — Implémentation P2+** (profil BaaS).

Ce package est intentionnellement vide. Il **réserve l'emplacement** pour la future plateforme `webhooks_out` qui dispatchera les webhooks sortants signés HMAC vers les tenants.

## Quand sera-t-il implémenté ?

Quand le flag `BAAS_ENABLED=true` deviendra actif (P2+ du [plan de spécialisation](../../docs/plan.txt)).

Cependant, l'**outbox transactionnel** (RFC-0015 v1.1, implémenté en P0-8) est déjà **le terrain préparé** : `webhooks_out` en sera simplement un consommateur supplémentaire — aucune modification de l'infrastructure outbox n'est nécessaire. C'est l'un des bénéfices du choix d'architecture P0.

Le helper `kernel/webhook/signing.go` (HMAC-SHA256 sur le body + header `X-Webhook-Signature: t=<unix>,v1=<hex>` style Stripe) sera implémenté en P2 dans `kernel/`.

## Spécification

Voir [RFC-0023 — Platform `webhooks_out` — Webhooks Sortants Signés](../../docs/architecture/rfcs/0023-Platform-Webhooks-Sortants-Signes.md).

Résumé : tenants configurent `webhook_endpoints` (URL + secret HMAC + filtre `event_type`). `WebhookDispatcher` consomme l'outbox et POSTe signé. Retry exponentiel, dead-letter queue après 10 échecs, endpoint admin de replay.

## Pourquoi un README dans un package vide ?

Pour signaler explicitement aux contributeurs que cet emplacement est **réservé** et que la vision est figée.
