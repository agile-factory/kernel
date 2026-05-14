package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	kernelauth "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/auth"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
)

// jwtPayload mirrors the claims written by platform/auth/infrastructure/jwt-service.go.
// It is intentionally re-declared here to avoid importing platform code into the kernel
// delivery layer (dependency rule: kernel must not depend on platform).
type jwtPayload struct {
	Sub  string `json:"sub"`  // user ID
	Role string `json:"role"` // platform role
	Exp  int64  `json:"exp"`  // expiration unix timestamp
	Iat  int64  `json:"iat"`  // issued-at unix timestamp (kept in sync with jwt-service)
}

// Auth returns a middleware that validates the JWT Bearer token present in the
// Authorization header and injects the authenticated user ID and platform role
// into the request context.
//
// Downstream handlers MUST read identity from the context (kernelauth.UserIDFromContext,
// kernelauth.PlatformRoleFromContext) — never from request headers — because headers
// are client-controllable and could be spoofed by a misconfigured proxy.
//
// On failure: responds with 401 Unauthorized and does not call next.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	secretKey := []byte(jwtSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, errors.Unauthorized("missing or malformed Authorization header"))
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			payload, err := validateJWT(token, secretKey)
			if err != nil {
				response.Error(w, errors.Unauthorized("invalid or expired token"))
				return
			}

			ctx := kernelauth.WithUserID(r.Context(), payload.Sub)
			ctx = kernelauth.WithPlatformRole(ctx, payload.Role)
			r = r.WithContext(ctx)

			next.ServeHTTP(w, r)
		})
	}
}

// jwtHeader holds the bits of the JWT header we care about (only the algorithm).
type jwtHeader struct {
	Alg string `json:"alg"`
}

// validateJWT parses and verifies an HS256 JWT, returning the payload.
func validateJWT(token string, secretKey []byte) (*jwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.Unauthorized("invalid token format")
	}

	// Pin the algorithm: refuse any token whose header advertises something
	// other than HS256. This blocks the classic "alg=none" downgrade and
	// asymmetric-key confusion attacks even though the HMAC compare below
	// would also catch them in practice.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.Unauthorized("invalid token header encoding")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errors.Unauthorized("invalid token header format")
	}
	if header.Alg != "HS256" {
		return nil, errors.Unauthorized("unsupported token algorithm")
	}

	// Verify signature
	unsignedToken := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(unsignedToken))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, errors.Unauthorized("invalid token signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Unauthorized("invalid token payload encoding")
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.Unauthorized("invalid token payload format")
	}

	if time.Now().Unix() > payload.Exp {
		return nil, errors.Unauthorized("token expired")
	}

	return &payload, nil
}
