package middleware

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
)

// jwtPayload mirrors the claims written by platform/auth/infrastructure/jwt-service.go.
// It is intentionally re-declared here to avoid importing platform code into the kernel
// delivery layer (dependency rule: kernel must not depend on platform).
type jwtPayload struct {
	Sub  string `json:"sub"`           // user ID
	Role string `json:"role"`          // platform role
	Exp  int64  `json:"exp"`           // expiration unix timestamp
	Iat  int64  `json:"iat"`           // issued-at unix timestamp (kept in sync with jwt-service)
	Nbf  int64  `json:"nbf,omitempty"` // not-before unix timestamp (optional)
}

// TokenValidator verifies a raw JWT and returns its subject and platform role.
// NewHS256Validator and NewRS256Validator cover the two signing modes of
// platform/auth; consumers may plug their own (e.g. an external IdP).
type TokenValidator interface {
	ValidateToken(token string) (userID, role string, err error)
}

// Auth returns a middleware that validates the HS256 JWT Bearer token present in
// the Authorization header and injects the authenticated user ID and platform role
// into the request context. It panics if jwtSecret is empty: an empty HMAC key
// would let anyone forge tokens.
//
// Downstream handlers MUST read identity from the context (kernelauth.UserIDFromContext,
// kernelauth.PlatformRoleFromContext) — never from request headers — because headers
// are client-controllable and could be spoofed by a misconfigured proxy.
//
// On failure: responds with 401 Unauthorized and does not call next.
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return AuthWith(NewHS256Validator(jwtSecret))
}

// AuthWith is Auth with an explicit validator (use it for RS256).
func AuthWith(v TokenValidator) func(http.Handler) http.Handler {
	return authMiddleware(v, false)
}

// OptionalAuth validates the Bearer token when one is present and injects the
// identity; requests without an Authorization header pass through
// unauthenticated (handlers then see an empty user ID and answer 401
// themselves). A header that is present but invalid is still rejected with 401.
// It lets a single mux serve public and protected routes.
func OptionalAuth(v TokenValidator) func(http.Handler) http.Handler {
	return authMiddleware(v, true)
}

func authMiddleware(v TokenValidator, optional bool) func(http.Handler) http.Handler {
	if v == nil {
		panic("middleware: nil TokenValidator")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" && optional {
				next.ServeHTTP(w, r)
				return
			}
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, errors.Unauthorized("missing or malformed Authorization header"))
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			userID, role, err := v.ValidateToken(token)
			if err != nil {
				response.Error(w, errors.Unauthorized("invalid or expired token"))
				return
			}

			ctx := kernelauth.WithUserID(r.Context(), userID)
			ctx = kernelauth.WithPlatformRole(ctx, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// jwtHeader holds the bits of the JWT header we care about.
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid,omitempty"`
}

type hs256Validator struct{ key []byte }

// NewHS256Validator returns a validator for HS256 tokens. It panics on an
// empty secret.
func NewHS256Validator(secret string) TokenValidator {
	if secret == "" {
		panic("middleware: empty JWT secret")
	}
	return &hs256Validator{key: []byte(secret)}
}

func (v *hs256Validator) ValidateToken(token string) (string, string, error) {
	p, err := validateJWT(token, "HS256", "", func(unsigned string, sig []byte) bool {
		mac := hmac.New(sha256.New, v.key)
		mac.Write([]byte(unsigned))
		return hmac.Equal(sig, mac.Sum(nil))
	})
	if err != nil {
		return "", "", err
	}
	return p.Sub, p.Role, nil
}

type rs256Validator struct {
	pub *rsa.PublicKey
	kid string
}

// NewRS256Validator returns a validator for RS256 tokens signed by the
// private half of pub. When kid is non-empty, tokens must carry that key ID.
func NewRS256Validator(pub *rsa.PublicKey, kid string) TokenValidator {
	if pub == nil {
		panic("middleware: nil RSA public key")
	}
	return &rs256Validator{pub: pub, kid: kid}
}

func (v *rs256Validator) ValidateToken(token string) (string, string, error) {
	p, err := validateJWT(token, "RS256", v.kid, func(unsigned string, sig []byte) bool {
		digest := sha256.Sum256([]byte(unsigned))
		return rsa.VerifyPKCS1v15(v.pub, crypto.SHA256, digest[:], sig) == nil
	})
	if err != nil {
		return "", "", err
	}
	return p.Sub, p.Role, nil
}

// validateJWT parses token, pins its algorithm to alg (and its kid when
// wantKid is set), checks the signature with verify, then the time claims.
func validateJWT(token, alg, wantKid string, verify func(unsigned string, sig []byte) bool) (*jwtPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.Unauthorized("invalid token format")
	}

	// Pin the algorithm: refuse any token whose header advertises something
	// else. This blocks the classic "alg=none" downgrade and HS/RS key
	// confusion attacks.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.Unauthorized("invalid token header encoding")
	}
	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, errors.Unauthorized("invalid token header format")
	}
	if header.Alg != alg {
		return nil, errors.Unauthorized("unsupported token algorithm")
	}
	if wantKid != "" && header.Kid != wantKid {
		return nil, errors.Unauthorized("unknown token key id")
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !verify(parts[0]+"."+parts[1], sig) {
		return nil, errors.Unauthorized("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Unauthorized("invalid token payload encoding")
	}
	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, errors.Unauthorized("invalid token payload format")
	}

	now := time.Now().Unix()
	if now > payload.Exp {
		return nil, errors.Unauthorized("token expired")
	}
	if payload.Nbf != 0 && now < payload.Nbf {
		return nil, errors.Unauthorized("token not yet valid")
	}
	if payload.Sub == "" {
		return nil, errors.Unauthorized("token has no subject")
	}

	return &payload, nil
}
