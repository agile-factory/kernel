// Package customer_notifications is the B2B2C end-customer notification platform (RFC-0036).
// It dispatches email and SMS notifications triggered by outbox events, with per-tenant
// template overrides and branded content from platform/white_label.
//
// Wiring:
//
//	cnMod, err := customer_notifications.New(customer_notifications.Config{
//	    DB:               db,
//	    LogRepo:          cninfra.NewPostgresNotificationLogRepository(db),
//	    TemplateRepo:     cninfra.NewPostgresCustomerTemplateRepository(db),
//	    CustomerResolver: cninfra.NewPostgresEndCustomerResolver(db),
//	    BrandResolver:    cninfra.NewPostgresBrandResolver(db),
//	    EmailSender:      myEmailAdapter,
//	    SMSSender:        cninfra.NewNoopSMSAdapter(log),
//	    Checker:          checker,
//	    Logger:           log,
//	})
//	bus.Subscribe("customer_auth.password_reset_requested", cnMod.HandleEvent)
//	cnMod.RegisterRoutes(mux)
package customer_notifications

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"text/template"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/permission"
	cnhttp "github.com/agile-factory/kernel/platform/customer_notifications/delivery/http"
	"github.com/agile-factory/kernel/platform/customer_notifications/domain"
	"github.com/agile-factory/kernel/platform/customer_notifications/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds all SQL migration files for this platform.
//
//go:embed migrations/*.sql
var Migrations embed.FS

//go:embed templates
var defaultTemplatesFS embed.FS

// Config captures the dependencies of the customer_notifications Module.
type Config struct {
	DB               *pgxpool.Pool                         // may be nil in unit tests
	LogRepo          domain.NotificationLogRepository
	TemplateRepo     domain.CustomerTemplateRepository
	CustomerResolver domain.EndCustomerResolver
	BrandResolver    domain.BrandResolver
	EmailSender      domain.EmailSender
	SMSSender        domain.SMSSender
	Checker          permission.Checker
	Logger           *slog.Logger
}

// Module is the self-contained customer_notifications platform.
// Event handlers are stored as instance state (not global vars) per RFC-0038 and Aziz amendment.
type Module struct {
	cfg        Config
	handler    *cnhttp.CustomerNotificationsHandler
	sendUC     *usecase.SendCustomerNotificationUseCase
	eventTypes []string // event types this module handles
}

// New wires the customer_notifications module.
// Templates are parsed at boot to detect syntax errors immediately (Lucas amendment).
func New(cfg Config) (*Module, error) {
	if cfg.LogRepo == nil {
		return nil, errors.New("customer_notifications.Config: LogRepo is required")
	}
	if cfg.TemplateRepo == nil {
		return nil, errors.New("customer_notifications.Config: TemplateRepo is required")
	}
	if cfg.CustomerResolver == nil {
		return nil, errors.New("customer_notifications.Config: CustomerResolver is required")
	}
	if cfg.BrandResolver == nil {
		return nil, errors.New("customer_notifications.Config: BrandResolver is required")
	}
	if cfg.EmailSender == nil {
		return nil, errors.New("customer_notifications.Config: EmailSender is required")
	}
	if cfg.SMSSender == nil {
		return nil, errors.New("customer_notifications.Config: SMSSender is required")
	}
	if cfg.Checker == nil {
		return nil, errors.New("customer_notifications.Config: Checker is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("customer_notifications.Config: Logger is required")
	}

	// Parse all embedded templates at boot (Lucas amendment).
	defaultTemplates, err := parseDefaultTemplates()
	if err != nil {
		return nil, err
	}

	sendUC := usecase.NewSendCustomerNotificationUseCase(
		cfg.DB,
		cfg.LogRepo,
		cfg.TemplateRepo,
		cfg.CustomerResolver,
		cfg.BrandResolver,
		cfg.EmailSender,
		cfg.SMSSender,
		defaultTemplates,
	)

	upsertUC := usecase.NewUpsertCustomerTemplateUseCase(cfg.TemplateRepo, cfg.Checker)
	deleteUC := usecase.NewDeleteCustomerTemplateUseCase(cfg.TemplateRepo, cfg.Checker)
	listLogsUC := usecase.NewListCustomerNotificationLogsUseCase(cfg.LogRepo, cfg.Checker)

	h := cnhttp.NewCustomerNotificationsHandler(upsertUC, deleteUC, listLogsUC)

	return &Module{
		cfg:        cfg,
		handler:    h,
		sendUC:     sendUC,
		eventTypes: []string{
			"end_customer.created",
			"customer_auth.registered",
			"customer_auth.password_reset_requested",
			"billing.invoice_paid",
			"billing.payment_reminder",
		},
	}, nil
}

// RegisterHandlers subscribes all event handlers to the given bus.Subscriber.
// Call this before bus.Run() to ensure handlers receive events from the first batch.
// This replaces the anti-pattern of a global mutable eventHandlers map (Aziz amendment).
func (m *Module) RegisterHandlers(sub events.Subscriber) {
	for _, evType := range m.eventTypes {
		evType := evType // capture loop variable
		sub.Subscribe(evType, func(ctx context.Context, ev events.Event) error {
			return m.handleEvent(ctx, ev)
		})
	}
}

// RegisterRoutes mounts the customer_notifications endpoints on mux.
//
//	GET    /api/v1/tenants/{id}/customer-notification-logs
//	PUT    /api/v1/tenants/{id}/customer-templates/{key}/{channel}
//	DELETE /api/v1/tenants/{id}/customer-templates/{key}/{channel}
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tenants/{id}/customer-notification-logs", m.handler.ListLogs)
	mux.HandleFunc("PUT /api/v1/tenants/{id}/customer-templates/{key}/{channel}", m.handler.UpsertTemplate)
	mux.HandleFunc("DELETE /api/v1/tenants/{id}/customer-templates/{key}/{channel}", m.handler.DeleteTemplate)
}

// handleEvent maps an outbox event to a notification dispatch.
func (m *Module) handleEvent(ctx context.Context, ev events.Event) error {
	payload := ev.Payload

	endCustomerID, _ := payload["end_customer_id"].(string)
	tenantID, _ := payload["tenant_id"].(string)
	if endCustomerID == "" || tenantID == "" {
		m.cfg.Logger.WarnContext(ctx, "customer_notifications: missing end_customer_id or tenant_id", "event_type", ev.Type)
		return nil
	}

	extraData := map[string]string{}

	switch ev.Type {
	case "end_customer.created", "customer_auth.registered":
		return m.sendUC.Execute(ctx, usecase.SendCustomerNotificationInput{
			TenantID:      tenantID,
			EndCustomerID: endCustomerID,
			TemplateKey:   "welcome",
			ExtraData:     extraData,
		})

	case "customer_auth.password_reset_requested":
		if token, ok := payload["reset_token"].(string); ok {
			extraData["otp_code"] = token
		}
		return m.sendUC.Execute(ctx, usecase.SendCustomerNotificationInput{
			TenantID:      tenantID,
			EndCustomerID: endCustomerID,
			TemplateKey:   "otp_reset",
			ExtraData:     extraData,
		})

	case "billing.invoice_paid":
		if amt, ok := payload["amount"].(string); ok {
			extraData["invoice_amount"] = amt
		}
		if cur, ok := payload["currency"].(string); ok {
			extraData["invoice_currency"] = cur
		}
		return m.sendUC.Execute(ctx, usecase.SendCustomerNotificationInput{
			TenantID:      tenantID,
			EndCustomerID: endCustomerID,
			TemplateKey:   "invoice_paid",
			ExtraData:     extraData,
		})

	case "billing.payment_reminder":
		if amt, ok := payload["amount"].(string); ok {
			extraData["invoice_amount"] = amt
		}
		if cur, ok := payload["currency"].(string); ok {
			extraData["invoice_currency"] = cur
		}
		return m.sendUC.Execute(ctx, usecase.SendCustomerNotificationInput{
			TenantID:      tenantID,
			EndCustomerID: endCustomerID,
			TemplateKey:   "payment_reminder",
			ExtraData:     extraData,
		})
	}
	return nil
}

// parseDefaultTemplates parses all embedded template files at boot.
// Returns a map keyed by "locale/key.channel.ext" (e.g. "fr/otp_reset.email.html").
func parseDefaultTemplates() (map[string]*template.Template, error) {
	templates := map[string]*template.Template{}
	sub, err := fs.Sub(defaultTemplatesFS, "templates")
	if err != nil {
		return nil, err
	}
	err = fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := fs.ReadFile(sub, path)
		if readErr != nil {
			return readErr
		}
		t, parseErr := template.New(path).Parse(string(data))
		if parseErr != nil {
			return parseErr
		}
		templates[path] = t
		return nil
	})
	return templates, err
}
