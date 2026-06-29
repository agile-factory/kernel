// Package metrics exposes the observability port for the kernel.
//
// Usage:
//
//	reg := prometheus.NewRegistry()
//	rec := metrics.NewPrometheusRecorder(reg)
//	// wire rec into middlewares and repositories
//	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
package metrics

import (
	"time"
)

// Recorder is the observability port. Call its methods from HTTP middlewares,
// DB pool collectors, and the audit platform.
type Recorder interface {
	// RecordRequest increments the request counter and records duration.
	// route should be the pattern (e.g. "/api/v1/users/{userId}/profile"),
	// not the raw path, to avoid cardinality explosion.
	RecordRequest(method, route string, status int, duration time.Duration)

	// RecordDBPoolStats updates DB pool gauges. Call periodically from a
	// background goroutine that reads pgxpool.Pool.Stat().
	RecordDBPoolStats(active, idle int)

	// RecordAuditWrite increments the audit write counter.
	RecordAuditWrite(success bool)
}
