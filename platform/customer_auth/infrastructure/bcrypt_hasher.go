// Package infrastructure provides the external-dependency implementations
// for the customer_auth platform.
package infrastructure

import "golang.org/x/crypto/bcrypt"

const bcryptCost = bcrypt.DefaultCost // 10 rounds

// BcryptHasher hashes and verifies passwords using bcrypt (cost=10).
// Input is capped at 72 bytes by bcrypt — callers must validate length first.
type BcryptHasher struct{}

// NewBcryptHasher creates a BcryptHasher.
func NewBcryptHasher() *BcryptHasher { return &BcryptHasher{} }

// Hash returns the bcrypt hash of the given password.
func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify returns nil when password matches the stored hash.
func (h *BcryptHasher) Verify(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
