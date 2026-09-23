package middleware_test

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agile-factory/kernel/delivery/http/middleware"
)

func buildRS256(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	hdr, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	pl, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(hdr) + "." + base64.RawURLEncoding.EncodeToString(pl)
	digest := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// buildJWTRaw signs arbitrary claims with HS256.
func buildJWTRaw(secret string, claims map[string]any) string {
	pl, _ := json.Marshal(claims)
	hdr := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	unsigned := hdr + "." + base64.RawURLEncoding.EncodeToString(pl)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func serve(h http.Handler, authz string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAuth_EmptySecret_Panics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for an empty secret")
		}
	}()
	middleware.Auth("")
}

func TestOptionalAuth_NoHeader_PassesThroughAnonymous(t *testing.T) {
	next := &captureHandler{}
	rec := serve(middleware.OptionalAuth(middleware.NewHS256Validator(testSecret))(next), "")
	if rec.Code != http.StatusOK || !next.called || next.userID != "" {
		t.Fatalf("code=%d called=%v user=%q", rec.Code, next.called, next.userID)
	}
}

func TestOptionalAuth_InvalidToken_ContinuesAnonymous(t *testing.T) {
	for name, authz := range map[string]string{
		"bad signature": "Bearer " + validToken("other-secret", "u", "r"),
		"expired":       "Bearer " + expiredToken(testSecret, "u"),
		"not bearer":    "Basic dXNlcjpwYXNz",
	} {
		t.Run(name, func(t *testing.T) {
			next := &captureHandler{}
			rec := serve(middleware.OptionalAuth(middleware.NewHS256Validator(testSecret))(next), authz)
			if rec.Code != http.StatusOK || !next.called || next.userID != "" || next.role != "" {
				t.Fatalf("code=%d called=%v user=%q role=%q", rec.Code, next.called, next.userID, next.role)
			}
		})
	}
}

func TestOptionalAuth_ValidToken_InjectsIdentity(t *testing.T) {
	next := &captureHandler{}
	serve(middleware.OptionalAuth(middleware.NewHS256Validator(testSecret))(next), "Bearer "+validToken(testSecret, "user-9", "ROLE_ADMIN"))
	if next.userID != "user-9" || next.role != "ROLE_ADMIN" {
		t.Fatalf("user=%q role=%q", next.userID, next.role)
	}
}

func TestAuth_NotBeforeInFuture_Returns401(t *testing.T) {
	tok := buildJWTRaw(testSecret, map[string]any{
		"sub": "u", "role": "r",
		"exp": time.Now().Add(time.Hour).Unix(),
		"nbf": time.Now().Add(30 * time.Minute).Unix(),
	})
	next := &captureHandler{}
	if rec := serve(middleware.Auth(testSecret)(next), "Bearer "+tok); rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
}

func TestRS256Validator(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := map[string]any{"sub": "user-1", "role": "ROLE_VIEWER", "exp": time.Now().Add(time.Hour).Unix()}
	mw := middleware.AuthWith(middleware.NewRS256Validator(&key.PublicKey, "k1"))

	cases := []struct {
		name  string
		token string
		want  int
	}{
		{"valid", buildRS256(t, key, "k1", claims), http.StatusOK},
		{"wrong kid", buildRS256(t, key, "k2", claims), http.StatusUnauthorized},
		{"wrong key", buildRS256(t, other, "k1", claims), http.StatusUnauthorized},
		{"hs256 token", validToken(testSecret, "user-1", "ROLE_VIEWER"), http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			next := &captureHandler{}
			if rec := serve(mw(next), "Bearer "+tc.token); rec.Code != tc.want {
				t.Fatalf("want %d, got %d", tc.want, rec.Code)
			}
		})
	}
}
