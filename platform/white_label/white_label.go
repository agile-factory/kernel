// Package white_label is the per-tenant branding platform (RFC-0035).
// It manages BrandSettings — visual identity, custom domain DNS verification,
// and email template overrides for the B2B2C customer-facing portal.
//
// Wiring:
//
//	repo    := wlinfra.NewPostgresBrandRepository(db)
//	checker := permission.New(memberRepo)
//	wlMod, err := white_label.New(white_label.Config{
//	    DB:        db,
//	    Repo:      repo,
//	    Checker:   checker,
//	    Publisher: bus.Publisher(),
//	    Logger:    log,
//	})
//	wlMod.RegisterRoutes(mux)
package white_label

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	wlhttp "github.com/agile-factory/kernel/platform/white_label/delivery/http"
	"github.com/agile-factory/kernel/platform/white_label/domain"
	"github.com/agile-factory/kernel/platform/white_label/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds all SQL migration files for this platform.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Config captures the dependencies of the white_label Module.
type Config struct {
	DB          *pgxpool.Pool        // may be nil in unit tests
	Repo        domain.BrandRepository
	Checker     permission.Checker
	Publisher   events.Publisher
	Logger      *slog.Logger
	DNSResolver usecase.DNSResolver  // nil → stdlib net.DefaultResolver
}

// Module is the self-contained white_label platform.
type Module struct {
	cfg     Config
	handler *wlhttp.BrandHandler
}

// New wires the white_label module.
// cfg.DB may be nil in unit tests when all repositories are stubs.
func New(cfg Config) (*Module, error) {
	if cfg.Repo == nil {
		return nil, errors.New("white_label.Config: Repo is required")
	}
	if cfg.Checker == nil {
		return nil, errors.New("white_label.Config: Checker is required")
	}
	if cfg.Publisher == nil {
		return nil, errors.New("white_label.Config: Publisher is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("white_label.Config: Logger is required")
	}

	resolver := cfg.DNSResolver
	if resolver == nil {
		resolver = usecase.NetResolver()
	}

	getUC    := usecase.NewGetBrandSettingsUseCase(cfg.Repo, cfg.Checker)
	updateUC := usecase.NewUpdateBrandSettingsUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher)
	verifyUC := usecase.NewVerifyCustomDomainUseCase(cfg.DB, cfg.Repo, cfg.Checker, cfg.Publisher, resolver)
	publicUC := usecase.NewGetPublicBrandUseCase(cfg.Repo)

	handler := wlhttp.NewBrandHandler(getUC, updateUC, verifyUC, publicUC)

	return &Module{cfg: cfg, handler: handler}, nil
}

// RegisterRoutes mounts the white_label endpoints on mux.
//
//	GET    /api/v1/tenants/{id}/brand
//	PATCH  /api/v1/tenants/{id}/brand
//	POST   /api/v1/tenants/{id}/brand/verify-domain
//	GET    /public/v1/brand
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tenants/{id}/brand", m.handler.GetBrand)
	mux.HandleFunc("PATCH /api/v1/tenants/{id}/brand", m.handler.UpdateBrand)
	mux.HandleFunc("POST /api/v1/tenants/{id}/brand/verify-domain", m.handler.VerifyDomain)
	mux.HandleFunc("GET /public/v1/brand", m.handler.GetPublicBrand)
}
