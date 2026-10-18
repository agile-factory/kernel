package metrics

import (
	"net/http"
	"time"
)

// captureWriter wraps http.ResponseWriter to capture the status code.
type captureWriter struct {
	http.ResponseWriter
	status int
}

func (w *captureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware returns an http.Handler middleware that records request metrics
// via rec. The route label uses r.Pattern (Go 1.22+ ServeMux pattern) to avoid
// high-cardinality labels from raw paths.
//
//	mux := http.NewServeMux()
//	handler := metrics.HTTPMiddleware(rec)(mux)
func HTTPMiddleware(rec Recorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			cw := &captureWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(cw, r)

			// r.Pattern is set by Go 1.22+ ServeMux after routing.
			route := r.Pattern
			if route == "" {
				route = "unknown"
			}
			rec.RecordRequest(r.Method, route, cw.status, time.Since(start))
		})
	}
}

// InternalOnlyMiddleware restricts access to the wrapped handler by IP.
// Any request whose remote address is not in the loopback range receives 403.
// Caveat: behind a reverse proxy running on the same host every request
// comes from loopback — do not expose /metrics through such a proxy.
// For production use, configure METRICS_ALLOWED_CIDR and a proper network policy.
//
// This middleware is intentionally simple — production security relies on
// network-level controls (VPC, iptables). This is a defence-in-depth layer only.
func InternalOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopback(r.RemoteAddr) {
			http.Error(w, "forbidden: metrics endpoint is internal-only", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLoopback reports whether addr (host:port or host) is a loopback address.
func isLoopback(addr string) bool {
	// Strip port if present
	host := addr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			host = addr[:i]
			break
		}
		if addr[i] == ']' {
			break // IPv6 without port
		}
	}
	// Strip IPv6 brackets
	if len(host) > 0 && host[0] == '[' {
		host = host[1 : len(host)-1]
	}
	switch host {
	case "127.0.0.1", "::1", "localhost", "":
		return true
	}
	return false
}
