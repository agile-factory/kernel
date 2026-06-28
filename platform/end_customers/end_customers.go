// Package end_customers is the B2B2C end-customer platform (RFC-0033).
// It manages end-customers — the consumers of a tenant's customer-facing
// portal. Distinct from auth.User (workspace members).
//
// Wiring:
//
//	repo    := ecinfra.NewPostgresEndCustomerRepository(db)
//	checker := permission.New(memberRepo)
//	ecMod, err := end_customers.New(end_customers.Config{
//	    DB:        db,
//	    Repo:      repo,
//	    Checker:   checker,
//	    Publisher: bus.Publisher(),
//	    Logger:    log,
//	})
//	ecMod.RegisterRoutes(mux)
package end_customers

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	echttp "github.com/agile-factory/kernel/platform/end_customers/delivery/http"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
	"github.com/agile-factory/kernel/platform/end_customers/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds all SQL migration files for this platform.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Config captures the dependencies of the end_customers Module.
type Config struct {
	DB        *pgxpool.Pool
	Repo      domain.EndCustomerRepository
	Checker   permission.Checker
	Publisher events.Publisher
	Logger    *slog.Logger
}

// Module is the self-contained end_customers platform.
type Module struct {
	cfg     Config
	handler *echttp.EndCustomerHandler
}

// New wires the end_customers module.
// cfg.DB may be nil in unit tests when all repositories are stubs; write use
// cases bypass txctx.ExecInTx transparently via execWithOptionalTx.
func New(cfg Config) (*Module, error) {
	if cfg.Repo == nil {
		return nil, errors.New("end_customers.Config: Repo is required")
	}
	if cfg.Checker == nil {
		return nil, errors.New("end_customers.Config: Checker is required")
	}
	if cfg.Publisher == nil {
		return nil, errors.New("end_customers.Config: Publisher is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("end_customers.Config: Logger is required")
	}

	createUC := usecase.NewCreateEndCustomerUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher)
	getUC := usecase.NewGetEndCustomerUseCase(cfg.Repo, cfg.Checker)
	updateUC := usecase.NewUpdateEndCustomerUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher)
	deleteUC := usecase.NewDeleteEndCustomerUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher)
	suspendUC := usecase.NewSuspendEndCustomerUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher)
	restoreUC := usecase.NewRestoreEndCustomerUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher)
	listUC := usecase.NewListEndCustomersUseCase(cfg.Repo, cfg.Checker)

	handler := echttp.NewEndCustomerHandler(createUC, getUC, updateUC, deleteUC, suspendUC, restoreUC, listUC)

	return &Module{cfg: cfg, handler: handler}, nil
}

// RegisterRoutes mounts the end_customers endpoints on mux.
//
//	POST   /api/v1/end-customers
//	GET    /api/v1/end-customers
//	GET    /api/v1/end-customers/{id}
//	PATCH  /api/v1/end-customers/{id}
//	DELETE /api/v1/end-customers/{id}
//	POST   /api/v1/end-customers/{id}/suspend
//	POST   /api/v1/end-customers/{id}/restore
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/end-customers", m.handler.Create)
	mux.HandleFunc("GET /api/v1/end-customers", m.handler.List)
	mux.HandleFunc("GET /api/v1/end-customers/{id}", m.handler.Get)
	mux.HandleFunc("PATCH /api/v1/end-customers/{id}", m.handler.Update)
	mux.HandleFunc("DELETE /api/v1/end-customers/{id}", m.handler.Delete)
	mux.HandleFunc("POST /api/v1/end-customers/{id}/suspend", m.handler.Suspend)
	mux.HandleFunc("POST /api/v1/end-customers/{id}/restore", m.handler.Restore)
}
