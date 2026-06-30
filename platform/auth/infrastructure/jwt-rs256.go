package infrastructure

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/agile-factory/kernel/platform/auth/domain"
)

// TokenSigner is the common interface implemented by both JWTService (HS256)
// and RS256JWTService. Use this type for injection so callers are not tied
// to a specific algorithm.
type TokenSigner interface {
	GenerateToken(user *domain.User) (string, error)
	ValidateToken(token string) (string, error)
}

// RS256JWTService signs and verifies JWT tokens using RSA-SHA256 (RS256).
// Use it when you need JWKS-compatible tokens that third-party services can
// verify using the public key from /.well-known/jwks.json.
type RS256JWTService struct {
	privateKey    *rsa.PrivateKey
	kid           string // Key ID included in JWT header and JWKS
	tokenLifespan time.Duration
}

// NewRS256JWTService creates an RS256JWTService. kid identifies the signing
// key in the JWKS response; rotate it when you rotate the key pair.
func NewRS256JWTService(key *rsa.PrivateKey, kid string, tokenLifespan time.Duration) *RS256JWTService {
	return &RS256JWTService{
		privateKey:    key,
		kid:           kid,
		tokenLifespan: tokenLifespan,
	}
}

// PublicKey returns the RSA public key for JWKS endpoint construction.
func (s *RS256JWTService) PublicKey() *rsa.PublicKey {
	return &s.privateKey.PublicKey
}

// KID returns the key identifier used in JWT headers and JWKS.
func (s *RS256JWTService) KID() string { return s.kid }

// GenerateToken issues a signed RS256 JWT for the given user.
func (s *RS256JWTService) GenerateToken(user *domain.User) (string, error) {
	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": s.kid}
	headerBytes, _ := json.Marshal(header)
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)

	now := time.Now()
	payload := jwtPayload{
		Sub:  user.ID,
		Role: string(user.Role),
		Exp:  now.Add(s.tokenLifespan).Unix(),
		Iat:  now.Unix(),
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	unsignedToken := encodedHeader + "." + encodedPayload
	sig, err := s.signRSA([]byte(unsignedToken))
	if err != nil {
		return "", err
	}
	return unsignedToken + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// ValidateToken verifies the RS256 signature and returns the subject (user ID).
func (s *RS256JWTService) ValidateToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("invalid token header encoding")
	}
	var header map[string]string
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", errors.New("invalid token header")
	}
	if header["alg"] != "RS256" {
		return "", errors.New("unsupported token algorithm")
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", errors.New("invalid token signature encoding")
	}
	unsigned := parts[0] + "." + parts[1]
	if err := s.verifyRSA([]byte(unsigned), sigBytes); err != nil {
		return "", errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid token payload encoding")
	}
	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", errors.New("invalid token payload")
	}
	if time.Now().Unix() > payload.Exp {
		return "", errors.New("token expired")
	}
	return payload.Sub, nil
}

func (s *RS256JWTService) signRSA(data []byte) ([]byte, error) {
	h := sha256.New()
	h.Write(data)
	digest := h.Sum(nil)
	return rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, digest)
}

func (s *RS256JWTService) verifyRSA(data, sig []byte) error {
	h := sha256.New()
	h.Write(data)
	digest := h.Sum(nil)
	return rsa.VerifyPKCS1v15(&s.privateKey.PublicKey, crypto.SHA256, digest, sig)
}
