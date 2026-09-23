package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func serveTimeout(t *testing.T, d time.Duration, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	TimeoutMiddleware(d)(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	return rec
}

func TestTimeout_FastHandler_ResponseAndHeadersFlushed(t *testing.T) {
	rec := serveTimeout(t, time.Second, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "1")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	})
	if rec.Code != http.StatusCreated || rec.Body.String() != "ok" || rec.Header().Get("X-Test") != "1" {
		t.Fatalf("code=%d body=%q header=%q", rec.Code, rec.Body.String(), rec.Header().Get("X-Test"))
	}
}

func TestTimeout_SlowHandler_503EvenAfterPartialWrite(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	rec := serveTimeout(t, 20*time.Millisecond, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Partial", "1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("partial"))
		<-r.Context().Done()
		<-release
	})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
	if rec.Header().Get("X-Partial") != "" {
		t.Fatal("headers of a timed-out handler must not leak into the 503")
	}
}

func TestTimeout_LateWritesDoNotRace(t *testing.T) {
	// Run with -race: the handler keeps writing headers/body after the
	// deadline while the middleware writes the 503.
	finished := make(chan struct{})
	rec := serveTimeout(t, 10*time.Millisecond, func(w http.ResponseWriter, r *http.Request) {
		defer close(finished)
		<-r.Context().Done()
		// Give the middleware time to take the timeout branch; the handler
		// seeing ctx.Done does not order it after tw.timeout().
		time.Sleep(20 * time.Millisecond)
		for i := 0; i < 100; i++ {
			w.Header().Set("X-Late", "1")
			if _, err := w.Write([]byte("x")); err != http.ErrHandlerTimeout {
				t.Errorf("late write: want ErrHandlerTimeout, got %v", err)
				return
			}
		}
	})
	<-finished
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503, got %d", rec.Code)
	}
}

func TestTimeout_PanicReRaisedOnRequestGoroutine(t *testing.T) {
	defer func() {
		if p := recover(); p != "boom" {
			t.Fatalf("want re-raised panic \"boom\", got %v", p)
		}
	}()
	serveTimeout(t, time.Second, func(http.ResponseWriter, *http.Request) { panic("boom") })
	t.Fatal("expected a panic")
}
