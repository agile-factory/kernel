package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// PrometheusRecorder implements Recorder using prometheus/client_golang.
// It registers all metrics on the provided *prometheus.Registry.
//
// Pass the registry to promhttp.HandlerFor to expose /metrics.
type PrometheusRecorder struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	dbPoolActive    prometheus.Gauge
	dbPoolIdle      prometheus.Gauge
	auditWrites     *prometheus.CounterVec
}

// NewPrometheusRecorder registers all kernel metrics on reg and returns the recorder.
// reg must not be nil. Pass prometheus.NewRegistry() for an isolated registry (recommended
// for tests) or prometheus.DefaultRegisterer for the global registry.
func NewPrometheusRecorder(reg prometheus.Registerer) *PrometheusRecorder {
	r := &PrometheusRecorder{
		requestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests partitioned by method, route and status code.",
		}, []string{"method", "route", "status"}),

		requestDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds (p50, p95, p99).",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),

		dbPoolActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_connections_active",
			Help: "Number of active (in-use) pgxpool connections.",
		}),

		dbPoolIdle: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "db_pool_connections_idle",
			Help: "Number of idle pgxpool connections.",
		}),

		auditWrites: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "audit_log_writes_total",
			Help: "Total number of audit log write attempts.",
		}, []string{"result"}), // result: "ok" | "error"
	}

	reg.MustRegister(
		r.requestsTotal,
		r.requestDuration,
		r.dbPoolActive,
		r.dbPoolIdle,
		r.auditWrites,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return r
}

func (r *PrometheusRecorder) RecordRequest(method, route string, status int, duration time.Duration) {
	statusStr := statusLabel(status)
	r.requestsTotal.WithLabelValues(method, route, statusStr).Inc()
	r.requestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}

func (r *PrometheusRecorder) RecordDBPoolStats(active, idle int) {
	r.dbPoolActive.Set(float64(active))
	r.dbPoolIdle.Set(float64(idle))
}

func (r *PrometheusRecorder) RecordAuditWrite(success bool) {
	if success {
		r.auditWrites.WithLabelValues("ok").Inc()
	} else {
		r.auditWrites.WithLabelValues("error").Inc()
	}
}

// statusLabel converts an HTTP status code to a concise string label.
// Using the full integer would create too many label values.
func statusLabel(code int) string {
	switch {
	case code < 200:
		return "1xx"
	case code < 300:
		return "2xx"
	case code < 400:
		return "3xx"
	case code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
