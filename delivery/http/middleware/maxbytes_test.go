package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agile-factory/kernel/delivery/http/middleware"
)

func TestMaxBytes(t *testing.T) {
	var readErr error
	h := middleware.MaxBytes(8)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, readErr = io.ReadAll(r.Body)
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader("small")))
	if readErr != nil {
		t.Fatalf("small body: %v", readErr)
	}

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this body is too large")))
	if readErr == nil {
		t.Fatal("oversized body should fail to read")
	}
}
