# _RFC #0016:_ Observabilité — Métriques, Traces et Health Checks

**date:** `2026 May 01`

---

## **Status:** `DRAFT`

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
| `go_goroutines` | Gauge | Nombre de goroutines actives (stdlib `expvar`) |

**Implémentation** : port abstrait dans `kernel/metrics/` avec implémentation via `expvar` stdlib en v1, migration vers `prometheus/client_golang` en v2 si l'écosystème Prometheus est déployé.

```go
// kernel/metrics/recorder.go
type Recorder interface {
    RecordRequest(method, route string, status int, duration time.Duration)
    RecordDBPoolStats(active, idle int)
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

### `expvar` moins puissant que Prometheus
La bibliothèque stdlib `expvar` est limitée (pas de histogrammes, pas de labels). La migration vers `prometheus/client_golang` sera nécessaire pour des métriques avancées.

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

## **Open Questions**

1. Faut-il exposer `/metrics` sur un port séparé (ex: `:9090`) pour l'isolation réseau ?
2. Quelle est la rétention recommandée pour les métriques (30 jours ? 90 jours ?) ?
3. Le `request_id` doit-il être un UUID v7 ou un identifiant plus court (ex: 8 caractères hex) pour les logs ?
4. Faut-il un dashboard Grafana préconfiguré dans le repo (comme `docker-compose.observability.yml`) ?

---

## **References**

- RFC-0002 : Architecture & Stack Technique (section 6 : Observabilité)
- RFC-0005 : Platform Audit (métriques d'audit)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Google SRE Book — Monitoring Distributed Systems](https://sre.google/sre-book/monitoring-distributed-systems/)
