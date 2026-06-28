// Package billing is the subscription and payment billing platform.
// It implements Platform Mode (Stripe + Wave for plan subscriptions).
//
// Wiring:
//
//	stripeAdapter := adapter.NewStripeAdapter(adapter.StripeConfig{...})
//	billingMod := billing.New(billing.Config{
//	    SubRepo:   billingInfra.NewPostgresSubscriptionRepository(db),
//	    PlanRepo:  billingInfra.NewPostgresPlanRepository(db),
//	    Processor: stripeAdapter,
//	    Checker:   checker,
//	    Logger:    log,
//	})
//	billingMod.RegisterRoutes(mux)
package billing

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/permission"
	billinghttp "github.com/agile-factory/kernel/platform/billing/delivery/http"
	"github.com/agile-factory/kernel/platform/billing/domain"
	"github.com/agile-factory/kernel/platform/billing/usecase"
)

// Migrations holds the SQL files this platform owns. Consumed by migrator.Run.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Config captures the dependencies of the billing Module.
type Config struct {
	SubRepo   domain.SubscriptionRepository
	PlanRepo  domain.PlanRepository
	Processor domain.PaymentProcessor
	Checker   permission.Checker
	Logger    *slog.Logger
}

// Module is the self-contained billing platform.
type Module struct {
	cfg     Config
	handler *billinghttp.BillingHandler
}

// New wires the billing module.
func New(cfg Config) (*Module, error) {
	if cfg.SubRepo == nil {
		return nil, errors.New("billing.Config: SubRepo is required")
	}
	if cfg.PlanRepo == nil {
		return nil, errors.New("billing.Config: PlanRepo is required")
	}
	if cfg.Processor == nil {
		return nil, errors.New("billing.Config: Processor is required")
	}
	if cfg.Checker == nil {
		return nil, errors.New("billing.Config: Checker is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("billing.Config: Logger is required")
	}

	listPlansUC := usecase.NewListPlansUseCase(cfg.PlanRepo)
	getSubUC := usecase.NewGetSubscriptionUseCase(cfg.SubRepo)

	return &Module{
		cfg:     cfg,
		handler: billinghttp.NewBillingHandler(listPlansUC, getSubUC, cfg.Processor, cfg.Checker),
	}, nil
}

// RegisterRoutes mounts the billing endpoints on mux.
//
//	GET  /api/v1/plans
//	GET  /api/v1/tenants/{tenantId}/subscription
//	POST /api/v1/billing/webhooks/stripe
//	POST /api/v1/billing/webhooks/wave
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/plans", m.handler.ListPlans)
	mux.HandleFunc("GET /api/v1/tenants/{tenantId}/subscription", m.handler.GetSubscription)
	mux.HandleFunc("POST /api/v1/billing/webhooks/stripe", m.handler.StripeWebhook)
	mux.HandleFunc("POST /api/v1/billing/webhooks/wave", m.handler.WaveWebhook)
}
