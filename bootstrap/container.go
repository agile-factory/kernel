package bootstrap

import (
	"log/slog"
	"net/http"

	deliveryhttp "github.com/AgileFactory/kernel/delivery/http"
	"github.com/AgileFactory/kernel/kernel/config"
	audithttp "github.com/AgileFactory/kernel/platform/audit/delivery/http"
	auditinfra "github.com/AgileFactory/kernel/platform/audit/infrastructure"
	auditusecase "github.com/AgileFactory/kernel/platform/audit/usecase"
	authhttp "github.com/AgileFactory/kernel/platform/auth/delivery/http"
	authmock "github.com/AgileFactory/kernel/platform/auth/infrastructure"
	authusecase "github.com/AgileFactory/kernel/platform/auth/usecase"
	memberhttp "github.com/AgileFactory/kernel/platform/membership/delivery/http"
	memberinfra "github.com/AgileFactory/kernel/platform/membership/infrastructure"
	memberusecase "github.com/AgileFactory/kernel/platform/membership/usecase"
	tenanthttp "github.com/AgileFactory/kernel/platform/tenant/delivery/http"
	tenantinfra "github.com/AgileFactory/kernel/platform/tenant/infrastructure"
	tenantusecase "github.com/AgileFactory/kernel/platform/tenant/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Container holds the application's dependencies
type Container struct {
	cfg *config.Config
	log *slog.Logger
	mux *http.ServeMux
}

// NewContainer initializes a new Container with all dependencies.
func NewContainer(cfg *config.Config, log *slog.Logger, db *pgxpool.Pool) *Container {
	c := &Container{
		cfg: cfg,
		log: log,
		mux: deliveryhttp.NewRouter(log),
	}

	c.setupRoutes(db)

	return c
}

// Router returns the configured HTTP multiplexer wrapped with global middlewares.
func (c *Container) Router() http.Handler {
	return deliveryhttp.ApplyGlobalMiddlewares(c.mux, c.log)
}

// setupRoutes initializes all the application routes.
func (c *Container) setupRoutes(db *pgxpool.Pool) {
	// Health check
	c.mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// ── Audit Platform ────────────────────────────────────────────────────────
	auditRepo := auditinfra.NewPostgresAuditRepository(db)
	logEventUC := auditusecase.NewLogEventUseCase(auditRepo)
	listAuditUC := auditusecase.NewListAuditLogsUseCase(auditRepo)

	// ── Tenant Platform ───────────────────────────────────────────────────────
	tenantRepo := tenantinfra.NewPostgresTenantRepository(db)
	createTenantUC := tenantusecase.NewCreateTenantUseCase(tenantRepo, logEventUC)
	findTenantUC := tenantusecase.NewFindTenantUseCase(tenantRepo)
	tenantHandler := tenanthttp.NewTenantHandler(createTenantUC, findTenantUC)
	tenanthttp.RegisterRoutes(c.mux, tenantHandler)

	// ── Membership Platform ───────────────────────────────────────────────────
	membershipRepo := memberinfra.NewPostgresMembershipRepository(db)
	inviteUC := memberusecase.NewInviteMemberUseCase(membershipRepo, tenantRepo, logEventUC)
	acceptUC := memberusecase.NewAcceptInviteUseCase(membershipRepo, logEventUC)
	revokeUC := memberusecase.NewRevokeMemberUseCase(membershipRepo, logEventUC)
	listUC := memberusecase.NewListMembersUseCase(membershipRepo)
	memberHandler := memberhttp.NewMembershipHandler(inviteUC, acceptUC, revokeUC, listUC)
	memberhttp.RegisterRoutes(c.mux, memberHandler)

	// ── Auth Platform ─────────────────────────────────────────────────────────
	// membershipRepo is passed to RegisterUseCase to auto-create a self-membership.
	// listUC is passed to AuthHandler so register/login responses include memberships.
	authRepo := authmock.NewPostgresUserRepository(db)
	hasher := authmock.NewBcryptHasher(0)
	jwtService := authmock.NewJWTService(c.cfg.JWTSecret, c.cfg.TokenLifespan)

	registerUC := authusecase.NewRegisterUseCase(authRepo, hasher, tenantRepo, membershipRepo, logEventUC)
	loginUC := authusecase.NewLoginUseCase(authRepo, hasher, jwtService, logEventUC)

	authHandler := authhttp.NewAuthHandler(registerUC, loginUC, listUC)
	authhttp.RegisterRoutes(c.mux, authHandler)

	// ── Audit Platform (read model) ───────────────────────────────────────────
	auditHandler := audithttp.NewAuditHandler(listAuditUC, membershipRepo)
	auditHandlerV2 := audithttp.NewAuditHandlerV2(listAuditUC, membershipRepo)
	audithttp.RegisterRoutes(c.mux, auditHandler, auditHandlerV2)

	// TODO: Register module routes here (billing, notifications, etc.)
}


