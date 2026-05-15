# Runbook opérationnel — `kernel` v0.1.0

Procédures d'exploitation pour un SaaS qui consomme `github.com/AgileFactory/kernel`. Cible : oncall, SRE, ops. Chaque section indique l'**incident**, le **diagnostic**, et la **résolution** — pas de prose narrative.

> **Prérequis** : accès psql à la DB de prod, accès aux logs (slog JSON), accès au pool de variables d'env / secret manager.

---

## Sommaire

1. [Déploiement standard](#deploy)
2. [Migration de schéma — checklist zero-downtime](#migrations)
3. [Rollback applicatif](#rollback-app)
4. [Rollback migration](#rollback-migration)
5. [Rotation `JWT_SECRET`](#rotate-jwt)
6. [Outbox bloqué — events en quarantaine](#outbox-stuck)
7. [Refresh tokens — invalidation de masse](#refresh-flush)
8. [Suppression d'un tenant (RGPD / Right-to-erasure)](#tenant-purge)
9. [`db-reset` accidentel en prod — récupération](#db-reset-recovery)
10. [Audit gaps après crash](#audit-gaps)
11. [Métriques à surveiller](#metrics)

---

## 1. Déploiement standard {#deploy}

**Pré-conditions :**
- CI verte sur `main` (build + tests + staticcheck + lint-conventions + lint-migrations).
- Migrations ajoutées (s'il y en a) sont **rétro-compatibles** avec la version N-1 (cf. §2).

**Procédure :**

1. Build de l'image Docker du consommateur (le kernel n'a pas d'image, c'est une lib).
2. Push vers registry.
3. Rolling deploy (k8s ou équivalent). L'app applique automatiquement les migrations au boot via `migrator.Run`.
4. Health check : `GET /health` (à câbler dans le consommateur — kernel n'en fournit pas, RFC à venir).
5. Observer les logs pendant 5 min pour :
   - `outbox dispatcher starting` → bus actif.
   - Aucun `outbox dispatcher` `failed to acquire lock` (signe que plusieurs instances tentent d'être writer, mais c'est OK — single-writer enforcé par `pg_try_advisory_lock`).
   - Pas d'augmentation du taux de `error` dans `error.log`.

**Sortie :**
- Bonne : trafic 100% nouvelle version, métrique d'erreur stable.
- Mauvaise : voir §3 (rollback applicatif).

---

## 2. Migration de schéma — checklist zero-downtime {#migrations}

⚠️ **Toute migration doit être rétro-compatible avec la version N-1 qui tourne pendant le rolling deploy.**

**Checklist avant merge :**

| ❓ | Item | Si non |
|---|---|---|
| ☐ | La migration `up.sql` n'ajoute QUE des colonnes nullable ou avec default ? | Refonte requise — `ADD COLUMN NOT NULL` sans default casse les inserts existants. |
| ☐ | Pas de `DROP COLUMN` / `RENAME COLUMN` ? | Refonte en 2 PRs : (a) lire les deux noms ; (b) après deploy, supprimer l'ancien. |
| ☐ | Pas de `ALTER TYPE ... DROP VALUE` sur un enum ? | Postgres ne supporte pas. Migration impossible — refonte enum (créer nouveau type, migrer, drop ancien). |
| ☐ | Si nouvelle table FK : la migration est **après** la table parente dans l'ordre `migrator.Run` ? | Renommer / réordonner dans le `main.go`. |
| ☐ | `down.sql` symétrique et testé localement ? | Écrire la down, tester via `make db-reset && make run`. |
| ☐ | Naming conforme `NNNN_<description>.{up,down}.sql` ? | `scripts/lint-migrations.sh` bloquera la CI. |

**Cas particuliers :**

- **Index sur grosse table** : utiliser `CREATE INDEX CONCURRENTLY`. ⚠️ `CONCURRENTLY` est incompatible avec une transaction → `golang-migrate` doit avoir le pragma `-- statement_timeout = 0` et la migration ne doit pas contenir d'autres statements.
- **Backfill data** : à faire dans une migration **séparée** de la DDL. Le backfill est dans une `up.sql` qui ne fait QUE des `UPDATE`. Garantit qu'on peut rollback la donnée sans rollback la structure.
- **Suppression de table** : 3 PRs. (1) stopper les écritures ; (2) deploy ; (3) drop dans une migration.

---

## 3. Rollback applicatif {#rollback-app}

**Quand :** régression de comportement après un deploy (5xx en hausse, latence x10, erreur métier).

**Procédure :**

1. Vérifier que la version N-1 **lit** la DB nouvelle version sans plantage (les migrations doivent être rétro-compatibles — cf. §2).
2. Si oui : `kubectl rollout undo deployment/<name>` (ou équivalent CI/CD).
3. **NE PAS** annuler les migrations DB. Une migration appliquée doit le rester (sauf cas catastrophique — §4).

**Si la nouvelle migration est responsable de la panne :** voir §4.

---

## 4. Rollback migration {#rollback-migration}

⚠️ **Action destructive — confirmer avec un humain second avant d'exécuter en prod.**

**Quand :** la migration a corrompu des données ou cassé un invariant que l'app N-1 ne tolère pas.

**Procédure :**

1. **Snapshot DB immédiat** avant toute action :
   ```bash
   pg_dump -Fc -d $DATABASE_URL > rollback-snapshot-$(date +%s).dump
   ```

2. Identifier la dernière migration appliquée :
   ```sql
   SELECT * FROM schema_migrations_<package> ORDER BY version DESC LIMIT 1;
   ```

3. Appliquer le `down.sql` correspondant :
   ```bash
   migrate -database "$DATABASE_URL" \
           -path platform/<package>/migrations \
           -table schema_migrations_<package> \
           down 1
   ```

4. Vérifier que le schéma est cohérent (`\d <table>` dans psql).
5. Rolling deploy de la version applicative N-1.
6. Documenter l'incident (rapport ≤ 24h).

**Si le `down.sql` lui-même échoue :** restaurer le snapshot avec `pg_restore`. Perte = transactions entre le snapshot et l'incident.

---

## 5. Rotation `JWT_SECRET` {#rotate-jwt}

**Quand :** secret compromis (fuite, employé parti, dump DB attaquant).

**Conséquences immédiates :** **TOUS les access tokens en cours sont invalidés** dès le redéploiement. Les users devront se re-login. Les refresh tokens en DB **survivent** (ils ne sont pas signés par JWT_SECRET, juste hashés SHA-256 stockés en clair côté serveur).

**Procédure :**

1. Générer un nouveau secret (≥ 32 bytes) :
   ```bash
   openssl rand -base64 32
   ```
2. Le déposer dans le secret manager (Vault, AWS SM, k8s Secret).
3. Mettre à jour la variable `JWT_SECRET` côté déploiement.
4. Rolling restart de toutes les instances. ⚠️ **Pas zero-downtime côté UX** : les tokens en flight pendant le restart sont rejetés (les uns signés ancien secret, les autres nouveau).
5. Communiquer aux users : "veuillez vous reconnecter". Idéalement : forcer un re-login UI-side en interceptant un 401 INVALID_TOKEN.
6. Si fuite confirmée : aussi révoquer les refresh tokens actifs :
   ```sql
   UPDATE refresh_tokens SET revoked_at = NOW() WHERE revoked_at IS NULL;
   ```

**Future-proof (P1-7, RFC-0028) :** migration vers RS256 + JWKS endpoint avec key rotation graceful (2 clés actives en parallèle, période de chevauchement). Aujourd'hui en HS256, le swap est brutal.

---

## 6. Outbox bloqué — events en quarantaine {#outbox-stuck}

**Symptôme :**
- Alert `events_outbox_quarantine_total > 0`.
- Users signalent ne pas recevoir des emails (invitation, vérification).

**Diagnostic :**

```sql
-- 1. Compteurs globaux
SELECT
  count(*) FILTER (WHERE published_at IS NOT NULL) AS published,
  count(*) FILTER (WHERE published_at IS NULL AND attempts < 10) AS pending,
  count(*) FILTER (WHERE published_at IS NULL AND attempts >= 10) AS quarantine
FROM event_outbox;

-- 2. Erreurs récentes
SELECT id, event_type, attempts, last_error, next_attempt_at
FROM event_outbox
WHERE published_at IS NULL AND attempts > 0
ORDER BY next_attempt_at DESC
LIMIT 20;
```

**Causes typiques :**

| `last_error` contient... | Cause | Action |
|---|---|---|
| `connection refused` / `timeout` | EmailSender down | Vérifier le service en aval (Postmark, SES). |
| `invalid template` | Erreur dans `platform/notification/templates/` | Hot-fix du template + redeploy + manuel replay (cf. plus bas). |
| `no handler registered` | `Subscribe` jamais appelé pour ce type | Vérifier que `notifMod.Subscribe(bus)` est avant `bus.Run` dans le main.go. |
| `context deadline exceeded` | Handler trop lent (> poll cycle) | Réduire le batch size ou paralléliser le handler. |

**Replay manuel d'un event quarantaine :**

```sql
-- Reset les attempts + déclenche un retry au prochain poll
UPDATE event_outbox
SET attempts = 0, next_attempt_at = NOW(), last_error = NULL
WHERE id = '<event_id>';
```

**Replay en masse (après fix root cause) :**

```sql
UPDATE event_outbox
SET attempts = 0, next_attempt_at = NOW(), last_error = NULL
WHERE published_at IS NULL AND attempts >= 10;
```

⚠️ Pour les events anciens (> 30j), valider que la sémantique a encore du sens (ex. envoyer un email d'invitation expirée n'a aucun intérêt).

---

## 7. Refresh tokens — invalidation de masse {#refresh-flush}

**Quand :**
- Suspicion de compromission large.
- Changement de politique (force re-login après update majeur).
- User signale activité suspecte sur son compte.

**Invalidation d'un user :**

```sql
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE user_id = '<user_uuid>' AND revoked_at IS NULL;
```

**Invalidation d'un tenant entier :**

```sql
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE tenant_id = '<tenant_uuid>' AND revoked_at IS NULL;
```

⚠️ Les **access tokens** (JWT) en cours restent valides jusqu'à leur expiration (max 1h par défaut). Pour les invalider aussi : rotation de JWT_SECRET (§5).

---

## 8. Suppression d'un tenant (RGPD) {#tenant-purge}

**Quand :** demande d'effacement RGPD ou contrat résilié.

**Sémantique :**
- **Soft-delete (default)** via `DELETE /api/v1/tenants/{id}` : `tenants.is_active = false`. Les memberships restent, les audit logs restent.
- **Hard-delete (RGPD)** : `DELETE FROM tenants WHERE id = ...`. `ON DELETE CASCADE` emporte memberships, refresh_tokens, invite_tokens, tenant_settings. `audit_logs.tenant_id` devient NULL (`ON DELETE SET NULL`).

**Procédure hard-delete (RGPD) :**

```sql
BEGIN;

-- 1. Snapshot avant action (au cas où)
COPY (SELECT * FROM tenants WHERE id = '<tenant>') TO '/tmp/tenant-<id>.csv';
COPY (SELECT * FROM memberships WHERE tenant_id = '<tenant>') TO '/tmp/memberships-<id>.csv';
COPY (SELECT * FROM audit_logs WHERE tenant_id = '<tenant>') TO '/tmp/audit-<id>.csv';

-- 2. Hard-delete (cascade)
DELETE FROM tenants WHERE id = '<tenant>';

-- 3. Anonymiser les audit logs résiduels (tenant_id devient NULL via FK SET NULL)
UPDATE audit_logs
SET metadata = jsonb_set(metadata - 'email' - 'tenant_name', '{anonymized}', 'true')
WHERE tenant_id IS NULL AND target_id::text = '<tenant>';

COMMIT;
```

⚠️ **Les emails dans `notification_logs` ne sont pas effacés automatiquement.** Selon ta politique RGPD :

```sql
UPDATE notification_logs
SET metadata = jsonb_set(metadata, '{recipient}', '"anonymized"')
WHERE tenant_id_nullable = '<tenant>';
```

---

## 9. `db-reset` accidentel en prod — récupération {#db-reset-recovery}

⚠️ **`make db-reset` est destructif** : `DROP SCHEMA public CASCADE`. Si exécuté en prod par erreur :

**Procédure immédiate :**

1. **STOP les writes** : kill l'app applicative immédiatement (`kubectl scale deployment app --replicas=0`).
2. **Restaurer le dernier backup** :
   ```bash
   pg_restore -Fc -d $DATABASE_URL --clean --if-exists latest-backup.dump
   ```
3. **Quantifier la perte** : RPO = depuis le dernier backup → toutes les transactions de la fenêtre sont perdues. Vérifier les logs applicatifs pour identifier les opérations entre `t_backup` et `t_reset`.
4. **Replay manuel** des opérations critiques si possible (signups, payments).
5. **Redéployer** l'app, vérifier que `migrator.Run` re-applique tout proprement (idempotent — si le backup contient déjà les `schema_migrations_*`, rien ne se passe).

**Prévention :** retirer la cible `db-reset` du Makefile en prod (ou la guarder par `$APP_ENV != production`). Sera fait dans une PR à part — voir backlog.

---

## 10. Audit gaps après crash {#audit-gaps}

**Contexte :** ADR-0009. L'audit est best-effort (fire-and-forget). Si l'app crash entre le commit business et le save audit (fenêtre ≤ 5s), la trace est perdue.

**Détection :**

```sql
-- Compter les memberships créés sans event MEMBER_INVITED ou MEMBER_ACCEPTED
SELECT count(*) FROM memberships m
WHERE NOT EXISTS (
  SELECT 1 FROM audit_logs a
  WHERE a.target_type = 'membership'
    AND a.target_id = m.id
    AND a.action IN ('MEMBER_INVITED', 'MEMBER_ACCEPTED')
);
```

**Action :** documenter le gap dans le rapport d'incident. Aucune reconstruction automatique possible — les events business sont rejouables (RFC-0015) mais pas les audits best-effort.

**Long terme :** migration de l'audit vers l'outbox at-least-once (P1-3) éliminera cette classe de gap.

---

## 11. Métriques à surveiller {#metrics}

> Le kernel ne wire pas encore Prometheus (RFC-0029, P1). En attendant, requêter directement Postgres ou parser `app.log`.

| Métrique | Requête / Source | Seuil d'alerte |
|---|---|---|
| Events pending | `SELECT count(*) FROM event_outbox WHERE published_at IS NULL` | > 1000 = enquête |
| Events quarantine | `SELECT count(*) FROM event_outbox WHERE published_at IS NULL AND attempts >= 10` | > 0 = alerte |
| Refresh tokens actifs | `SELECT count(*) FROM refresh_tokens WHERE revoked_at IS NULL AND expires_at > NOW()` | Croissance anormale = bot ? |
| Login failures rate | grep `INVALID_CREDENTIALS` dans `error.log`, agréger par user_id | > 5/min/user = brute-force ? |
| Audit save failures | grep `audit: best-effort save failed` dans `error.log` | > 0 = enquête DB |
| Dispatcher lag | `SELECT MAX(NOW() - next_attempt_at) FROM event_outbox WHERE published_at IS NULL` | > 5 min = dispatcher down |
| DB pool exhaustion | `pgxpool` stats côté app | utilisation > 80% = augmenter pool |

---

## Contacts d'escalade

> À remplir par le consommateur. Le kernel ne définit pas de procédure d'escalade — c'est org-specific.

| Sujet | Owner | Canal |
|---|---|---|
| Incident kernel (bug suspecté en lib) | Mainteneur kernel | issue privée `AgileFactory/kernel` |
| Incident DB (corruption, perf) | DBA equipe | Slack #ops-database |
| Incident sécurité (compromission credentials) | Security lead | mail security@example.com + Slack #security-urgent |
| Décision architecture / RFC en cours | Tech lead | Slack #architecture |
