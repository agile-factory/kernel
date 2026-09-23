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

	deliveryhttp "github.com/agile-factory/kernel/delivery/http"
	"github.com/agile-factory/kernel/delivery/http/middleware"
	"github.com/agile-factory/kernel/kernel/config"
	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/health"
	"github.com/agile-factory/kernel/kernel/logger"
	"github.com/agile-factory/kernel/kernel/metrics"
	"github.com/agile-factory/kernel/kernel/migrator"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/kernel/server"
	"github.com/agile-factory/kernel/platform/audit"
	auditinfra "github.com/agile-factory/kernel/platform/audit/infrastructure"
	"github.com/agile-factory/kernel/platform/auth"
	authinfra "github.com/agile-factory/kernel/platform/auth/infrastructure"
	"github.com/agile-factory/kernel/platform/membership"
	memberinfra "github.com/agile-factory/kernel/platform/membership/infrastructure"
	"github.com/agile-factory/kernel/platform/notification"
	notifinfra "github.com/agile-factory/kernel/platform/notification/infrastructure"
	"github.com/agile-factory/kernel/platform/tenant"
	tenantinfra "github.com/agile-factory/kernel/platform/tenant/infrastructure"
	"github.com/agile-factory/kernel/platform/billing"
	"github.com/agile-factory/kernel/platform/billing/adapter"
	billinginfra "github.com/agile-factory/kernel/platform/billing/infrastructure"
	"github.com/agile-factory/kernel/platform/user"
	userinfra "github.com/agile-factory/kernel/platform/user/infrastructure"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	if err := cfg.Validate(); err != nil {
		slog.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}
	if cfg.JWTSecretGenerated {
		slog.Warn("JWT_SECRET is not set: using a random per-process secret (development only; tokens will not survive a restart)")
	}

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
		{Name: "notification", FS: notification.Migrations, Path: "migrations"},
		{Name: "user", FS: user.Migrations, Path: "migrations"},
		{Name: "billing", FS: billing.Migrations, Path: "migrations"},
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
	tenantSettingsRepo := tenantinfra.NewPostgresTenantSettingsRepository(db)
	memberRepo := memberinfra.NewPostgresMembershipRepository(db)
	userRepo := authinfra.NewPostgresUserRepository(db)
	refreshTokenRepo := authinfra.NewPostgresRefreshTokenRepository(db)
	emailVerifRepo := authinfra.NewPostgresEmailVerificationTokenRepository(db)
	inviteTokenRepo := memberinfra.NewPostgresInviteTokenRepository(db)
	notifLogRepo := notifinfra.NewPostgresNotificationLogRepository(db)

	// ── Modules ──────────────────────────────────────────────────────────────
	// The audit Module uses the outbox at-least-once pattern (RFC-0040 /
	// ADR-0020). LogEventUseCase publishes to event_outbox; the consumer
	// is registered via auditMod.Subscribe(bus) below.
	auditMod, err := audit.New(audit.Config{
		DB:             db,
		Repo:           auditRepo,
		MembershipRepo: memberRepo,
		Publisher:      bus.Publisher(),
		Logger:         log,
	})
	if err != nil {
		slog.Error("Failed to initialise audit module", "error", err)
		os.Exit(1)
	}

	tenantMod := tenant.New(tenant.Config{
		Pool:           db,
		Repo:           tenantRepo,
		SettingsRepo:   tenantSettingsRepo,
		MembershipRepo: memberRepo,
		AuditLogger:    auditMod.EventLogger(),
		Logger:         log,
	})

	memberMod := membership.New(membership.Config{
		Pool:            db,
		Repo:            memberRepo,
		InviteTokenRepo: inviteTokenRepo,
		TenantRepo:      tenantRepo,
		UserLookup:      userRepo,
		AuditLogger:     auditMod.EventLogger(),
		Publisher:       bus.Publisher(),
		Logger:          log,
	})

	authMod := auth.New(auth.Config{
		Pool:               db,
		UserRepo:           userRepo,
		TenantRepo:         tenantRepo,
		TenantSettingsRepo: tenantSettingsRepo,
		MembershipRepo:     memberRepo,
		InviteTokenRepo:    inviteTokenRepo,
		RefreshTokenRepo:   refreshTokenRepo,
		EmailVerifRepo:     emailVerifRepo,
		MembershipListUC:   memberMod.ListUC,
		AuditLogger:        auditMod.EventLogger(),
		Publisher:          bus.Publisher(),
		Logger:             log,
		JWTSecret:          cfg.JWTSecret,
		TokenLifespan:      cfg.TokenLifespan,
	})

	// Notification module — subscribes to outbox events emitted by auth /
	// membership and renders + sends emails. v0.1.0 ships a ConsoleSender
	// fallback (logs the rendered email) so the binary runs without an
	// SMTP server configured.
	notifMod, err := notification.New(notification.Config{
		LogRepo:     notifLogRepo,
		EmailSender: notification.NewConsoleSender(log),
		Logger:      log,
		From:        "no-reply@agilefactory.local",
		BaseURL:     "http://localhost:" + cfg.Port,
	})
	if err != nil {
		slog.Error("Failed to initialise notification module", "error", err)
		os.Exit(1)
	}
	auditMod.Subscribe(bus)  // audit.log_created → audit_logs (at-least-once, RFC-0040)
	notifMod.Subscribe(bus)  // register handlers before bus.Run starts.

	// ── User profile module (P1-2) ───────────────────────────────────────────
	profileRepo := userinfra.NewPostgresUserProfileRepository(db)
	checker := permission.New(memberRepo)
	userMod, err := user.New(user.Config{
		Repo:    profileRepo,
		Checker: checker,
		Members: memberRepo,
		Logger:  log,
	})
	if err != nil {
		slog.Error("Failed to initialise user module", "error", err)
		os.Exit(1)
	}
	userMod.Subscribe(bus) // creates profile on auth.signup_owner events

	// ── Observability (P1-3) ─────────────────────────────────────────────────
	// Prometheus metrics registry (isolated, not the global default).
	promReg := prometheus.NewRegistry()
	rec := metrics.NewPrometheusRecorder(promReg)

	// Health / readiness handlers (replaces the inline GET /health stub).
	healthChecker := health.New(health.Config{
		Version:  "0.2.0",
		DBPinger: db,
	})

	// ── HTTP wiring ──────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Health & readiness (public — before InternalOnly middleware)
	mux.HandleFunc("GET /health", healthChecker.LivenessHandler)
	mux.HandleFunc("GET /ready", healthChecker.ReadinessHandler)

	// Prometheus metrics — internal only (loopback CIDR guard)
	mux.Handle("GET /metrics", metrics.InternalOnlyMiddleware(
		promhttp.HandlerFor(promReg, promhttp.HandlerOpts{}),
	))

	auditMod.RegisterRoutes(mux)
	tenantMod.RegisterRoutes(mux)
	memberMod.RegisterRoutes(mux)
	authMod.RegisterRoutes(mux)
	notifMod.RegisterRoutes(mux)
	userMod.RegisterRoutes(mux)

	// ── Billing module (P1-4) ────────────────────────────────────────────────
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	stripeAdapter := adapter.NewStripeAdapter(adapter.StripeConfig{
		SecretKey:     stripeKey,
		WebhookSecret: stripeWebhookSecret,
	})
	billingMod, err := billing.New(billing.Config{
		SubRepo:   billinginfra.NewPostgresSubscriptionRepository(db),
		PlanRepo:  billinginfra.NewPostgresPlanRepository(db),
		Processor: stripeAdapter,
		Checker:   checker,
		Logger:    log,
	})
	if err != nil {
		slog.Error("Failed to initialise billing module", "error", err)
		os.Exit(1)
	}
	billingMod.RegisterRoutes(mux)

	// Authentication: every module mounts its routes on the same mux, so the
	// token is validated globally in optional mode — public routes (signup,
	// login, health…) work without a header, protected handlers answer 401
	// on their own when no identity is in the context, and a header that is
	// present but invalid is rejected here. Tenant resolution (X-Tenant-ID,
	// ?tenantId) is NOT trusted by itself: handlers check membership through
	// permission.Checker.
	var handler http.Handler = mux
	handler = middleware.TenantContextOptional(handler)
	handler = middleware.OptionalAuth(middleware.NewHS256Validator(cfg.JWTSecret))(handler)

	// Wrap with the Prometheus HTTP middleware so all routes are measured.
	instrumentedMux := metrics.HTTPMiddleware(rec)(handler)

	srv := server.New(cfg, deliveryhttp.ApplyGlobalMiddlewares(instrumentedMux, log))
	healthChecker.MarkReady()

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
