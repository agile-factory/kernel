// Package auth is the authentication platform. It exposes a self-contained
// Module that wires the user repository, JWT service, password hasher,
// signup-owner / accept-invite / verify-email / login / refresh / logout use
// cases and HTTP handlers. Aligned with RFC-0018.
package auth

import (
	"embed"
	"log/slog"
	"net/http"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/audit"
	authhttp "github.com/agile-factory/kernel/platform/auth/delivery/http"
	"github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/agile-factory/kernel/platform/auth/infrastructure"
	"github.com/agile-factory/kernel/platform/auth/usecase"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
	memberusecase "github.com/agile-factory/kernel/platform/membership/usecase"
	tenantdomain "github.com/agile-factory/kernel/platform/tenant/domain"
	"github.com/jackc/pgx/v5/pgxpool"
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
	// Database pool — required for the use cases that wrap their work in a
	// transaction (SignupOwner, AcceptInvite, VerifyEmail, RefreshToken).
	Pool *pgxpool.Pool

	UserRepo           UserRepository
	TenantRepo         tenantdomain.TenantRepository
	TenantSettingsRepo tenantdomain.TenantSettingsRepository
	MembershipRepo     memberdomain.MembershipRepository
	InviteTokenRepo    memberdomain.InviteTokenRepository
	RefreshTokenRepo   domain.RefreshTokenRepository
	EmailVerifRepo     domain.EmailVerificationTokenRepository
	MembershipListUC   *memberusecase.ListMembersUseCase

	AuditLogger audit.EventLogger
	Publisher   events.Publisher // nil-safe; events.signup_owner etc. skipped when nil
	Logger      *slog.Logger

	JWTSecret       string
	TokenLifespan   time.Duration
	RefreshTokenTTL time.Duration
	BCryptCost      int // 0 = bcrypt default
}

// Module is the self-contained auth platform.
type Module struct {
	cfg Config

	SignupOwnerUC  *usecase.SignupOwnerUseCase
	AcceptInviteUC *usecase.AcceptInviteUseCase
	VerifyEmailUC  *usecase.VerifyEmailUseCase
	LoginUC        *usecase.LoginUseCase
	RefreshUC      *usecase.RefreshTokenUseCase
	LogoutUC       *usecase.LogoutUseCase

	jwt     *infrastructure.JWTService
	hasher  *infrastructure.BcryptHasher
	handler *authhttp.AuthHandler
}

// New wires the auth module's internal components.
func New(cfg Config) *Module {
	if cfg.RefreshTokenTTL == 0 {
		cfg.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	hasher := infrastructure.NewBcryptHasher(cfg.BCryptCost)
	jwt := infrastructure.NewJWTService(cfg.JWTSecret, cfg.TokenLifespan)

	signupOwnerUC := usecase.NewSignupOwnerUseCase(
		cfg.Pool,
		cfg.UserRepo,
		cfg.TenantRepo,
		cfg.TenantSettingsRepo,
		cfg.MembershipRepo,
		cfg.EmailVerifRepo,
		cfg.RefreshTokenRepo,
		hasher,
		jwt,
		cfg.Publisher,
		cfg.AuditLogger,
		cfg.RefreshTokenTTL,
	)
	acceptInviteUC := usecase.NewAcceptInviteUseCase(
		cfg.Pool,
		cfg.UserRepo,
		cfg.MembershipRepo,
		cfg.InviteTokenRepo,
		cfg.RefreshTokenRepo,
		hasher,
		jwt,
		cfg.Publisher,
		cfg.AuditLogger,
		cfg.RefreshTokenTTL,
	)
	verifyEmailUC := usecase.NewVerifyEmailUseCase(cfg.Pool, cfg.UserRepo, cfg.EmailVerifRepo)
	loginUC := usecase.NewLoginUseCase(cfg.UserRepo, cfg.RefreshTokenRepo, hasher, jwt, cfg.AuditLogger, cfg.RefreshTokenTTL)
	refreshUC := usecase.NewRefreshTokenUseCase(cfg.Pool, cfg.UserRepo, cfg.RefreshTokenRepo, jwt, cfg.RefreshTokenTTL)
	logoutUC := usecase.NewLogoutUseCase(cfg.RefreshTokenRepo)

	return &Module{
		cfg:            cfg,
		SignupOwnerUC:  signupOwnerUC,
		AcceptInviteUC: acceptInviteUC,
		VerifyEmailUC:  verifyEmailUC,
		LoginUC:        loginUC,
		RefreshUC:      refreshUC,
		LogoutUC:       logoutUC,
		jwt:            jwt,
		hasher:         hasher,
		handler: authhttp.NewAuthHandler(
			signupOwnerUC, acceptInviteUC, verifyEmailUC,
			loginUC, refreshUC, logoutUC,
			cfg.MembershipListUC,
		),
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
