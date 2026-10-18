// Package config loads runtime configuration from environment variables
// (with optional .env fallback via godotenv). It is the single ingress
// point for env-driven settings on the kernel side — platform packages
// MUST NOT call os.Getenv directly (enforced by scripts/lint.sh per
// RFC-0038).
//
// Usage:
//
//	cfg := config.Load()
//	log, _, _ := logger.New(logger.Config{Level: cfg.LogLevel, ...})
//
// Recognised variables and defaults are documented on the Config fields.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// MinJWTSecretLength is the minimum HS256 key size accepted outside
// development (RFC 7518 §3.2: key at least as long as the hash output).
const MinJWTSecretLength = 32

// legacyDefaultJWTSecret was the hardcoded fallback of earlier versions. It
// is public knowledge, so it is refused like an empty secret.
const legacyDefaultJWTSecret = "super-secret-key-change-in-production"

// devDatabaseURL is only used when APP_ENV is development or test.
const devDatabaseURL = "postgres://lokatik:lokatik_dev_password@postgres:5432/lokatik"

// Config holds the configuration for the application
type Config struct {
	Environment     string
	LogLevel        string
	LogDir          string
	Port            string
	ShutdownTimeout time.Duration
	JWTSecret       string
	TokenLifespan   time.Duration
	DatabaseURL     string

	// JWTSecretGenerated is true when JWT_SECRET was unset in development
	// and a random per-process secret was generated (tokens do not survive
	// a restart). Callers should log a warning.
	JWTSecretGenerated bool
}

// Load retrieves the configuration from environment variables
// with fallback to default values. There is no default JWT secret nor
// database URL outside development: call Validate before using the Config.
func Load() *Config {
	_ = godotenv.Load()          // Load .env from current directory (backend)
	_ = godotenv.Load("../.env") // Load .env from root directory as fallback
	cfg := &Config{
		Environment:     getEnv("APP_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogDir:          getEnv("LOG_DIR", "logs"),
		Port:            getEnv("PORT", "8080"),
		ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second),
		JWTSecret:       getEnv("JWT_SECRET", ""),
		TokenLifespan:   getEnvAsDuration("TOKEN_LIFESPAN", time.Hour),
		DatabaseURL:     getEnv("DATABASE_URL", ""),
	}
	if cfg.IsDevelopment() {
		if cfg.JWTSecret == "" {
			cfg.JWTSecret = randomSecret()
			cfg.JWTSecretGenerated = true
		}
		if cfg.DatabaseURL == "" {
			cfg.DatabaseURL = devDatabaseURL
		}
	}
	return cfg
}

// IsDevelopment reports whether APP_ENV is development or test — the only
// environments where insecure defaults are tolerated.
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development" || c.Environment == "test"
}

// Validate rejects configurations that are unsafe to run outside
// development: missing, short or publicly known JWT secret, missing
// database URL, non-positive token lifespan.
func (c *Config) Validate() error {
	if c.TokenLifespan <= 0 {
		return fmt.Errorf("config: TOKEN_LIFESPAN must be positive, got %s", c.TokenLifespan)
	}
	if c.IsDevelopment() {
		return nil
	}
	switch {
	case c.JWTSecret == "":
		return fmt.Errorf("config: JWT_SECRET is required when APP_ENV=%s", c.Environment)
	case c.JWTSecret == legacyDefaultJWTSecret:
		return fmt.Errorf("config: JWT_SECRET is the publicly known legacy default; generate one with `openssl rand -base64 32`")
	case len(c.JWTSecret) < MinJWTSecretLength:
		return fmt.Errorf("config: JWT_SECRET must be at least %d bytes, got %d", MinJWTSecretLength, len(c.JWTSecret))
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("config: DATABASE_URL is required when APP_ENV=%s", c.Environment)
	}
	return nil
}

func randomSecret() string {
	b := make([]byte, MinJWTSecretLength)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("config: crypto/rand failed: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return fallback
}

// getEnvAsDuration reads an environment variable and parses it as a time.Duration.
// If it fails or doesn't exist, it returns a default value.
func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return fallback
}
