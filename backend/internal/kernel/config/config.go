package config

import (
	"os"
	"time"
)

// Config holds the configuration for the application
type Config struct {
	Environment     string
	LogLevel        string
	Port            string
	ShutdownTimeout time.Duration
	JWTSecret       string
	TokenLifespan   time.Duration
}

// Load retrieves the configuration from environment variables
// with fallback to default values.
func Load() *Config {
	return &Config{
		Environment:     getEnv("APP_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Port:            getEnv("PORT", "8080"),
		ShutdownTimeout: getEnvAsDuration("SHUTDOWN_TIMEOUT", 5*time.Second),
		JWTSecret:       getEnv("JWT_SECRET", "super-secret-key-change-in-production"),
		TokenLifespan:   getEnvAsDuration("TOKEN_LIFESPAN", 24*time.Hour),
	}
}

// getEnv reads an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
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
