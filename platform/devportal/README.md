# `platform/devportal/` — *réservé, non implémenté*

> ⚠️ **DEFERRED — Implémentation P2+** (profil BaaS).

Ce package est intentionnellement vide. Il **réserve l'emplacement** pour la future plateforme `devportal` qui exposera la documentation OpenAPI auto-générée et la sandbox des développeurs externes consommateurs de la surface `/public/v1/*`.

## Quand sera-t-il implémenté ?

Quand le flag `BAAS_ENABLED=true` deviendra actif (P2+ du [plan de spécialisation](../../docs/plan.txt)).

## Spécification

Voir [RFC-0025 — Surface Publique BaaS — Versioning Calendar + OpenAPI](../../docs/architecture/rfcs/0025-Surface-Publique-BaaS-Versioning.md).

Résumé : génération OpenAPI 3.1 auto via reflection sur les routes `/public/v1/*`, calendar versioning à la Stripe (`X-API-Version: 2026-05-12`), dépréciation 12 mois minimum, SDK generation Go/TS/Python.

## Pourquoi un README dans un package vide ?

Pour signaler explicitement aux contributeurs que cet emplacement est **réservé**.
