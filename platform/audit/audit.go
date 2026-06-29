// Package audit is the audit-trail platform. It exposes a self-contained
// Module that wires the audit use cases and HTTP handlers, plus the
// cross-platform contracts (EventLogger) other platforms depend on.
//
// Consumers should depend on the EventLogger interface rather than the
// concrete *usecase.LogEventUseCase, so the audit implementation can evolve
// without breaking dependent platforms.
package audit

import (
	"context"
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/events"
	audithttp "github.com/agile-factory/kernel/platform/audit/delivery/http"
	"github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/agile-factory/kernel/platform/audit/usecase"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the SQL files this platform owns. Consumers compose them
// via migrator.Run alongside their own modules' migrations.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// LogEventParams re-exports the parameters used by EventLogger.Execute so
// consumers only need to import this package.
type LogEventParams = usecase.LogEventParams

// EventLogger is the producer-side contract for emitting audit events.
// Other platforms depend on this interface, not on *usecase.LogEventUseCase.
type EventLogger interface {
	Execute(ctx context.Context, params LogEventParams)
}

// Config captures the dependencies required by the audit Module. All fields
// are mandatory — New returns an error if any are nil.
type Config struct {
	DB             *pgxpool.Pool                     // dedicated tx for outbox publish
	Repo           domain.AuditRepository
	MembershipRepo memberdomain.MembershipRepository
	Publisher      events.Publisher                  // outbox publisher (RFC-0015)
	Logger         *slog.Logger
}

// Module is the self-contained audit platform.
type Module struct {
	cfg Config

	logUC     *usecase.LogEventUseCase
	consumeUC *usecase.ConsumeAuditLogUseCase
	ListUC    *usecase.ListAuditLogsUseCase

	handler   *audithttp.AuditHandler
	handlerV2 *audithttp.AuditHandlerV2
}

// New wires the audit module's internal components.
// Returns an error if any required Config field is nil.
func New(cfg Config) (*Module, error) {
	var errs []error
	if cfg.DB == nil {
		errs = append(errs, errors.New("audit.Config: DB is required"))
	}
	if cfg.Repo == nil {
		errs = append(errs, errors.New("audit.Config: Repo is required"))
	}
	if cfg.MembershipRepo == nil {
		errs = append(errs, errors.New("audit.Config: MembershipRepo is required"))
	}
	if cfg.Publisher == nil {
		errs = append(errs, errors.New("audit.Config: Publisher is required"))
	}
	if cfg.Logger == nil {
		errs = append(errs, errors.New("audit.Config: Logger is required"))
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	logUC := usecase.NewLogEventUseCase(cfg.DB, cfg.Publisher)
	consumeUC := usecase.NewConsumeAuditLogUseCase(cfg.Repo)
	listUC := usecase.NewListAuditLogsUseCase(cfg.Repo)

	return &Module{
		cfg:       cfg,
		logUC:     logUC,
		consumeUC: consumeUC,
		ListUC:    listUC,
		handler:   audithttp.NewAuditHandler(listUC, cfg.MembershipRepo),
		handlerV2: audithttp.NewAuditHandlerV2(listUC, cfg.MembershipRepo),
	}, nil
}

// EventLogger returns the audit producer contract satisfied by this module.
// Other platforms wire this into their own Config.
func (m *Module) EventLogger() EventLogger {
	return m.logUC
}

// Subscribe registers the audit consumer with the events bus.
// Call before bus.Run to ensure audit.log_created events are delivered to
// audit_logs with at-least-once guarantees.
func (m *Module) Subscribe(sub events.Subscriber) {
	sub.Subscribe("audit.log_created", m.consumeUC.Handle)
}

// RegisterRoutes mounts the audit HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	audithttp.RegisterRoutes(mux, m.handler, m.handlerV2)
}
