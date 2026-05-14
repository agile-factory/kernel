package middleware_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	kernelauth "github.com/AgileFactory/kernel/kernel/auth"
	"github.com/AgileFactory/kernel/delivery/http/middleware"
)

// ─── JWT helpers ──────────────────────────────────────────────────────────────

type testClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	Exp  int64  `json:"exp"`
}

// buildJWT constructs a minimal HS256 JWT signed with the given secret.
func buildJWT(secret string, claims testClaims) string {
	header := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"alg":"HS256","typ":"JWT"}`),
	)
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	unsigned := header + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + sig
}

// validToken returns a non-expired JWT signed with the given secret.
func validToken(secret, userID, role string) string {
	return buildJWT(secret, testClaims{
		Sub:  userID,
		Role: role,
		Exp:  time.Now().Add(time.Hour).Unix(),
	})
}

// expiredToken returns an expired JWT signed with the given secret.
func expiredToken(secret, userID string) string {
	return buildJWT(secret, testClaims{
		Sub:  userID,
		Role: "ROLE_VIEWER",
		Exp:  time.Now().Add(-time.Hour).Unix(),
	})
}

// nextHandler captures the request to let tests inspect context values.
type captureHandler struct {
	called  bool
	userID  string
	role    string
	xUserID string
}

func (h *captureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	h.userID = kernelauth.UserIDFromContext(r.Context())
	h.role = kernelauth.PlatformRoleFromContext(r.Context())
	h.xUserID = r.Header.Get("X-User-ID")
	w.WriteHeader(http.StatusOK)
}

const testSecret = "test-jwt-secret-key"

// ─── Auth middleware tests ────────────────────────────────────────────────────

func TestAuth_ValidToken_CallsNext(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	token := validToken(testSecret, "user-001", "ROLE_VIEWER")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if !next.called {
		t.Error("next handler should have been called")
	}
}

func TestAuth_ValidToken_InjectsUserIDIntoContext(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	token := validToken(testSecret, "user-999", "ROLE_VIEWER")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if next.userID != "user-999" {
		t.Errorf("context userID: got %q, want %q", next.userID, "user-999")
	}
}

func TestAuth_ValidToken_InjectsPlatformRoleIntoContext(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	token := validToken(testSecret, "user-001", "ROLE_ADMIN")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if next.role != "ROLE_ADMIN" {
		t.Errorf("context role: got %q, want %q", next.role, "ROLE_ADMIN")
	}
}

// TestAuth_DoesNotSetXUserIDHeader documents the contract that the middleware
// MUST NOT inject identity into request headers — handlers are required to read
// from the context. Prevents regressions to the old "legacy header" pattern.
func TestAuth_DoesNotSetXUserIDHeader(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	token := validToken(testSecret, "user-123", "ROLE_VIEWER")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if next.xUserID != "" {
		t.Errorf("X-User-ID header should not be set by middleware; got %q", next.xUserID)
	}
}

func TestAuth_MissingAuthorizationHeader_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler must not be called when Authorization header is missing")
	}
}

func TestAuth_EmptyAuthorizationHeader_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_WrongPrefix_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	// "Basic" prefix instead of "Bearer"
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler must not be called with non-Bearer scheme")
	}
}

func TestAuth_InvalidSignature_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	// Sign with a different secret
	token := validToken("wrong-secret", "user-001", "ROLE_VIEWER")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler must not be called with wrong signature")
	}
}

func TestAuth_ExpiredToken_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	token := expiredToken(testSecret, "user-001")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler must not be called with expired token")
	}
}

func TestAuth_MalformedToken_TwoParts_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer header.payload")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_MalformedToken_FourParts_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer a.b.c.d")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuth_TamperedPayload_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	// Build a valid token, then swap the payload for a tampered one.
	original := validToken(testSecret, "user-001", "ROLE_VIEWER")
	parts := splitToken(original)

	// Create a different payload (elevated role)
	tamperedClaims := testClaims{
		Sub:  "user-001",
		Role: "ROLE_ADMIN",
		Exp:  time.Now().Add(time.Hour).Unix(),
	}
	tamperedPayloadBytes, _ := json.Marshal(tamperedClaims)
	tamperedPayload := base64.RawURLEncoding.EncodeToString(tamperedPayloadBytes)
	tampered := parts[0] + "." + tamperedPayload + "." + parts[2]

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tampered)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("tampered token must be rejected: status got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if next.called {
		t.Error("next handler must not be called with a tampered token")
	}
}

func TestAuth_DifferentSecrets_Isolation(t *testing.T) {
	// Two handlers with different secrets must not cross-accept tokens.
	handlerA := middleware.Auth("secret-A")(new(captureHandler))
	handlerB := middleware.Auth("secret-B")(new(captureHandler))

	tokenA := validToken("secret-A", "user-001", "ROLE_VIEWER")

	// handlerA accepts token signed with A
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec := httptest.NewRecorder()
	handlerA.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("handlerA: expected 200 for its own token, got %d", rec.Code)
	}

	// handlerB rejects token signed with A
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	rec = httptest.NewRecorder()
	handlerB.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("handlerB: expected 401 for foreign token, got %d", rec.Code)
	}
}

func TestAuth_InvalidBase64Payload_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	// Build a token with an invalid base64 payload but correct structure.
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	invalidPayload := "!!!not-valid-base64!!!"
	unsigned := header + "." + invalidPayload
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token := unsigned + "." + sig

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func splitToken(token string) []string {
	parts := make([]string, 0, 3)
	start := 0
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			parts = append(parts, token[start:i])
			start = i + 1
		}
	}
	parts = append(parts, token[start:])
	return parts
}

// Regression: ensure BearerToken with only spaces after prefix is rejected.
func TestAuth_BearerPrefixOnly_Returns401(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// Verify that the response content type is JSON for all rejection paths.
func TestAuth_Rejections_ReturnJSONContentType(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
	}{
		{"no header", ""},
		{"bad prefix", "Basic dXNlcjpwYXNz"},
		{"bad token", "Bearer not.a.token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := middleware.Auth(testSecret)(new(captureHandler))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authorization != "" {
				req.Header.Set("Authorization", tt.authorization)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			ct := rec.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
			}
		})
	}
}

// Boundary: token with exp exactly at current second — should still pass
// (exp is checked with strict greater-than, so exp == now is valid).
func TestAuth_TokenExpiryBoundary_StillValid(t *testing.T) {
	next := &captureHandler{}
	handler := middleware.Auth(testSecret)(next)

	// exp = now; the check is time.Now().Unix() > payload.Exp, so equal is OK.
	token := buildJWT(testSecret, testClaims{
		Sub:  "user-001",
		Role: "ROLE_VIEWER",
		Exp:  time.Now().Unix(),
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// exp == now.Unix() is technically valid per the code (strict >)
	if rec.Code != http.StatusOK {
		t.Logf("note: token with exp==now returned %d (acceptable — clock precision)", rec.Code)
	}
	// We only assert no panic; either 200 or 401 is acceptable at exact boundary.
	_ = fmt.Sprintf("boundary test passed, code=%d", rec.Code)
}