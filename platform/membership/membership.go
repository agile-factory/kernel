// Package membership is the membership-lifecycle platform. It exposes a
// self-contained Module that wires the membership repository, use cases and
// HTTP handlers. Consumers depend on the Repository interface (defined in
// platform/membership/domain) for cross-platform lookups.
package membership

import (
	"log/slog"
	"net/http"

	"github.com/AgileFactory/kernel/platform/audit"
	memberhttp "github.com/AgileFactory/kernel/platform/membership/delivery/http"
	"github.com/AgileFactory/kernel/platform/membership/domain"
	"github.com/AgileFactory/kernel/platform/membership/usecase"
	tenantdomain "github.com/AgileFactory/kernel/platform/tenant/domain"
)

// Repository re-exports the membership repository contract for consumers.
type Repository = domain.MembershipRepository

// Config captures the dependencies required by the membership Module.
type Config struct {
	Repo        Repository
	TenantRepo  tenantdomain.TenantRepository
	AuditLogger audit.EventLogger
	Logger      *slog.Logger
}

// Module is the self-contained membership platform.
type Module struct {
	cfg Config

	InviteUC *usecase.InviteMemberUseCase
	AcceptUC *usecase.AcceptInviteUseCase
	RevokeUC *usecase.RevokeMemberUseCase
	ListUC   *usecase.ListMembersUseCase

	handler *memberhttp.MembershipHandler
}

// New wires the membership module's internal components.
func New(cfg Config) *Module {
	inviteUC := usecase.NewInviteMemberUseCase(cfg.Repo, cfg.TenantRepo, cfg.AuditLogger)
	acceptUC := usecase.NewAcceptInviteUseCase(cfg.Repo, cfg.AuditLogger)
	revokeUC := usecase.NewRevokeMemberUseCase(cfg.Repo, cfg.AuditLogger)
	listUC := usecase.NewListMembersUseCase(cfg.Repo)

	return &Module{
		cfg:      cfg,
		InviteUC: inviteUC,
		AcceptUC: acceptUC,
		RevokeUC: revokeUC,
		ListUC:   listUC,
		handler:  memberhttp.NewMembershipHandler(inviteUC, acceptUC, revokeUC, listUC),
	}
}

// RegisterRoutes mounts the membership HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	memberhttp.RegisterRoutes(mux, m.handler)
}
