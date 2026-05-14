// Package audit is the audit-trail platform. It exposes a self-contained
// Module that wires the audit use cases and HTTP handlers, plus the
// cross-platform contracts (EventLogger) other platforms depend on.
//
// Consumers should depend on the EventLogger interface rather than the
// concrete *usecase.LogEventUseCase, so the audit implementation can evolve
// (e.g. swap best-effort fire-and-forget for the outbox pattern in RFC-0015)
// without breaking dependent platforms.
package audit

import (
	"context"
	"log/slog"
	"net/http"

	audithttp "github.com/AgileFactory/kernel/platform/audit/delivery/http"
	"github.com/AgileFactory/kernel/platform/audit/domain"
	"github.com/AgileFactory/kernel/platform/audit/usecase"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
)

// LogEventParams re-exports the parameters used by EventLogger.Execute so
// consumers only need to import this package.
type LogEventParams = usecase.LogEventParams

// EventLogger is the producer-side contract for emitting audit events.
// Other platforms depend on this interface, not on *usecase.LogEventUseCase.
type EventLogger interface {
	Execute(ctx context.Context, params LogEventParams)
}

// Config captures the dependencies required by the audit Module. Repos are
// constructed outside the module so platforms can share a single repo
// instance and avoid a construction cycle (audit needs a membership reader
// for HTTP perm checks, while membership use cases need audit.EventLogger).
type Config struct {
	Repo           domain.AuditRepository
	MembershipRepo memberdomain.MembershipRepository
	Logger         *slog.Logger
}

// Module is the self-contained audit platform.
type Module struct {
	cfg Config

	LogEventUC *usecase.LogEventUseCase
	ListUC     *usecase.ListAuditLogsUseCase

	handler   *audithttp.AuditHandler
	handlerV2 *audithttp.AuditHandlerV2
}

// New wires the audit module's internal components.
func New(cfg Config) *Module {
	logUC := usecase.NewLogEventUseCase(cfg.Repo)
	listUC := usecase.NewListAuditLogsUseCase(cfg.Repo)

	return &Module{
		cfg:        cfg,
		LogEventUC: logUC,
		ListUC:     listUC,
		handler:    audithttp.NewAuditHandler(listUC, cfg.MembershipRepo),
		handlerV2:  audithttp.NewAuditHandlerV2(listUC, cfg.MembershipRepo),
	}
}

// EventLogger returns the audit producer contract satisfied by this module.
// Other platforms wire this into their own Config.
func (m *Module) EventLogger() EventLogger {
	return m.LogEventUC
}

// RegisterRoutes mounts the audit HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	audithttp.RegisterRoutes(mux, m.handler, m.handlerV2)
}
