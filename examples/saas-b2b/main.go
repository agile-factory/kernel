// saas-b2b is the minimal example showing how to consume the kernel platform
// packages from a consumer SaaS repo. It composes the audit, tenant,
// membership and auth modules explicitly — no global state, no init().
//
// Real consumer repos would mirror this `main.go`, adding their own business
// modules alongside the kernel modules.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	deliveryhttp "github.com/AgileFactory/kernel/delivery/http"
	"github.com/AgileFactory/kernel/kernel/config"
	"github.com/AgileFactory/kernel/kernel/events"
	"github.com/AgileFactory/kernel/kernel/logger"
	"github.com/AgileFactory/kernel/kernel/migrator"
	"github.com/AgileFactory/kernel/kernel/server"
	"github.com/AgileFactory/kernel/platform/audit"
	auditinfra "github.com/AgileFactory/kernel/platform/audit/infrastructure"
	"github.com/AgileFactory/kernel/platform/auth"
	authinfra "github.com/AgileFactory/kernel/platform/auth/infrastructure"
	"github.com/AgileFactory/kernel/platform/membership"
	memberinfra "github.com/AgileFactory/kernel/platform/membership/infrastructure"
	"github.com/AgileFactory/kernel/platform/tenant"
	tenantinfra "github.com/AgileFactory/kernel/platform/tenant/infrastructure"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	log, closeLog, err := logger.New(logger.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.Environment,
		Dir:         cfg.LogDir,
	})
	if err != nil {
		log = logger.NewSimple(cfg.LogLevel, cfg.Environment)
		log.Error("Failed to initialize file logger, falling back to stdout", "error", err)
	} else {
		defer closeLog()
	}
	slog.SetDefault(log)

	slog.Info("Starting application", "env", cfg.Environment, "port", cfg.Port)

	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.Ping(context.Background()); err != nil {
		panic(fmt.Sprintf("Failed to connect to database: %v", err))
	}
	slog.Info("Successfully connected to database")

	// kernel/events Migrations come first: the outbox table must exist before
	// any platform use case can publish into it. Per-platform tables follow
	// in dependency order (tenant → auth → membership → audit).
	if err := migrator.Run(cfg.DatabaseURL, []migrator.Source{
		{Name: "events", FS: events.Migrations, Path: "migrations"},
		{Name: "tenant", FS: tenant.Migrations, Path: "migrations"},
		{Name: "auth", FS: auth.Migrations, Path: "migrations"},
		{Name: "membership", FS: membership.Migrations, Path: "migrations"},
		{Name: "audit", FS: audit.Migrations, Path: "migrations"},
	}); err != nil {
		slog.Error("Failed to run database migrations", "error", err)
		os.Exit(1)
	}

	// ── Events Bus ───────────────────────────────────────────────────────────
	// Built before the platforms so its Publisher can be injected. The
	// dispatcher is started in a goroutine below — handlers are subscribed
	// by platforms that want at-least-once delivery (P1-3 onwards).
	bus, err := events.New(events.Config{Pool: db, Logger: log})
	if err != nil {
		slog.Error("Failed to initialise events bus", "error", err)
		os.Exit(1)
	}

	// ── Repositories ─────────────────────────────────────────────────────────
	// Repos live outside the modules so platforms can share a single repo
	// instance (e.g. audit's HTTP handler needs the membership repo for
	// perm checks while membership's use cases need audit.EventLogger —
	// constructing repos here breaks that cycle cleanly).
	auditRepo := auditinfra.NewPostgresAuditRepository(db)
	tenantRepo := tenantinfra.NewPostgresTenantRepository(db)
	memberRepo := memberinfra.NewPostgresMembershipRepository(db)
	userRepo := authinfra.NewPostgresUserRepository(db)

	// ── Modules ──────────────────────────────────────────────────────────────
	// The audit Module currently uses best-effort direct injection (ADR-0009
	// — DEPRECATED). The events Bus is wired in but not yet consumed by
	// audit; that migration happens in P1-3 and will replace EventLogger()
	// with an outbox-backed handler.
	auditMod := audit.New(audit.Config{
		Repo:           auditRepo,
		MembershipRepo: memberRepo,
		Logger:         log,
	})

	tenantMod := tenant.New(tenant.Config{
		Repo:        tenantRepo,
		AuditLogger: auditMod.EventLogger(),
		Logger:      log,
	})

	memberMod := membership.New(membership.Config{
		Repo:        memberRepo,
		TenantRepo:  tenantRepo,
		AuditLogger: auditMod.EventLogger(),
		Logger:      log,
	})

	authMod := auth.New(auth.Config{
		UserRepo:         userRepo,
		TenantRepo:       tenantRepo,
		MembershipRepo:   memberRepo,
		MembershipListUC: memberMod.ListUC,
		AuditLogger:      auditMod.EventLogger(),
		Logger:           log,
		JWTSecret:        cfg.JWTSecret,
		TokenLifespan:    cfg.TokenLifespan,
	})

	_ = bus.Publisher() // silence unused until consumers wire it in P1-3

	// ── HTTP wiring ──────────────────────────────────────────────────────────
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	auditMod.RegisterRoutes(mux)
	tenantMod.RegisterRoutes(mux)
	memberMod.RegisterRoutes(mux)
	authMod.RegisterRoutes(mux)

	srv := server.New(cfg, deliveryhttp.ApplyGlobalMiddlewares(mux, log))

	// Run lifecycle: a cancellable parent context drives the dispatcher and
	// is cancelled on SIGINT/SIGTERM so background loops shut down cleanly
	// before the HTTP server is drained.
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()

	busDone := make(chan struct{})
	go func() {
		defer close(busDone)
		if err := bus.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("Events bus dispatcher exited", "error", err)
		}
	}()

	go func() {
		if err := srv.Start(); err != nil {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	// Stop the dispatcher and wait for it to drain.
	cancelRun()
	<-busDone

	slog.Info("Server exiting")
}
