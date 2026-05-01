package infrastructure

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
)

// jwtHeader represents the JWT header
type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// jwtPayload represents our specific JWT claims.
type jwtPayload struct {
	Sub  string `json:"sub"`  // User ID
	Role string `json:"role"` // User's platform role
	Exp  int64  `json:"exp"`  // Expiration time
	Iat  int64  `json:"iat"`  // Issued at
}

// JWTService implements TokenGenerator and TokenValidator using stdlib.
type JWTService struct {
	secretKey     []byte
	tokenLifespan time.Duration
}

// NewJWTService creates a new JWTService.
func NewJWTService(secretKey string, tokenLifespan time.Duration) *JWTService {
	return &JWTService{
		secretKey:     []byte(secretKey),
		tokenLifespan: tokenLifespan,
	}
}

// GenerateToken generates a JWT for the given user.
func (s *JWTService) GenerateToken(user *domain.User) (string, error) {
	header := jwtHeader{Alg: "HS256", Typ: "JWT"}
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
	signature := s.sign(unsignedToken)

	return unsignedToken + "." + signature, nil
}


// ValidateToken validates the JWT and returns the User ID (Sub).
func (s *JWTService) ValidateToken(token string) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid token format")
	}

	unsignedToken := parts[0] + "." + parts[1]
	signature := parts[2]

	expectedSignature := s.sign(unsignedToken)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", errors.New("invalid token signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("invalid token payload encoding")
	}

	var payload jwtPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", errors.New("invalid token payload format")
	}

	if time.Now().Unix() > payload.Exp {
		return "", errors.New("token expired")
	}

	return payload.Sub, nil
}

// sign creates an HMAC SHA256 signature for the given data.
func (s *JWTService) sign(data string) string {
	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
