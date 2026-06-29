# _RFC #0016:_ Observabilité — Métriques, Traces et Health Checks

**date:** `2026 May 01`

---

## **Status:** `PROPOSED — EN ATTENTE DE VALIDATION (Aziz / Lucas)`

> **Amendement 2026-06-28 (Cheikh)** : Statut élevé de DRAFT à PROPOSED. Questions ouvertes résolues. Approche `expvar` abandonnée au profit de `prometheus/client_golang` directement. Note : le README référence RFC-0029 pour l'observabilité — c'est une erreur de numérotation, **ce RFC-0016 est la référence canonique**. RFC-0029 ne sera pas créé.

---

## **Summary**

Ce RFC définit la stratégie d'**observabilité** du socle Agile Factory, couvrant les trois piliers classiques :
1. **Logs** — déjà définis dans RFC-0002 (fan-out via `log/slog`)
2. **Métriques** — exposition au format Prometheus via `/metrics`
3. **Traces** — instrumentation OpenTelemetry abstraite, implémentation différée
4. **Health checks** — endpoints `/health` et `/ready` pour l'orchestration

---

## **Motivation**

Un SaaS en production sans observabilité est aveugle. Les incidents sont découverts par les clients, pas par les équipes. Le socle doit fournir les fondations d'observabilité dès le départ :

- **Métriques** : latence des endpoints, taux d'erreurs, connexions DB actives
- **Traces** : tracer une requête à travers plusieurs use cases et queries SQL
- **Health** : permettre aux load balancers et orchestrateurs (Docker Compose, K8s) de savoir si l'instance est saine

RFC-0002 mentionne ces trois piliers mais délègue leur implémentation à ce RFC.

---

## **Proposal**

### 1. Métriques Prometheus

**Endpoint** : `GET /metrics` (format texte Prometheus, accessible uniquement en interne)

**Métriques exposées** :

| Métrique | Type | Description |
|---|---|---|
| `http_requests_total` | Counter | Nombre de requêtes par méthode, route, status code |
| `http_request_duration_seconds` | Histogram | Latence des requêtes (p50, p95, p99) |
| `db_pool_connections_active` | Gauge | Connexions pgxpool actives |
| `db_pool_connections_idle` | Gauge | Connexions pgxpool idle |
| `audit_log_writes_total` | Counter | Événements audit persistés (succès + erreurs) |
| `go_goroutines` | Gauge | Nombre de goroutines actives — exposé via `prometheus.NewGoCollector()` (collecteur Go par défaut de `prometheus/client_golang`, pas `expvar`) |

**Implémentation** : port abstrait dans `kernel/metrics/` avec implémentation via `prometheus/client_golang` **dès v1**.

> **Alerte technique (amendement 2026-06-28)** : l'approche initiale (`expvar` en v1, migration Prometheus en v2) est une source de **dette technique avérée**. `expvar` ne supporte pas les histogrammes ni les labels — les métriques de latence (p50/p95/p99) sont impossibles sans labels. Migrer depuis `expvar` vers `prometheus/client_golang` implique de réécrire tous les points d'instrumentation. Verdict : **utiliser `prometheus/client_golang` directement en v1**, la bibliothèque est légère (< 1 MB), stable, et standard industry.

```go
// kernel/metrics/recorder.go
type Recorder interface {
    RecordRequest(method, route string, status int, duration time.Duration)
    RecordDBPoolStats(active, idle int)
    RecordAuditWrite(success bool)
}

// Implémentation v1 — prometheus/client_golang
type PrometheusRecorder struct {
    requestsTotal    *prometheus.CounterVec
    requestDuration  *prometheus.HistogramVec
    dbPoolActive     prometheus.Gauge
    dbPoolIdle       prometheus.Gauge
    auditWritesTotal *prometheus.CounterVec
}
```

### 2. Health Checks

```
GET /health  → liveness probe  : le processus est-il vivant ?
GET /ready   → readiness probe : l'application est-elle prête à recevoir du trafic ?
```

**Liveness** (`/health`) :

```json
{ "status": "ok", "version": "1.0.0", "uptime": "3h22m" }
```

Retourne toujours `200 OK` si le processus tourne. Utilisé par Docker pour restart automatique.

**Readiness** (`/ready`) :

```json
{
    "status": "ok",
    "checks": {
        "database": "ok",
        "migrations": "ok"
    }
}
```

Retourne `200 OK` si la DB est accessible et les migrations appliquées. Retourne `503 Service Unavailable` sinon. Utilisé par les load balancers pour retirer l'instance du pool pendant le démarrage.

### 3. Traces — OpenTelemetry (abstrait)

Un port abstrait est défini dans `kernel/tracing/` :

```go
// kernel/tracing/tracer.go
type Tracer interface {
    Start(ctx context.Context, spanName string) (context.Context, Span)
}

type Span interface {
    End()
    SetError(err error)
    SetAttribute(key string, value any)
}
```

**v1** : implémentation `NoopTracer` (zéro overhead).
**v2** : implémentation `OTelTracer` via `go.opentelemetry.io/otel` avec export vers Jaeger ou OTLP Collector.

### 4. Propagation du Trace ID

Chaque requête HTTP reçoit un `request_id` unique (UUID v7) généré par le middleware de logging. Ce `request_id` est :
- Injecté dans `context.Context`
- Inclus dans tous les logs de la requête
- Retourné dans le header de réponse `X-Request-ID`
- Inclus dans les réponses d'erreur (`trace_id` dans `AppError`)

---

## **Drawbacks**

### `/metrics` non sécurisé en v1
L'endpoint `/metrics` expose des données internes. En v1, il est protégé par réseau (accès interne uniquement via Docker network). En production, un middleware d'authentification ou un réseau privé est requis.

### ~~`expvar` moins puissant que Prometheus~~
*(Section obsolète — approche `expvar` abandonnée, voir amendement 2026-06-28. `prometheus/client_golang` est utilisé dès v1.)*

### Traces NoopTracer en v1
Sans traces actives, le debugging de latences complexes (multi-use-cases, multi-queries) reste difficile. L'activation des traces (v2) nécessite un backend (Jaeger, Tempo).

---

## **Alternatives**

### DataDog / New Relic (APM complet)
- ✅ Métriques + traces + logs dans une seule plateforme
- ❌ Coût élevé, dépendance à un SaaS externe
- ❌ Agent propriétaire à déployer
- **Verdict : compatible** — l'implémentation OTel peut exporter vers DataDog

### Grafana Stack (Prometheus + Loki + Tempo)
- ✅ Open source, self-hostable
- ✅ Standard industry
- ❌ Infrastructure à opérer (3 services)
- **Verdict : recommandé pour la production**

### Pas d'observabilité au niveau du socle
- ✅ Aucune complexité initiale
- ❌ Impossible de diagnostiquer les incidents en production
- **Verdict : rejeté**

---

## **Open Questions — RÉSOLUES (amendement 2026-06-28)**

### Q1 : Port séparé pour `/metrics` ?

**Réponse : même port, protégé par middleware IP en v1 ; port séparé optionnel via config pour K8s en v2.**

Un port dédié (`:9090`) est une bonne pratique K8s mais ajoute de la complexité réseau en dev et Docker Compose. Solution v1 : middleware `InternalOnly` sur `/metrics` qui vérifie `X-Forwarded-For` ou l'IP source et retourne `403` pour tout accès non-loopback. Configurable via `METRICS_ALLOWED_CIDR` (défaut: `127.0.0.1/8`).

### Q2 : Rétention des métriques ?

**Réponse : 30 jours (standard industrie pour alerting + post-mortems).**

90 jours est excessif pour les métriques haute fréquence (request counters) — stockage Prometheus prohibitif sans downsampling. Recommandation : Prometheus scrape toutes les 15s, rétention 30 jours. Pour l'historique long terme, Thanos ou Grafana Mimir (différé à v3+).

### Q3 : UUID v7 ou hex court pour `request_id` ?

**Réponse : UUID v7 (monotone, triable, standard).**

UUID v7 est monotone (tri chronologique natif), supporté nativement par Go 1.25 (`uuid.New()` via `google/uuid` v1.6+). Les IDs courts (8 hex) ont un risque de collision non négligeable à partir de 100k req/h. UUID v7 est la décision sans regret.

### Q4 : Dashboard Grafana préconfiguré dans le repo ?

**Réponse : oui, `docker-compose.observability.yml` avec Prometheus + Grafana + dashboard préchargé.**

Un docker-compose de dev observable accélère l'onboarding des futurs consommateurs du kernel. Le dashboard JSON (provisionné via `grafana/provisioning/dashboards/`) expose les métriques v1 sans configuration manuelle. Optionnel à l'exécution : `make observability` lance la stack, `make dev` reste inchangé.

---

## **Drawbacks — amendement**

### `/metrics` sécurisé par middleware, pas par port

La protection CIDR est simple à contourner si un proxy inverse mal configuré transmet des requêtes externes. En production, un pare-feu réseau (VPC security groups, iptables) est requis en complément. Ce point doit être documenté dans le runbook opérations.

### `prometheus/client_golang` = nouvelle dépendance Go

La bibliothèque ajoute ~1 MB au binaire. Acceptable. Elle est une dépendance directe du `kernel/metrics` package uniquement ; les repos consommateurs qui n'activent pas les métriques n'importent pas cette dépendance (via interface + injection).

---

## **References**

- RFC-0002 : Architecture & Stack Technique (section 6 : Observabilité)
- RFC-0005 : Platform Audit (métriques d'audit)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Google SRE Book — Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
