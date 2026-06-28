package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agile-factory/kernel/kernel/health"
)

func TestLivenessHandler_AlwaysOK(t *testing.T) {
	c := health.New(health.Config{Version: "0.1.0"})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	c.LivenessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
	if resp["version"] != "0.1.0" {
		t.Errorf("expected version=0.1.0, got %v", resp["version"])
	}
}

func TestReadinessHandler_NotReady(t *testing.T) {
	c := health.New(health.Config{Version: "0.1.0"})
	// MarkReady not called

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	c.ReadinessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestReadinessHandler_Ready_NoDBPinger(t *testing.T) {
	c := health.New(health.Config{Version: "0.1.0"})
	c.MarkReady()

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	c.ReadinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
}

// stubPinger simulates a DB ping result.
type stubPinger struct{ err error }

func (s *stubPinger) Ping(_ context.Context) error { return s.err }

func TestReadinessHandler_DBDown(t *testing.T) {
	c := health.New(health.Config{
		Version:  "0.1.0",
		DBPinger: &stubPinger{err: context.DeadlineExceeded},
	})
	c.MarkReady()

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	c.ReadinessHandler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when DB down, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "error") {
		t.Errorf("expected error in response body, got %s", body)
	}
}

func TestReadinessHandler_DBUp(t *testing.T) {
	c := health.New(health.Config{
		Version:  "0.1.0",
		DBPinger: &stubPinger{err: nil},
	})
	c.MarkReady()

	req := httptest.NewRequest("GET", "/ready", nil)
	w := httptest.NewRecorder()
	c.ReadinessHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when DB up, got %d", w.Code)
	}
	var resp struct {
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Checks["database"] != "ok" {
		t.Errorf("expected database=ok, got %s", resp.Checks["database"])
	}
}
