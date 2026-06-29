package server

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// DefaultRequestTimeout is the recommended upper bound for a single HTTP
// request. Wire it into TimeoutMiddleware at the router level.
const DefaultRequestTimeout = 30 * time.Second

// timeoutBody is the JSON shape returned when a request exceeds its deadline.
type timeoutBody struct {
	Error string `json:"error"`
}

// TimeoutMiddleware wraps every request with a context that carries a deadline
// of d. If the handler does not complete before the deadline, a 503 JSON
// response is sent and context cancellation propagates to the handler.
//
// Note: the middleware can only send the 503 when the handler has not yet
// called w.WriteHeader(). If the handler writes headers before timing out,
// the status code it chose is preserved — only the body will be cut short.
func TimeoutMiddleware(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			done := make(chan struct{})
			tw := &timeoutResponseWriter{w: w}

			go func() {
				next.ServeHTTP(tw, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				// Handler finished in time — flush whatever it wrote.
				tw.flush(w)
			case <-ctx.Done():
				// Deadline exceeded — write 503 if the handler has not yet
				// committed a status code.
				if !tw.headerWritten() {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					_ = json.NewEncoder(w).Encode(timeoutBody{Error: "request_timeout"})
				}
			}
		})
	}
}

// timeoutResponseWriter buffers the response so that TimeoutMiddleware can
// detect whether the handler has started writing before the deadline fires.
type timeoutResponseWriter struct {
	mu          sync.Mutex // protects all fields below
	w           http.ResponseWriter
	code        int
	buf         []byte
	wroteHeader bool
}

func (tw *timeoutResponseWriter) Header() http.Header {
	return tw.w.Header()
}

func (tw *timeoutResponseWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.wroteHeader {
		tw.code = code
		tw.wroteHeader = true
	}
}

func (tw *timeoutResponseWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.wroteHeader {
		tw.code = http.StatusOK
		tw.wroteHeader = true
	}
	tw.buf = append(tw.buf, b...)
	return len(b), nil
}

// headerWritten reports whether the handler called WriteHeader or Write.
func (tw *timeoutResponseWriter) headerWritten() bool {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	return tw.wroteHeader
}

// flush writes the buffered response to the underlying ResponseWriter.
// It must only be called once, from the goroutine that owns w.
func (tw *timeoutResponseWriter) flush(w http.ResponseWriter) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.wroteHeader {
		w.WriteHeader(tw.code)
		_, _ = w.Write(tw.buf)
	}
}
