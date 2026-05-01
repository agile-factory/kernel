package bootstrap

import (
	"log/slog"
	"net/http"

	deliveryhttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/delivery/http"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/config"
	audithttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/delivery/http"
	auditinfra "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/infrastructure"
	auditusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
	authhttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/delivery/http"
	authmock "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/infrastructure"
	authusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/usecase"
	memberhttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/delivery/http"
	memberinfra "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/infrastructure"
	memberusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/usecase"
	tenanthttp "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/delivery/http"
	tenantinfra "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/infrastructure"
	tenantusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/usecase"
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
	audithttp.RegisterRoutes(c.mux, auditHandler)

	// TODO: Register module routes here (billing, notifications, etc.)
}


