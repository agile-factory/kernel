package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agile-factory/kernel/kernel/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
)

func newRegistry() (*prometheus.Registry, *metrics.PrometheusRecorder) {
	reg := prometheus.NewRegistry()
	rec := metrics.NewPrometheusRecorder(reg)
	return reg, rec
}

func gatherCounter(t *testing.T, reg *prometheus.Registry, name string, labels prometheus.Labels) float64 {
	t.Helper()
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != name {
			continue
		}
		for _, m := range mf.GetMetric() {
			if labelsMatch(m.GetLabel(), labels) {
				return m.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func labelsMatch(got []*dto.LabelPair, want prometheus.Labels) bool {
	if len(got) != len(want) {
		return false
	}
	for _, lp := range got {
		v, ok := want[lp.GetName()]
		if !ok || v != lp.GetValue() {
			return false
		}
	}
	return true
}

func TestPrometheusRecorder_RecordRequest(t *testing.T) {
	reg, rec := newRegistry()

	rec.RecordRequest("GET", "/api/v1/users/{userId}/profile", 200, 10*time.Millisecond)
	rec.RecordRequest("GET", "/api/v1/users/{userId}/profile", 200, 5*time.Millisecond)
	rec.RecordRequest("POST", "/api/v1/tenants", 400, 2*time.Millisecond)

	if v := gatherCounter(t, reg, "http_requests_total", prometheus.Labels{
		"method": "GET", "route": "/api/v1/users/{userId}/profile", "status": "2xx",
	}); v != 2 {
		t.Errorf("expected 2 GET 200 requests, got %v", v)
	}
	if v := gatherCounter(t, reg, "http_requests_total", prometheus.Labels{
		"method": "POST", "route": "/api/v1/tenants", "status": "4xx",
	}); v != 1 {
		t.Errorf("expected 1 POST 400 request, got %v", v)
	}
}

func TestPrometheusRecorder_RecordAuditWrite(t *testing.T) {
	reg, rec := newRegistry()

	rec.RecordAuditWrite(true)
	rec.RecordAuditWrite(true)
	rec.RecordAuditWrite(false)

	if v := gatherCounter(t, reg, "audit_log_writes_total", prometheus.Labels{"result": "ok"}); v != 2 {
		t.Errorf("expected 2 ok, got %v", v)
	}
	if v := gatherCounter(t, reg, "audit_log_writes_total", prometheus.Labels{"result": "error"}); v != 1 {
		t.Errorf("expected 1 error, got %v", v)
	}
}

func TestNoopRecorder_NoPanic(t *testing.T) {
	var rec metrics.NoopRecorder
	rec.RecordRequest("GET", "/", 200, time.Millisecond)
	rec.RecordDBPoolStats(5, 10)
	rec.RecordAuditWrite(true)
}

func TestHTTPMiddleware_RecordsMetrics(t *testing.T) {
	reg, rec := newRegistry()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := metrics.HTTPMiddleware(rec)(mux)

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify counter was incremented
	if v := gatherCounter(t, reg, "http_requests_total", prometheus.Labels{
		"method": "GET", "route": "GET /ping", "status": "2xx",
	}); v != 1 {
		t.Errorf("expected 1 request recorded, got %v", v)
	}
}

func TestInternalOnlyMiddleware_AllowsLoopback(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := metrics.InternalOnlyMiddleware(inner)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 from loopback, got %d", w.Code)
	}
}

func TestInternalOnlyMiddleware_BlocksExternal(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := metrics.InternalOnlyMiddleware(inner)

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.RemoteAddr = "203.0.113.1:54321"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 from external IP, got %d", w.Code)
	}
}

func TestMetricsEndpoint_Scrape(t *testing.T) {
	reg, rec := newRegistry()
	rec.RecordRequest("GET", "/api/v1/test", 200, 5*time.Millisecond)

	handler := metrics.InternalOnlyMiddleware(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	req := httptest.NewRequest("GET", "/metrics", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Errorf("expected http_requests_total in /metrics output")
	}
	if !strings.Contains(body, "go_goroutines") {
		t.Errorf("expected go_goroutines in /metrics output")
	}
}
