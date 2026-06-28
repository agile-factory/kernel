// Package infrastructure provides the external-dependency implementations
// for the customer_auth platform (JWT, bcrypt).
package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// customerJWTPayload holds the claims for a customer access token.
// Distinct from the staff jwtPayload: adds tenant_id + aud:"customer".
type customerJWTPayload struct {
	Sub      string `json:"sub"`       // end_customer_id
	TenantID string `json:"tenant_id"` // tenant scope
	Aud      string `json:"aud"`       // always "customer"
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// CustomerJWTService emits and validates customer JWT access tokens.
// Uses HS256 (same algorithm as staff, different audience claim).
// Staff tokens have aud absent or "staff" — customer tokens have aud "customer".
// A customer token presented to a staff middleware is rejected, and vice-versa.
type CustomerJWTService struct {
	secretKey     []byte
	tokenLifespan time.Duration
}

// NewCustomerJWTService creates a CustomerJWTService.
func NewCustomerJWTService(secretKey string, tokenLifespan time.Duration) *CustomerJWTService {
	return &CustomerJWTService{
		secretKey:     []byte(secretKey),
		tokenLifespan: tokenLifespan,
	}
}

// GenerateToken emits a signed JWT for the given end-customer.
func (s *CustomerJWTService) GenerateToken(endCustomerID, tenantID string) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
	headerBytes, _ := json.Marshal(header)
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)

	now := time.Now()
	payload := customerJWTPayload{
		Sub:      endCustomerID,
		TenantID: tenantID,
		Aud:      "customer",
		Exp:      now.Add(s.tokenLifespan).Unix(),
		Iat:      now.Unix(),
	}
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	unsigned := encodedHeader + "." + encodedPayload
	return unsigned + "." + s.sign(unsigned), nil
}

// ValidateToken validates the token signature, expiry, and audience.
// Returns endCustomerID and tenantID on success.
func (s *CustomerJWTService) ValidateToken(token string) (endCustomerID, tenantID string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", errors.New("invalid token format")
	}

	// Pin algorithm to HS256 — rejects alg:none and key-confusion attacks.
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", errors.New("invalid token header encoding")
	}
	var h jwtHeader
	if err := json.Unmarshal(headerBytes, &h); err != nil || h.Alg != "HS256" {
		return "", "", errors.New("unsupported token algorithm")
	}

	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(s.sign(unsigned))) {
		return "", "", errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", errors.New("invalid token payload encoding")
	}
	var payload customerJWTPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", "", errors.New("invalid token payload")
	}

	if payload.Aud != "customer" {
		return "", "", errors.New("invalid token audience")
	}
	if time.Now().Unix() > payload.Exp {
		return "", "", errors.New("token expired")
	}

	return payload.Sub, payload.TenantID, nil
}

func (s *CustomerJWTService) sign(data string) string {
	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
