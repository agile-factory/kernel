package usecase

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// generateOpaqueToken returns a cryptographically random 32-byte URL-safe base64 string.
// It is the raw bearer value sent to the client; the hash is stored in the DB.
func generateOpaqueToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("customer_auth: crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// hashToken returns the hex-encoded SHA-256 digest of token (64 chars).
// This is what gets stored in the token_hash column (VARCHAR(64)).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
