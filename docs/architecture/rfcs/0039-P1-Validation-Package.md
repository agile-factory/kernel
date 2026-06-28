# RFC #0039 — Paquet de Validation P1 (v0.2.0)

**date:** `2026 Jun 28`
**auteur:** Cheikh (Agile Factory)
**destinataires:** Aziz, Lucas

---

## **Status:** `EN ATTENTE DE VALIDATION`

---

## **Objet**

Ce document est un **paquet de validation** soumis avant le démarrage de l'implémentation P1 (v0.2.0). Il résume :

1. L'état actuel du kernel (P0 livré)
2. Les RFCs mis à jour pour P1 avec questions résolues
3. Les décisions critiques qui nécessitent votre approbation
4. L'ordre d'implémentation recommandé
5. Les risques identifiés

> **Action requise** : relire les RFCs amendés listés ci-dessous et signaler tout désaccord avant que l'implémentation commence. Une fois validé, les RFCs passent à `ACCEPTED` et l'implémentation démarre.

---

## 1. État du kernel — P0 (référence)

| Module | Status | Notes |
|--------|--------|-------|
| `platform/auth` | ✅ IMPL | SignupOwner, AcceptInvite, Login, Refresh, Logout, Verify Email |
| `platform/tenant` | ✅ IMPL | CRUD Tenant + TenantSettings + admin endpoints |
| `platform/membership` | ✅ IMPL | Invite via token, revoke, list paginée |
| `platform/audit` | ✅ IMPL | v1 offset + v2 cursor (RFC-0006) |
| `platform/storage` | ✅ IMPL | Port + S3 backend (R2/MinIO/AWS) |
| `kernel/events` | ✅ IMPL | Outbox at-least-once, pg_try_advisory_lock, backoff exp. |
| CI / Lint | ✅ IMPL | build/vet/test/staticcheck/lint-migrations/lint-conventions |

---

## 2. RFCs soumis pour validation P1

### 2.1 RFC-0009 — Platform `user` (Profils Utilisateurs)

**Status actuel :** PROPOSED  
**Amendements :** aucun (RFC complet, prêt pour ACCEPTED)  
**Fichier :** [`docs/architecture/rfcs/0009-Platform-User-Profils.md`](0009-Platform-User-Profils.md)

**Design retenu :**
- Table `user_profiles` (1:1 avec `users`, optionnelle)
- Profil créé automatiquement à l'inscription avec `display_name = email prefix`
- `GET/PATCH /api/v1/users/{userId}/profile`
- Nettoyage `avatar_url` délégué à RFC-0017 (object storage) via event `user.deleted`

**Question pour Aziz/Lucas :**
- [ ] Le profil 1:1 global (vs profil par tenant comme Slack) est-il suffisant pour les SaaS cibles ?

---

### 2.2 RFC-0010 — Platform `permission` (RBAC)

**Status actuel :** PROPOSED (amendé 2026-06-28)  
**Amendements majeurs :** questions ouvertes résolues + **scope v1 simplifié**  
**Fichier :** [`docs/architecture/rfcs/0010-Platform-Permission-RBAC.md`](0010-Platform-Permission-RBAC.md)

**Décisions critiques :**

| Question | Décision retenue |
|----------|-----------------|
| Rôles personnalisés | Code-defined uniquement en v1, UI-configurable en v2 (critère : un SaaS consommateur le demande) |
| Audit permissions | Oui — `permission.role_assigned` + `permission.role_revoked` via RFC-0005 |
| Cache | In-memory par requête en v1, Redis optionnel en v2 |

**Alerte technique validée :** scope v1 réduit à l'interface `PermissionChecker` branchée sur `membership.role` **sans nouvelles tables SQL**. Les 3 tables (`roles`, `role_permissions`, `user_roles`) sont différées à v2 selon critère YAGNI.

**Questions pour Aziz/Lucas :**
- [ ] Accord sur le scope v1 sans nouvelles tables ? (impact : les modules P1 utilisent l'interface `Can()` mais pas de rôles custom avant qu'un SaaS en ait besoin)
- [ ] Le critère de déclenchement v2 (issue GitHub d'un consommateur) est-il acceptable ?

---

### 2.3 RFC-0016 — Observabilité (Métriques, Traces, Health)

**Status actuel :** DRAFT → PROPOSED (amendé 2026-06-28)  
**Amendements majeurs :** questions ouvertes résolues + **abandon `expvar` au profit de `prometheus/client_golang` en v1**  
**Fichier :** [`docs/architecture/rfcs/0016-Observabilite-Metriques-Traces.md`](0016-Observabilite-Metriques-Traces.md)

**Note :** Le README référençait RFC-0029 pour l'observabilité — c'est une erreur de numérotation. RFC-0016 est la référence canonique. RFC-0029 ne sera pas créé. Le README sera corrigé à l'implémentation.

**Décisions critiques :**

| Question | Décision retenue |
|----------|-----------------|
| Bibliothèque métriques | `prometheus/client_golang` directement en v1 (pas `expvar`) |
| Port `/metrics` | Même port, middleware CIDR (`METRICS_ALLOWED_CIDR`, défaut `127.0.0.1/8`) |
| Rétention métriques | 30 jours (Prometheus) |
| `request_id` format | UUID v7 (monotone, triable, Go 1.25 natif) |
| Dashboard Grafana | Oui — `docker-compose.observability.yml` + provisioning JSON |

**Questions pour Aziz/Lucas :**
- [ ] Accord sur `prometheus/client_golang` dès v1 (dépendance directe dans `kernel/metrics`) ?
- [ ] Un `docker-compose.observability.yml` dans le repo du kernel est-il pertinent ou devrait-il être dans un repo d'infrastructure séparé ?

---

## 3. Ordre d'implémentation P1 recommandé

> Chaque étape est bloquée par la validation du RFC correspondant.

```
P1-0  PermissionChecker interface (RFC-0010 v1 simplifié)    ← pas de SQL, unbloque les autres
P1-1  Platform notification (RFC-0019)                        ← notification partiellement impl.
P1-2  Platform user (RFC-0009)                               ← user_profiles table + 2 endpoints
P1-3  Observabilité kernel (RFC-0016)                        ← prometheus/client_golang + health
P1-4  Platform billing (RFC-0020)                            ← Stripe + Wave/OM adapters
P1-5  Audit outbox upgrade (best-effort → at-least-once)     ← ADR-0009 DEPRECATED → remplacé
P1-6  Platform SSO (RFC-0021 + RFC-0028)                     ← RS256 + JWKS
```

**Pourquoi P1-0 en premier ?** La `PermissionChecker` interface (sans tables) est une dépendance transversale des use cases de notification et billing. La livrer sans SQL élimine le risque de bloquer P1-1 et P1-2.

---

## 4. Risques identifiés

| Risque | Probabilité | Impact | Mitigation |
|--------|-------------|--------|------------|
| Notification : provider email lock-in | Moyen | Haut | RFC-0019 impose un port abstrait `EmailSender` — vérifier que l'impl existante respecte ce contrat |
| Billing : Wave API instabilité | Haut | Moyen | Adapter pattern RFC-0020 ; déclencher les tests Wave sur staging uniquement |
| SSO : SAML complexité sous-estimée | Moyen | Haut | RFC-0021 scope : OIDC d'abord, SAML en v2 si un client enterprise le demande |
| Permission v2 jamais déclenchée | Faible | Faible | Acceptable — le système 3 rôles couvre 80% des SaaS B2B |

---

## 5. ADRs à écrire après implémentation

> **Convention** : les ADRs ne sont écrits qu'après implémentation, une fois les décisions effectives confirmées.

| ADR | Sujet | Déclenché par |
|-----|-------|--------------|
| ADR-0011 | `prometheus/client_golang` comme implémentation métriques v1 | Après livraison P1-3 |
| ADR-0012 | `PermissionChecker` YAGNI — pas de RBAC tables en v1 | Après livraison P1-0 |
| ADR-0013 | UUID v7 pour `request_id` (vs UUID v4 ou ULID) | Après livraison P1-3 |
| ADR-0014 | Stripe primary + Wave/OM secondary adapters | Après livraison P1-4 |

---

## 6. Pour valider

**Aziz / Lucas** : merci de répondre aux questions cochables ([ ]) ci-dessus dans chaque section RFC.

Options de réponse :
- ✅ **Accord** — RFC accepté, implémentation peut démarrer
- ⚠️ **Accord avec réserve** — accepté avec une modification mineure spécifiée
- ❌ **Désaccord** — RFC à réviser avant implémentation, préciser le point bloquant

Une fois toutes les questions résolues, les RFCs passent à `ACCEPTED` et P1 démarre.

---

## References

- [RFC-0009](0009-Platform-User-Profils.md) — Platform User Profiles
- [RFC-0010](0010-Platform-Permission-RBAC.md) — Platform Permission RBAC (amendé)
- [RFC-0016](0016-Observabilite-Metriques-Traces.md) — Observabilité (amendé)
- [RFC-0019](0019-Platform-Notification-B2B-Email-First.md) — Notification B2B Email-First
- [RFC-0020](0020-Platform-Billing-Hybride-Merchant-Mode.md) — Billing Hybride
- [RFC-0021](0021-Platform-SSO-Enterprise.md) — SSO Enterprise
- [RFC-0028](0028-JWT-RS256-JWKS.md) — JWT RS256 + JWKS
