// Package auth is the authentication platform. It exposes a self-contained
// Module that wires the user repository, JWT service, password hasher, use
// cases and HTTP handlers.
package auth

import (
	"embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/AgileFactory/kernel/platform/audit"
	authhttp "github.com/AgileFactory/kernel/platform/auth/delivery/http"
	"github.com/AgileFactory/kernel/platform/auth/domain"
	"github.com/AgileFactory/kernel/platform/auth/infrastructure"
	"github.com/AgileFactory/kernel/platform/auth/usecase"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
	memberusecase "github.com/AgileFactory/kernel/platform/membership/usecase"
	tenantdomain "github.com/AgileFactory/kernel/platform/tenant/domain"
)

// Migrations holds the SQL files this platform owns. Consumers compose them
// via migrator.Run alongside their own modules' migrations.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// UserRepository re-exports the user repository contract for consumers.
type UserRepository = domain.UserRepository

// Config captures the dependencies required by the auth Module.
type Config struct {
	UserRepo         UserRepository
	TenantRepo       tenantdomain.TenantRepository
	MembershipRepo   memberdomain.MembershipRepository
	MembershipListUC *memberusecase.ListMembersUseCase
	AuditLogger      audit.EventLogger
	Logger           *slog.Logger

	JWTSecret     string
	TokenLifespan time.Duration
	BCryptCost    int // 0 = bcrypt default
}

// Module is the self-contained auth platform.
type Module struct {
	cfg Config

	RegisterUC *usecase.RegisterUseCase
	LoginUC    *usecase.LoginUseCase

	jwt     *infrastructure.JWTService
	hasher  *infrastructure.BcryptHasher
	handler *authhttp.AuthHandler
}

// New wires the auth module's internal components.
func New(cfg Config) *Module {
	hasher := infrastructure.NewBcryptHasher(cfg.BCryptCost)
	jwt := infrastructure.NewJWTService(cfg.JWTSecret, cfg.TokenLifespan)

	registerUC := usecase.NewRegisterUseCase(
		cfg.UserRepo,
		hasher,
		cfg.TenantRepo,
		cfg.MembershipRepo,
		cfg.AuditLogger,
	)
	loginUC := usecase.NewLoginUseCase(cfg.UserRepo, hasher, jwt, cfg.AuditLogger)

	return &Module{
		cfg:        cfg,
		RegisterUC: registerUC,
		LoginUC:    loginUC,
		jwt:        jwt,
		hasher:     hasher,
		handler:    authhttp.NewAuthHandler(registerUC, loginUC, cfg.MembershipListUC),
	}
}

// JWTService exposes the auth module's JWT signer/validator for middlewares
// that need to verify tokens issued by this module.
func (m *Module) JWTService() *infrastructure.JWTService {
	return m.jwt
}

// RegisterRoutes mounts the auth HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	authhttp.RegisterRoutes(mux, m.handler)
}
