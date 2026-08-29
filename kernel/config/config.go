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
	"os"
	"time"

	"github.com/joho/godotenv"
)

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
}

// Load retrieves the configuration from environment variables
// with fallback to default values.
func Load() *Config {
	_ = godotenv.Load("./.env")          // Load .env from current directory (backend)
	_ = godotenv.Load("../.env") // Load .env from root directory as fallback
	return &Config{
		Environment:     getEnv("APP_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogDir:          getEnv("LOG_DIR", "logs"),
		Port:            getEnv("PORT", "8080"),
		ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second),
		JWTSecret:       getEnv("JWT_SECRET", "super-secret-key-change-in-production"),
		TokenLifespan:   getEnvAsDuration("TOKEN_LIFESPAN", 24*time.Hour),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://lokatik:lokatik_dev_password@postgres:5432/lokatik"),
	}
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
