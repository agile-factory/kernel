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
// of d. The handler runs in its own goroutine and writes into a buffer
// (headers included); the buffer is copied to the real ResponseWriter only
// if the handler finishes in time. Past the deadline a 503 JSON response is
// sent, context cancellation propagates to the handler, and anything it
// writes afterwards is discarded (Write returns http.ErrHandlerTimeout).
//
// A panic in the handler is re-raised on the request goroutine, so an outer
// Recovery middleware turns it into a 500 — mount TimeoutMiddleware inside
// Recovery. Streaming responses (http.Flusher) are not supported.
func TimeoutMiddleware(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			done := make(chan struct{})
			panicked := make(chan any, 1)
			tw := &timeoutResponseWriter{h: make(http.Header)}

			go func() {
				defer func() {
					if p := recover(); p != nil {
						panicked <- p
					}
				}()
				next.ServeHTTP(tw, r.WithContext(ctx))
				close(done)
			}()

			select {
			case p := <-panicked:
				panic(p)
			case <-done:
				tw.flush(w)
			case <-ctx.Done():
				tw.timeout()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(timeoutBody{Error: "request_timeout"})
			}
		})
	}
}

// timeoutResponseWriter buffers the whole response so TimeoutMiddleware can
// discard it if the deadline fires first. Header() hands out a private map:
// the handler goroutine never touches the real ResponseWriter.
type timeoutResponseWriter struct {
	h http.Header // only touched by the handler goroutine until done

	mu          sync.Mutex // protects the fields below
	code        int
	buf         []byte
	wroteHeader bool
	timedOut    bool
}

func (tw *timeoutResponseWriter) Header() http.Header {
	return tw.h
}

func (tw *timeoutResponseWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut || tw.wroteHeader {
		return
	}
	tw.code = code
	tw.wroteHeader = true
}

func (tw *timeoutResponseWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if !tw.wroteHeader {
		tw.code = http.StatusOK
		tw.wroteHeader = true
	}
	tw.buf = append(tw.buf, b...)
	return len(b), nil
}

// timeout marks the writer as expired; later handler writes are dropped.
func (tw *timeoutResponseWriter) timeout() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.timedOut = true
}

// flush copies the buffered response to w. It must only be called once the
// handler goroutine has returned.
func (tw *timeoutResponseWriter) flush(w http.ResponseWriter) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	dst := w.Header()
	for k, v := range tw.h {
		dst[k] = v
	}
	if tw.wroteHeader {
		w.WriteHeader(tw.code)
		_, _ = w.Write(tw.buf)
	}
}
