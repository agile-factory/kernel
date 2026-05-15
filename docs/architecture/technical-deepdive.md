# Deep-dive technique — invariants & subtilités

Ce document complète les RFCs en documentant les **invariants critiques** dans le code livré et les **subtilités** qui ne sautent pas aux yeux à la lecture des handlers. Cible : contributeurs kernel + ingénieurs SaaS consommateurs qui veulent comprendre **pourquoi** un mécanisme est codé tel quel.

> Les RFCs disent "ce qu'on veut" et "pourquoi". Ce doc dit "comment on l'a câblé en code, et ce qui se casse si tu touches X".

---

## 1. Transactional outbox (RFC-0015 v1.1) — pourquoi `txctx` est obligatoire {#outbox}

**Le problème classique du dual-write.** Une use case veut faire deux choses :
1. Persister un état métier (`INSERT INTO memberships`).
2. Notifier le reste du système (envoyer un email d'invitation).

Naïvement, on appellerait `repo.Save()` puis `emailSender.Send()`. Deux modes d'échec :
- Save commit puis Send crash → l'invité n'est jamais notifié.
- Send réussit puis Save rollback → un email envoyé pour un membership inexistant.

**Solution kernel : outbox transactionnel.** L'événement est inséré dans `event_outbox` **dans la même transaction** que l'écriture business. Si le commit réussit, l'événement existe ; sinon il n'existe pas. Un dispatcher poll cette table dans une goroutine séparée et appelle les handlers en mode at-least-once.

**Mécanique côté code :**

```go
// kernel/txctx/txctx.go
func ExecInTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context) error) error {
    tx, _ := pool.BeginTx(ctx, pgx.TxOptions{})
    ctx = WithTx(ctx, tx) // installe la tx dans le context
    if err := fn(ctx); err != nil {
        tx.Rollback(ctx)
        return err
    }
    return tx.Commit(ctx)
}

// kernel/events/publisher.go
func (p *OutboxPublisher) Publish(ctx context.Context, event Event) error {
    tx, err := txctx.MustFromContext(ctx) // ← échoue si pas de tx
    if err != nil {
        return fmt.Errorf("outbox publish: %w", err)
    }
    _, err = tx.Exec(ctx, insertEventOutboxSQL, event.Type, payload, meta)
    return err
}
```

**Invariant : `Publish` HORS d'une tx renvoie `txctx.ErrNoTx`.** C'est volontaire. Si on tolérait le publish hors tx :
- L'event partirait immédiatement sur l'outbox.
- Si la suite du use case échoue, l'event reste publié → état fantôme.

**Conséquence pratique :** **toute** publication d'event doit être dans un `txctx.ExecInTx`. Pas de raccourci. Les tests qui font `publisher.Publish` sans tx doivent installer une tx mock via `txctx.WithTx(ctx, mockTx)`.

**Pourquoi pas Kafka/RabbitMQ ?** Cf. RFC-0015 §3. Outbox Postgres est suffisant pour v0.1.0 (volumes B2B faibles), zéro dépendance opérationnelle additionnelle, atomicité gratuite. Migration vers un broker n'est pas un anti-pattern — le dispatcher devient un consumer Postgres → producer broker.

---

## 2. Casse du cycle audit ↔ membership {#audit-membership-cycle}

**Le problème.** En lecture du code :

- `platform/audit` a besoin du `MembershipRepository` (pour vérifier OWNER/ADMIN sur les endpoints `GET /audit-logs`).
- `platform/membership` a besoin de `audit.EventLogger` (pour tracer `MEMBER_INVITED`, `MEMBER_REVOKED`).

Si chaque module construit ses propres repos en interne, on a un cycle : `audit.New` veut un `MembershipRepository` qui n'existe que dans `membership.Module`, lui-même construit avec `audit.EventLogger`.

**Solution.** Les repos sont **construits hors des modules**, par le consommateur :

```go
// examples/saas-b2b/main.go (étape 3)
auditRepo  := auditinfra.NewPostgresAuditRepository(db)
memberRepo := memberinfra.NewPostgresMembershipRepository(db)

auditMod   := audit.New(audit.Config{Repo: auditRepo, MembershipRepo: memberRepo, ...})
memberMod  := membership.New(membership.Config{Repo: memberRepo, AuditLogger: auditMod.EventLogger(), ...})
```

`membership.Module` **n'instancie pas** son repo — il reçoit celui partagé en Config. Pareil pour `audit`. Le cycle se résout linéairement :

```
construction des repos (zéro dépendance entre eux)
       ↓
audit.New(Repo=auditRepo, MembershipRepo=memberRepo)
       ↓
membership.New(Repo=memberRepo, AuditLogger=auditMod.EventLogger())
```

**Invariant : un repo a UNE seule instance dans le wiring.** Pas de "j'instancie un nouveau `PostgresMembershipRepository` dans `audit.New`". Au mieux ça gaspille des handles pgx ; au pire ça crée des incohérences de cache (si on en ajoute plus tard).

**Pour les contributeurs kernel :** quand tu ajoutes un nouveau module qui dépend d'un autre, tu **n'instancies pas** le repo dont tu dépends. Tu l'ajoutes à ta `Config` et c'est au main.go de le passer.

---

## 3. Audit best-effort — pourquoi fire-and-forget {#audit-best-effort}

`platform/audit/usecase/log_event.go` lance la persistance dans une **goroutine détachée** du request context :

```go
bgCtx, cancel := context.WithTimeout(
    context.WithoutCancel(ctx), // ← cancellation détachée
    5*time.Second,
)
go func() {
    defer cancel()
    if err := uc.repo.Save(bgCtx, log); err != nil {
        slog.Error("audit: best-effort save failed", "error", err, ...)
    }
}()
```

**Pourquoi `context.WithoutCancel` + goroutine ?**

| Si on faisait | Conséquence |
|---|---|
| `repo.Save(ctx, log)` synchrone | Le request path attend l'écriture audit. SLA dégradé pour rien. |
| `repo.Save(ctx, log)` async avec ctx normal | Si le client ferme la connexion, `ctx.Done()` se déclenche, l'écriture audit est annulée → trace perdue alors que le business avait commit. |
| Synchrone + ignore l'erreur | Couplage temporel inutile + le timeout pool DB plombe le request si l'audit DB rame. |

**Décision (ADR-0009) :** fire-and-forget. La signature `Execute(ctx, params)` **n'a pas** de retour d'erreur — la mention "tu ne dois pas brancher dessus" est encodée dans le type.

**Trade-off accepté :** sur un crash juste après le commit business, on perd l'entrée audit. Acceptable pour v0.1.0. P1-3 migre vers l'outbox at-least-once (ADR-0009 marquée DEPRECATED dès cette migration). À ce stade, `audit.EventLogger.Execute` deviendra un wrapper autour de `events.Publisher.Publish` et l'invariant outbox (§1) s'appliquera.

**Ne pas faire :** convertir `Execute` en synchrone "parce que c'est plus simple à tester". Les tests doivent mocker l'interface `audit.EventLogger`, pas appeler l'implémentation réelle.

---

## 4. Auth — cycle de vie des tokens {#auth-token-lifecycle}

Trois familles de tokens, chacune avec un rôle bien défini :

| Token | Stockage | TTL | Mécanisme de révocation |
|---|---|---|---|
| **Access token (JWT HS256)** | Pas stocké côté serveur | 1h (configurable) | Aucun — révocation par expiration uniquement |
| **Refresh token** | `refresh_tokens.token_hash` (SHA-256 du raw) | 30j (configurable) | `revoked_at` set à la rotation ou au logout |
| **Email verification token** | `email_verification_tokens.token` (raw) | 24h | `consumed_at` set après usage |
| **Invite token** | `invite_tokens.token` (raw) | 7j | `consumed_at` (accept) ou `revoked_at` (révoqué admin) |

**Subtilité 1 : refresh stocke le hash, les autres stockent le raw.**

- Refresh token : le user le présente à chaque `/refresh`. Si on stockait le raw, un dump DB compromet tous les sessions actifs. Hash = SHA-256(token) → le raw n'est connu que du client.
- Email verification + invite : ne sont présentés **qu'une fois** (consume immédiat). Pas de risque de replay → hashing redondant. Et `consumed_at` empêche la double-redemption.

**Subtilité 2 : rolling refresh rotation.**

Chaque `/refresh` :
1. Lookup `token_hash` dans `refresh_tokens` (partial index `WHERE revoked_at IS NULL`).
2. Génère un nouveau refresh token + l'insère.
3. Marque l'ancien `revoked_at = NOW()`.
4. Renvoie le nouveau au client.

Si un attaquant intercepte un refresh token et l'utilise avant le user légitime → le user reçoit `401 INVALID_TOKEN` au prochain refresh (son token a été révoqué). C'est un signal de compromission. P1-7 (RFC-0028) ajoutera la détection automatique (token-reuse detection) + révocation cascade des tokens du même user.

**Subtilité 3 : refresh tenant-scoped vs. non-scopé.**

- `SignupOwner` et `AcceptInvite` créent un refresh avec `tenant_id` non-nul (le tenant qu'on vient de créer/joindre).
- `Login` crée un refresh avec `tenant_id = NULL`. Le user peut être membre de N tenants ; il choisit le tenant actif par requête (via header `X-Tenant-Id` que le `TenantContext` middleware lit).

**Pour les contributeurs :** ne PAS lier les refresh tokens à un tenant au login. Ça casse les flows multi-tenant et impose un re-login pour chaque switch tenant.

---

## 5. Pagination — pourquoi deux versions {#pagination}

`GET /audit-logs` existe en `v1` (offset) ET `v2` (cursor). Cf. RFC-0006 + ADR-0010.

**v1 — offset (legacy, conservé pour compat) :**

```
GET /api/v1/tenants/{id}/audit-logs?limit=50&offset=200
→ { data: [...], total: 4231, limit: 50, offset: 200 }
```

- ✅ Simple à intégrer côté front (numéros de page).
- ❌ `COUNT(*) FROM audit_logs WHERE tenant_id = $1` scanne **toute** la table tenant à chaque page. Cher.
- ❌ Drift : si une ligne est insérée entre deux pages, on peut voir un row deux fois ou en sauter un.

**v2 — cursor (keyset, recommandé) :**

```
GET /api/v2/tenants/{id}/audit-logs?limit=50&cursor=<opaque>
→ { data: [...], total: -1, limit: 50, offset: 0, next_cursor: "...", has_more: true }
```

- ✅ Pas de `COUNT(*)` — `total = -1` est un marker.
- ✅ Index `(tenant_id, occurred_at DESC, id DESC)` → lookup en O(log n) sur n'importe quelle page.
- ✅ Pas de drift — le cursor encode `(occurred_at, id)` exactement.
- ❌ Pas de "page 4 sur 87" en UI. Acceptable pour les logs (UI scroll infini).

**Invariant : NE PAS utiliser `total` en v2.** Un client qui fait `if (total > 0) ...` casse silencieusement (`-1` est truthy). Toujours brancher sur `has_more`.

---

## 6. Multi-tenancy — shared schema {#multi-tenancy}

ADR-0003 : un seul schéma Postgres, isolation par `tenant_id` dans chaque table sensible. Pas de schéma par tenant, pas de DB par tenant.

**Trade-offs :**

| Approche | Coût | Sécurité | Quand |
|---|---|---|---|
| Shared schema (kernel) | Faible | Application enforce — le `WHERE tenant_id = $1` doit être présent dans chaque requête | < 1000 tenants, équipes dev disciplinées |
| Schema par tenant | Moyen | DB enforce via `SET search_path` | 100–10k tenants, certaines exigences GDPR |
| DB par tenant | Élevé | Total isolation | Enterprise, exigences résidence des données strictes |

**Mécanique :** chaque table sensible a `tenant_id UUID NOT NULL` + un index dessus. Les use cases reçoivent `tenant_id` en paramètre (extrait du JWT par `TenantContext` middleware).

**Invariant critique : aucune requête business ne doit oublier `WHERE tenant_id = $1`.** Pas de garde-fou DB. La revue de code est le seul filet. Convention : tous les repos `Postgres*` ont leurs queries dans `infrastructure/query.go` adjacent à l'implémentation, ce qui rend la revue facile (grep `WHERE tenant_id` doit matcher chaque SELECT métier).

**Pour les contributeurs :** quand tu ajoutes un endpoint qui lit/écrit des données business, **vérifie** que le `tenant_id` est :
1. Présent dans le JWT (sinon → `401`).
2. Inclus dans toutes les queries DB.
3. Ne peut pas être surchargé par un paramètre client (pas de `?tenant_id=other-tenant`).

---

## 7. Migrations — versioning par-package {#migrations}

ADR-0006 + RFC-0038. Chaque package livre ses migrations via `//go:embed`, et `migrator.Run` les applique avec une **table de version séparée par package** :

```go
migrator.Run(dbURL, []migrator.Source{
    {Name: "events",     FS: events.Migrations,     Path: "migrations"},
    {Name: "tenant",     FS: tenant.Migrations,     Path: "migrations"},
    {Name: "auth",       FS: auth.Migrations,       Path: "migrations"},
    {Name: "membership", FS: membership.Migrations, Path: "migrations"},
    {Name: "audit",      FS: audit.Migrations,      Path: "migrations"},
})
```

Sous le capot, golang-migrate utilise `schema_migrations_events`, `schema_migrations_tenant`, etc. — **tables séparées**.

**Pourquoi ce découpage ?**

- Chaque package est versionnable indépendamment. `kernel/events` peut avoir 5 migrations, `platform/audit` 2, sans conflit de numérotation.
- Le consommateur peut désactiver un module et ne pas avoir ses tables (skip la `Source` dans le wiring).
- Update sélectif : `migrator.Run` est idempotent ; ajouter une nouvelle migration dans un package n'oblige pas à toucher les autres.

**Subtilité : ordre du slice = ordre d'application.** Si tu mets `auth` avant `tenant`, la migration `0001_create_users` échoue (FK `tenants.id` n'existe pas encore).

**Ordre canonique (cf. `examples/saas-b2b/main.go`) :**

1. `events` — la table outbox doit exister avant que les autres modules publient.
2. `tenant` — pas de dépendance.
3. `auth` — FK `users.tenant_id` (en 0001, supprimée en 0002 — mais l'ordre s'applique au moment de chaque migration).
4. `membership` — FK `memberships.user_id` + `tenant_id`.
5. `audit` — FK `audit_logs.tenant_id` + `actor_id` (en SET NULL).
6. `notification` — autonome, peut être en n'importe quelle position.

**Convention : aucune migration n'est éditée après merge.** Si la 0002 a une erreur, écrire une 0003 qui corrige. golang-migrate refuse une migration dont le hash a changé après application.

---

## 8. Logger — fan-out vers 3 fichiers + stdout {#logger}

ADR-0005. `kernel/logger.New` ouvre 4 sinks parallèles :

| Sink | Contenu | Format |
|---|---|---|
| `stdout` | Tout | Text en dev, JSON en staging/prod |
| `logs/app.log` | Tout | JSON |
| `logs/error.log` | Niveau ≥ ERROR | JSON |
| `logs/server.log` | Records avec attribut `source="http"` | JSON |

**Avantages :**
- `app.log` = audit complet, parseable, conservation long terme.
- `error.log` = triage incident — un `tail -f` direct.
- `server.log` = access log équivalent — alimente les outils de monitoring.
- `stdout` = compatibilité Docker / journald sans config.

**Rotation :** non gérée par le logger. Confier à `logrotate` ou aux log drivers Docker. Les fichiers sont `O_APPEND`, donc rotation externe (signal SIGUSR1, reopen) compatible si tu rebranches le closer.

**Filtrage `source="http"` :** c'est l'attribut posé par le middleware `delivery/http/middleware.Logger`. Quand un autre code (ex. un cron) appelle `log.Info("...")` sans cet attribut, ça atterrit dans `app.log` seulement, pas dans `server.log`.

---

## 9. AppError — invariants {#apperror}

`kernel/errors`. Toute erreur sortant d'une use case/handler est ou wrap un `*AppError`. Pourquoi :

- **HTTP code** : le handler n'a pas à mapper `errors.Is` vers `404`/`403`/etc — l'AppError porte son `HTTPCode`.
- **Code applicatif stable** : `INVALID_CREDENTIALS` ne changera jamais, c'est un contrat externe. Voir `kernel/errors/codes.go`.
- **Stack trace** : capturée à la construction (`errors.New`/`Wrap`), formatée pour logs uniquement (pas exposée en API en prod).
- **slog.LogValuer** : `log.Error("...", "error", appErr)` → la structure est sérialisée nativement, pas string-parsée.

**Invariant : `errors.New` capture la stack frame _de l'appelant_.** Pas du factory. Implémenté via `newSkip` qui utilise `runtime.Caller(skip+1)`. Conséquence : les sentinel vars (`ErrNotFound`, etc.) ont une stack frame inutile (frame init de package), donc préférer toujours les factories pour des erreurs avec contexte.

**Subtilité : `response.Error` unwrap via `errors.As`.** Donc tu peux wrapper :

```go
return fmt.Errorf("loading user: %w", errors.UserNotFound())
```

Et le handler renverra toujours `404 USER_NOT_FOUND`. L'unwrap walke la chaîne.

**Anti-pattern :**

```go
if err != nil {
    return errors.New("USER_NOT_FOUND", err.Error(), 404, err) // ❌
}
```

`err.Error()` est probablement un message technique de pgx. Ça leak en API et fait du bruit dans les logs structurés. Préférer :

```go
return errors.UserNotFound() // factory + log la cause séparément si besoin
```

---

## 10. Pourquoi pas de framework DI (wire, fx, dig) {#no-di}

ADR-0008. Tous nos wirings sont **manuels**, dans `main.go` du consommateur (cf. §3 pour le cycle).

**Trade-offs assumés :**

| Avec wire/fx | Sans (kernel) |
|---|---|
| ~30 lignes de wiring déclaratif | ~130 lignes de wiring impératif |
| Magie au boot (code-gen ou réflexion) | Ce qui est écrit = ce qui s'exécute |
| Erreurs au boot, parfois cryptiques | Erreurs compile-time du compilateur Go |
| Onboarding : "wire" est une boîte noire | Onboarding : lire `main.go` suffit |
| Refactor du graphe DI = changement global | Refactor = changer les paramètres `New` |

À l'échelle d'un kernel partagé (où le consommateur change rarement le wiring après stabilisation), le coût d'avoir 100 lignes en plus de wiring est anecdotique. Et tester `main.go` au sens compile-time est plus précis qu'un graphe wire.

---

## Références croisées

- [RFC-0015](rfcs/0015-Event-Bus-Interne.md) — outbox.
- [RFC-0018](rfcs/0018-Amendements-B2B-Onboarding.md) — onboarding amendé.
- [RFC-0006](rfcs/0006-Strategie-Pagination-APIs.md) — v1 vs v2.
- [RFC-0038](rfcs/0038-Conventions-Package-Reutilisable.md) — conventions de module.
- [ADR-0002](adrs/0002-Clean-Architecture-Trois-Niveaux.md) — couches.
- [ADR-0003](adrs/0003-Multi-Tenancy-Shared-Schema-Memberships.md) — multi-tenancy.
- [ADR-0004](adrs/0004-Gestion-Erreurs-AppError.md) — AppError.
- [ADR-0006](adrs/0006-Migrations-SQL-Golang-Migrate.md) — migrations.
- [ADR-0008](adrs/0008-Injection-Dependances-Manuelle.md) — DI manuelle.
- [ADR-0009](adrs/0009-Audit-Best-Effort-Platform.md) — audit fire-and-forget.
