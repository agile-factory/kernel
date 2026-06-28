// Package customer_auth is the end-customer authentication platform (RFC-0034).
// It manages credentials, JWT access tokens, refresh tokens, and password-reset
// flows for the B2B2C customer-facing portal.
//
// Wiring:
//
//	hasher      := cainfra.NewBcryptHasher()
//	credRepo    := cainfra.NewPostgresCredentialRepository(db)
//	refreshRepo := cainfra.NewPostgresRefreshTokenRepository(db)
//	resetRepo   := cainfra.NewPostgresResetTokenRepository(db)
//	jwtSvc      := cainfra.NewCustomerJWTService(secret, 1*time.Hour)
//	caMod, err  := customer_auth.New(customer_auth.Config{
//	    DB:          db,
//	    CredRepo:    credRepo,
//	    RefreshRepo: refreshRepo,
//	    ResetRepo:   resetRepo,
//	    Hasher:      hasher,
//	    JWTSvc:      jwtSvc,
//	    Publisher:   bus.Publisher(),
//	    Logger:      log,
//	})
//	caMod.RegisterRoutes(mux)
package customer_auth

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	cahttp "github.com/agile-factory/kernel/platform/customer_auth/delivery/http"
	"github.com/agile-factory/kernel/platform/customer_auth/domain"
	"github.com/agile-factory/kernel/platform/customer_auth/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds all SQL migration files for this platform.
//
//go:embed migrations/*.sql
var Migrations embed.FS

const defaultRefreshTTL = 30 * 24 * time.Hour // 30 days

// Config captures the dependencies of the customer_auth Module.
type Config struct {
	DB          *pgxpool.Pool                          // may be nil in unit tests
	CredRepo    domain.CredentialRepository
	RefreshRepo domain.CustomerRefreshTokenRepository
	ResetRepo   domain.CustomerResetTokenRepository
	Hasher      usecase.PasswordHasher
	JWTSvc      usecase.CustomerJWTService
	Publisher   events.Publisher
	Logger      *slog.Logger
	RefreshTTL  time.Duration // 0 → 30 days
}

// Module is the self-contained customer_auth platform.
type Module struct {
	cfg     Config
	handler *cahttp.CustomerAuthHandler
}

// New wires the customer_auth module.
func New(cfg Config) (*Module, error) {
	if cfg.CredRepo == nil {
		return nil, errors.New("customer_auth.Config: CredRepo is required")
	}
	if cfg.RefreshRepo == nil {
		return nil, errors.New("customer_auth.Config: RefreshRepo is required")
	}
	if cfg.ResetRepo == nil {
		return nil, errors.New("customer_auth.Config: ResetRepo is required")
	}
	if cfg.Hasher == nil {
		return nil, errors.New("customer_auth.Config: Hasher is required")
	}
	if cfg.JWTSvc == nil {
		return nil, errors.New("customer_auth.Config: JWTSvc is required")
	}
	if cfg.Publisher == nil {
		return nil, errors.New("customer_auth.Config: Publisher is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("customer_auth.Config: Logger is required")
	}

	refreshTTL := cfg.RefreshTTL
	if refreshTTL == 0 {
		refreshTTL = defaultRefreshTTL
	}

	registerUC := usecase.NewCustomerRegisterUseCase(cfg.DB, cfg.CredRepo, cfg.Hasher, cfg.Publisher)
	loginUC := usecase.NewCustomerLoginUseCase(cfg.DB, cfg.CredRepo, cfg.RefreshRepo, cfg.Hasher, cfg.JWTSvc, cfg.Publisher, refreshTTL)
	refreshUC := usecase.NewCustomerRefreshUseCase(cfg.DB, cfg.RefreshRepo, cfg.JWTSvc, refreshTTL)
	logoutUC := usecase.NewCustomerLogoutUseCase(cfg.RefreshRepo)
	forgotUC := usecase.NewCustomerForgotPasswordUseCase(cfg.DB, cfg.CredRepo, cfg.ResetRepo, cfg.Publisher)
	resetUC := usecase.NewCustomerResetPasswordUseCase(cfg.DB, cfg.CredRepo, cfg.ResetRepo, cfg.Hasher, cfg.Publisher)

	handler := cahttp.NewCustomerAuthHandler(registerUC, loginUC, refreshUC, logoutUC, forgotUC, resetUC)

	return &Module{cfg: cfg, handler: handler}, nil
}

// RegisterRoutes mounts the customer_auth endpoints on mux.
//
//	POST /api/v1/tenants/{id}/customers/auth/register
//	POST /api/v1/tenants/{id}/customers/auth/login
//	POST /api/v1/tenants/{id}/customers/auth/refresh
//	POST /api/v1/tenants/{id}/customers/auth/logout
//	POST /api/v1/tenants/{id}/customers/auth/forgot-password
//	POST /api/v1/tenants/{id}/customers/auth/reset-password
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/tenants/{id}/customers/auth/register", m.handler.Register)
	mux.HandleFunc("POST /api/v1/tenants/{id}/customers/auth/login", m.handler.Login)
	mux.HandleFunc("POST /api/v1/tenants/{id}/customers/auth/refresh", m.handler.Refresh)
	mux.HandleFunc("POST /api/v1/tenants/{id}/customers/auth/logout", m.handler.Logout)
	mux.HandleFunc("POST /api/v1/tenants/{id}/customers/auth/forgot-password", m.handler.ForgotPassword)
	mux.HandleFunc("POST /api/v1/tenants/{id}/customers/auth/reset-password", m.handler.ResetPassword)
}

// CustomerRequired returns an HTTP middleware that validates customer JWTs.
// Wrap any handler that should only be accessible to authenticated end-customers.
func (m *Module) CustomerRequired(next http.Handler) http.Handler {
	return cahttp.CustomerRequired(m.cfg.JWTSvc, next)
}
