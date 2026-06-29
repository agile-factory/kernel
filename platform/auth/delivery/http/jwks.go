package http

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
)

// jwksKey is a single JSON Web Key (RFC 7517).
type jwksKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"` // base64url(modulus)
	E   string `json:"e"` // base64url(public exponent)
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// JWKSHandler returns an HTTP handler that serves the RSA public key in JWKS
// format at /.well-known/jwks.json. Third-party services (e.g. API gateways,
// frontend token validation) can fetch this endpoint to verify RS256 tokens
// without sharing a symmetric secret.
//
//	GET /.well-known/jwks.json
//	→ {"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"...","n":"...","e":"..."}]}
func JWKSHandler(pub *rsa.PublicKey, kid string) http.HandlerFunc {
	// Pre-compute the JWKS response at construction time — the key is static.
	key := jwksKey{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
	body, _ := json.Marshal(jwksResponse{Keys: []jwksKey{key}})

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// JWKS is public and cacheable. 1 hour max-age is a reasonable default;
		// operators should rotate keys with sufficient lead time before expiry.
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}
