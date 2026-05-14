// Package tenant is the tenant-organisation platform. It exposes a
// self-contained Module that wires the tenant repository, use cases and HTTP
// handlers. Consumers depend on the Repository interface defined in
// platform/tenant/domain for cross-platform lookups.
package tenant

import (
	"log/slog"
	"net/http"

	"github.com/AgileFactory/kernel/platform/audit"
	tenanthttp "github.com/AgileFactory/kernel/platform/tenant/delivery/http"
	"github.com/AgileFactory/kernel/platform/tenant/domain"
	"github.com/AgileFactory/kernel/platform/tenant/usecase"
)

// Repository re-exports the tenant repository contract for consumers.
type Repository = domain.TenantRepository

// Config captures the dependencies required by the tenant Module.
type Config struct {
	Repo        Repository
	AuditLogger audit.EventLogger
	Logger      *slog.Logger
}

// Module is the self-contained tenant platform.
type Module struct {
	cfg Config

	CreateUC *usecase.CreateTenantUseCase
	FindUC   *usecase.FindTenantUseCase

	handler *tenanthttp.TenantHandler
}

// New wires the tenant module's internal components.
func New(cfg Config) *Module {
	createUC := usecase.NewCreateTenantUseCase(cfg.Repo, cfg.AuditLogger)
	findUC := usecase.NewFindTenantUseCase(cfg.Repo)

	return &Module{
		cfg:      cfg,
		CreateUC: createUC,
		FindUC:   findUC,
		handler:  tenanthttp.NewTenantHandler(createUC, findUC),
	}
}

// RegisterRoutes mounts the tenant HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	tenanthttp.RegisterRoutes(mux, m.handler)
}
