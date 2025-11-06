package config

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Environment string
	Port        string
	AppURL      string

	// Database
	DatabaseURL string

	// Redis
	RedisURL string

	// JWT
	JWTPrivateKey *rsa.PrivateKey
	JWTPublicKey  *rsa.PublicKey
	JWTExpiry     int // in seconds

	// CORS
	CORSOrigins string

	// AWS S3
	S3Bucket        string
	S3Region        string
	AWSAccessKey    string
	AWSSecretKey    string
	S3Endpoint      string // for MinIO compatibility
	S3UsePathStyle  bool
	S3PresignExpiry int // in seconds

	// NATS
	NATSUrl string

	// Email (AWS SES)
	SESRegion     string
	SESFromEmail  string
	SESFromName   string
	EmailProvider string // "ses" or "smtp"

	// SMS (Twilio)
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioPhoneFrom  string

	// Payment providers
	WaveAPIKey         string
	WaveWebhookSecret  string
	OrangeAPIKey       string
	OrangeWebhookSecret string
	FreeAPIKey         string
	FreeWebhookSecret  string

	// DocuSign
	DocuSignIntegrationKey string
	DocuSignUserID         string
	DocuSignAccountID      string
	DocuSignPrivateKey     string

	// Sentry
	SentryDSN string

	// Rate limiting
	RateLimitMax        int
	RateLimitExpiration int // in seconds
}

func LoadConfig() *Config {
	// Load .env file if exists (for local development)
	_ = godotenv.Load()

	cfg := &Config{
		Environment:         getEnv("ENVIRONMENT", "development"),
		Port:                getEnv("PORT", "8080"),
		AppURL:              getEnv("APP_URL", "http://localhost:8080"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/immobilier_saas?sslmode=disable"),
		RedisURL:            getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTExpiry:           getEnvInt("JWT_EXPIRY", 900), // 15 minutes
		CORSOrigins:         getEnv("CORS_ORIGINS", "http://localhost:3000"),
		S3Bucket:            getEnv("S3_BUCKET", "immobilier-saas"),
		S3Region:            getEnv("S3_REGION", "us-east-1"),
		AWSAccessKey:        getEnv("AWS_ACCESS_KEY_ID", ""),
		AWSSecretKey:        getEnv("AWS_SECRET_ACCESS_KEY", ""),
		S3Endpoint:          getEnv("S3_ENDPOINT", ""), // for MinIO: http://localhost:9000
		S3UsePathStyle:      getEnv("S3_USE_PATH_STYLE", "false") == "true",
		S3PresignExpiry:     getEnvInt("S3_PRESIGN_EXPIRY", 3600),
		NATSUrl:             getEnv("NATS_URL", "nats://localhost:4222"),
		SESRegion:           getEnv("SES_REGION", "us-east-1"),
		SESFromEmail:        getEnv("SES_FROM_EMAIL", "noreply@immobilier.example.com"),
		SESFromName:         getEnv("SES_FROM_NAME", "Immobilier SaaS"),
		EmailProvider:       getEnv("EMAIL_PROVIDER", "ses"),
		TwilioAccountSID:    getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:     getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneFrom:     getEnv("TWILIO_PHONE_FROM", ""),
		WaveAPIKey:          getEnv("WAVE_API_KEY", ""),
		WaveWebhookSecret:   getEnv("WAVE_WEBHOOK_SECRET", ""),
		OrangeAPIKey:        getEnv("ORANGE_API_KEY", ""),
		OrangeWebhookSecret: getEnv("ORANGE_WEBHOOK_SECRET", ""),
		FreeAPIKey:          getEnv("FREE_API_KEY", ""),
		FreeWebhookSecret:   getEnv("FREE_WEBHOOK_SECRET", ""),
		DocuSignIntegrationKey: getEnv("DOCUSIGN_INTEGRATION_KEY", ""),
		DocuSignUserID:         getEnv("DOCUSIGN_USER_ID", ""),
		DocuSignAccountID:      getEnv("DOCUSIGN_ACCOUNT_ID", ""),
		DocuSignPrivateKey:     getEnv("DOCUSIGN_PRIVATE_KEY", ""),
		SentryDSN:              getEnv("SENTRY_DSN", ""),
		RateLimitMax:           getEnvInt("RATE_LIMIT_MAX", 100),
		RateLimitExpiration:    getEnvInt("RATE_LIMIT_EXPIRATION", 60),
	}

	// Load JWT keys
	cfg.loadJWTKeys()

	return cfg
}

func (c *Config) loadJWTKeys() {
	privateKeyPEM := getEnv("JWT_PRIVATE_KEY", "")
	publicKeyPEM := getEnv("JWT_PUBLIC_KEY", "")

	if privateKeyPEM == "" || publicKeyPEM == "" {
		log.Fatal("JWT_PRIVATE_KEY and JWT_PUBLIC_KEY must be set")
	}

	// Parse private key
	privateBlock, _ := pem.Decode([]byte(privateKeyPEM))
	if privateBlock == nil {
		log.Fatal("Failed to parse JWT private key PEM")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(privateBlock.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse JWT private key: %v", err)
	}
	c.JWTPrivateKey = privateKey.(*rsa.PrivateKey)

	// Parse public key
	publicBlock, _ := pem.Decode([]byte(publicKeyPEM))
	if publicBlock == nil {
		log.Fatal("Failed to parse JWT public key PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse JWT public key: %v", err)
	}
	c.JWTPublicKey = publicKey.(*rsa.PublicKey)
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var intValue int
	_, err := fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		return defaultValue
	}
	return intValue
}
