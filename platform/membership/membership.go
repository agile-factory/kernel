// Package membership is the membership-lifecycle platform. It exposes a
// self-contained Module that wires the membership + invite_token repositories,
// use cases and HTTP handlers. Per RFC-0018 §6, the invite flow now creates
// an invite_token (consumed by auth.AcceptInvite) rather than a PENDING
// membership row.
package membership

import (
	"embed"
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/audit"
	memberhttp "github.com/agile-factory/kernel/platform/membership/delivery/http"
	"github.com/agile-factory/kernel/platform/membership/domain"
	"github.com/agile-factory/kernel/platform/membership/usecase"
	tenantdomain "github.com/agile-factory/kernel/platform/tenant/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the SQL files this platform owns. Consumers compose them
// via migrator.Run alongside their own modules' migrations.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Repository re-exports the membership repository contract for consumers.
type Repository = domain.MembershipRepository

// InviteTokenRepository re-exports the invite-token repository contract.
type InviteTokenRepository = domain.InviteTokenRepository

// Config captures the dependencies required by the membership Module.
type Config struct {
	Pool             *pgxpool.Pool
	Repo             Repository
	InviteTokenRepo  InviteTokenRepository
	TenantRepo       tenantdomain.TenantRepository
	UserLookup       usecase.UserLookup // typically the auth UserRepo, accessed through this slice
	AuditLogger      audit.EventLogger
	Publisher        events.Publisher // nil-safe; member.invited event skipped when nil
	Logger           *slog.Logger
}

// Module is the self-contained membership platform.
type Module struct {
	cfg Config

	InviteUC *usecase.InviteMemberUseCase
	RevokeUC *usecase.RevokeMemberUseCase
	ListUC   *usecase.ListMembersUseCase

	handler *memberhttp.MembershipHandler
}

// New wires the membership module's internal components.
func New(cfg Config) *Module {
	inviteUC := usecase.NewInviteMemberUseCase(
		cfg.Pool,
		cfg.Repo,
		cfg.TenantRepo,
		cfg.InviteTokenRepo,
		cfg.UserLookup,
		cfg.Publisher,
		cfg.AuditLogger,
	)
	revokeUC := usecase.NewRevokeMemberUseCase(cfg.Repo, cfg.AuditLogger)
	listUC := usecase.NewListMembersUseCase(cfg.Repo)

	return &Module{
		cfg:      cfg,
		InviteUC: inviteUC,
		RevokeUC: revokeUC,
		ListUC:   listUC,
		handler:  memberhttp.NewMembershipHandler(inviteUC, revokeUC, listUC),
	}
}

// RegisterRoutes mounts the membership HTTP endpoints on the provided mux.
func (m *Module) RegisterRoutes(mux *http.ServeMux) {
	memberhttp.RegisterRoutes(mux, m.handler)
}
