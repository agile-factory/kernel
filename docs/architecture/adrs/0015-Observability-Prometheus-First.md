# _ADR #0015:_ Observabilité — `prometheus/client_golang` dès v1, pas `expvar`

**date:** `2026 Jun 28`

---

## **Status:** `IMPLEMENTED`

---

## **Context**

RFC-0016 (DRAFT original) proposait `expvar` en v1 et migration Prometheus en v2. L'amendement du 2026-06-28 a abandonné cette approche avant implémentation (voir RFC-0016). Ce ADR documente la décision effective d'implémentation.

---

## **Decision**

`kernel/metrics/` utilise `prometheus/client_golang` directement en v1 :

- `Recorder` interface dans `recorder.go` (port abstrait — les use-cases ne dépendent pas de Prometheus directement)
- `PrometheusRecorder` dans `prometheus.go` — implémente `Recorder` via `prometheus.Registry` isolée
- `NoopRecorder` dans `noop.go` — pour les tests sans métriques
- `HTTPMiddleware` dans `middleware.go` — wrap un `http.Handler`, enregistre method/route/status/duration
- `InternalOnlyMiddleware` dans `middleware.go` — loopback CIDR guard pour `/metrics`
- `kernel/health/` — `LivenessHandler` (GET /health, toujours 200) + `ReadinessHandler` (GET /ready, 200 si DB ping OK + MarkReady() appelé)

Métriques exposées : `http_requests_total` (counter), `http_request_duration_seconds` (histogram p50/p95/p99), `db_pool_connections_active`, `db_pool_connections_idle`, `audit_writes_total`, Go runtime metrics via `prometheus.NewGoCollector()`.

---

## **Alternatives Considered**

### `expvar` en v1 (plan original RFC-0016 DRAFT)
- ✅ Zéro dépendance externe
- ❌ Pas d'histogrammes → p50/p95/p99 impossibles
- ❌ Pas de labels → impossible de distinguer les routes dans les métriques
- ❌ Migration vers Prometheus = réécrire tous les points d'instrumentation
- **Verdict : abandonné avant implémentation**

### OpenTelemetry Metrics (OTel)
- ✅ Standard futur, vendor-neutral
- ❌ API instable (semantic conventions encore en flux en 2026)
- ❌ 5x plus de dépendances que prometheus/client_golang
- **Verdict : différé à v2+** — l'interface `Recorder` permet de swapper l'implémentation

---

## **Consequences**

### **Pros**
- Métriques opérationnelles dès v1 (latence, throughput, erreurs par route)
- L'interface `Recorder` isole les use-cases de Prometheus — un SaaS qui n'active pas les métriques n'importe pas la dépendance
- `GET /metrics` protégé par loopback guard — pas exposé accidentellement en production
- `GET /ready` permet des rolling deployments K8s zero-downtime (readiness gate avant MarkReady())

### **Cons**
- Nouvelle dépendance directe `prometheus/client_golang` dans `kernel/metrics`
- Le `NoopRecorder` peut masquer des bugs de wiring en test si utilisé à tort
- `InternalOnlyMiddleware` basé sur l'IP est facilement bypassable derrière un reverse proxy mal configuré — la documentation runbook doit le préciser
