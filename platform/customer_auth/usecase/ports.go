package usecase

// Hasher hashes plaintext passwords for storage.
type Hasher interface {
	Hash(password string) (string, error)
}

// PasswordVerifier checks a plaintext against a stored hash.
type PasswordVerifier interface {
	Verify(password, hash string) error
}

// PasswordHasher combines Hasher and PasswordVerifier.
// Implemented by infrastructure.BcryptHasher.
type PasswordHasher interface {
	Hasher
	PasswordVerifier
}

// CustomerJWTIssuer issues customer JWT access tokens.
type CustomerJWTIssuer interface {
	GenerateToken(endCustomerID, tenantID string) (string, error)
}

// CustomerJWTValidator validates a customer Bearer token and extracts claims.
// Implemented by infrastructure.CustomerJWTService.
type CustomerJWTValidator interface {
	ValidateToken(token string) (endCustomerID, tenantID string, err error)
}

// CustomerJWTService combines issuance and validation.
// Implemented by infrastructure.CustomerJWTService.
type CustomerJWTService interface {
	CustomerJWTIssuer
	CustomerJWTValidator
}
