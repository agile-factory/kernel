package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoad_DevelopmentGeneratesSecret(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()
	if !cfg.JWTSecretGenerated || len(cfg.JWTSecret) < MinJWTSecretLength {
		t.Fatalf("expected a generated secret, got generated=%v len=%d", cfg.JWTSecretGenerated, len(cfg.JWTSecret))
	}
	if cfg.JWTSecret == legacyDefaultJWTSecret {
		t.Fatal("must not fall back to the legacy default")
	}
	if cfg.DatabaseURL == "" {
		t.Fatal("development keeps a default DATABASE_URL")
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("development config should validate: %v", err)
	}
}

func TestLoad_ProductionHasNoDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()
	if cfg.JWTSecret != "" || cfg.DatabaseURL != "" {
		t.Fatalf("production must not get defaults: secret=%q db=%q", cfg.JWTSecret, cfg.DatabaseURL)
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected Validate to fail without JWT_SECRET")
	}
}

func TestValidate(t *testing.T) {
	strong := strings.Repeat("s", MinJWTSecretLength)
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"prod ok", Config{Environment: "production", JWTSecret: strong, DatabaseURL: "postgres://x", TokenLifespan: time.Hour}, false},
		{"prod short secret", Config{Environment: "production", JWTSecret: "short", DatabaseURL: "postgres://x", TokenLifespan: time.Hour}, true},
		{"prod legacy secret", Config{Environment: "staging", JWTSecret: legacyDefaultJWTSecret, DatabaseURL: "postgres://x", TokenLifespan: time.Hour}, true},
		{"prod no db", Config{Environment: "production", JWTSecret: strong, TokenLifespan: time.Hour}, true},
		{"zero lifespan", Config{Environment: "development", JWTSecret: strong}, true},
		{"dev short secret tolerated", Config{Environment: "development", JWTSecret: "short", TokenLifespan: time.Hour}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.Validate(); (err != nil) != tc.wantErr {
				t.Fatalf("wantErr=%v, got %v", tc.wantErr, err)
			}
		})
	}
}
