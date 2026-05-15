// Package tenant is the tenant-organisation platform. It exposes a
// self-contained Module that wires the tenant + tenant_settings repositories,
// use cases and HTTP handlers. Consumers depend on the Repository interface
// defined in platform/tenant/domain for cross-platform lookups.
package tenant

import (
	"embed"
	"log/slog"
	"net/http"

	"github.com/AgileFactory/kernel/platform/audit"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
	tenanthttp "github.com/AgileFactory/kernel/platform/tenant/delivery/http"
	"github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/AgileFactory/kernel/platform/tenant/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the SQL files this platform owns. Consumers compose them
// via migrator.Run alongside their own modules' migrations.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Repository re-exports the tenant repository contract for consumers.
type Repository = domain.TenantRepository

// SettingsRepository re-exports the tenant_settings repository contract.
type SettingsRepository = domain.TenantSettingsRepository

// Config captures the dependencies required by the tenant Module.
type Config struct {
	Pool         *pgxpool.Pool
	Repo         Repository
	SettingsRepo SettingsRepository
	// MembershipRepo is consulted by the HTTP layer for OWNER/ADMIN gating
	// on PATCH/DELETE/settings endpoints. Without it, those endpoints
	// would have no way to authorise the caller.
	MembershipRepo memberdomain.MembershipRepository
	AuditLogger    audit.EventLogger
	Logger         *slog.Logger
}

// Module is the self-contained tenant platform.
type Module struct {
	cfg Config

	CreateUC         *usecase.CreateTenantUseCase
	FindUC           *usecase.FindTenantUseCase
	UpdateUC         *usecase.UpdateTenantUseCase
	DeleteUC         *usecase.DeleteTenantUseCase
	GetSettingsUC    *usecase.GetTenantSettingsUseCase
	UpdateSettingsUC *usecase.UpdateTenantSettingsUseCase

	handler *tenanthttp.TenantHandler
}

// New wires the tenant module's internal components.
func New(cfg Config) *Module {
	createUC := usecase.NewCreateTenantUseCase(cfg.Pool, cfg.Repo, cfg.SettingsRepo, cfg.AuditLogger)
	findUC := usecase.NewFindTenantUseCase(cfg.Repo)
	updateUC := usecase.NewUpdateTenantUseCase(cfg.Repo, cfg.AuditLogger)
	deleteUC := usecase.NewDeleteTenantUseCase(cfg.Repo, cfg.AuditLogger)
	getSettingsUC := usecase.NewGetTenantSettingsUseCase(cfg.SettingsRepo)
	updateSettingsUC := usecase.NewUpdateTenantSettingsUseCase(cfg.SettingsRepo, cfg.AuditLogger)

	return &Module{
		cfg:              cfg,
		CreateUC:         createUC,
		FindUC:           findUC,
		UpdateUC:         updateUC,
		DeleteUC:         deleteUC,
		GetSettingsUC:    getSettingsUC,
		UpdateSettingsUC: updateSettingsUC,
		handler: tenanthttp.NewTenantHandler(
			createUC, findUC, updateUC, deleteUC,
			getSettingsUC, updateSettingsUC,
			cfg.MembershipRepo,
		),
	}
}

// RegisterRoutes mounts the tenant HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	tenanthttp.RegisterRoutes(mux, m.handler)
}
