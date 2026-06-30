import type { Metadata } from 'next'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import { Callout } from '@/components/docs/callout'
import { CodeBlock } from '@/components/docs/code-block'
import { DocLink, H2, H3, P, Section } from '@/components/docs/content'
import type { TocItem } from '@/components/docs/toc'

export const metadata: Metadata = {
  title: 'Building a Custom Module',
  description:
    'Learn how to consume existing kernel modules and how to build your own following the same conventions: domain, use cases, infrastructure, events, migrations.',
}

const toc: TocItem[] = [
  { id: 'using-modules', title: 'Using existing modules', level: 2 },
  { id: 'wiring-order', title: 'Wiring order', level: 3 },
  { id: 'injecting-services', title: 'Injecting shared services', level: 3 },
  { id: 'custom-module', title: 'Building a custom module', level: 2 },
  { id: 'structure', title: 'Directory structure', level: 3 },
  { id: 'domain', title: '1 · Domain', level: 3 },
  { id: 'usecase', title: '2 · Use cases', level: 3 },
  { id: 'infra', title: '3 · Infrastructure', level: 3 },
  { id: 'handler', title: '4 · HTTP handler', level: 3 },
  { id: 'migrations', title: '5 · Migrations', level: 3 },
  { id: 'events', title: '6 · Events (outbox)', level: 3 },
  { id: 'module-go', title: '7 · module.go', level: 3 },
  { id: 'checklist', title: 'Checklist', level: 2 },
]

// ── Using modules ─────────────────────────────────────────────────────────────

const fullWiringSnippet = `package main

import (
\t"log/slog"
\t"net/http"
\t"os"
\t"time"

\t"github.com/jackc/pgx/v5/pgxpool"

\t"github.com/agile-factory/kernel/platform/audit"
\tauditinfra "github.com/agile-factory/kernel/platform/audit/infrastructure"
\t"github.com/agile-factory/kernel/platform/auth"
\tauthinfra "github.com/agile-factory/kernel/platform/auth/infrastructure"
\t"github.com/agile-factory/kernel/platform/events"
\t"github.com/agile-factory/kernel/platform/membership"
\tmemberinfra "github.com/agile-factory/kernel/platform/membership/infrastructure"
\t"github.com/agile-factory/kernel/platform/migrator"
\t"github.com/agile-factory/kernel/platform/tenant"
\ttenantinfra "github.com/agile-factory/kernel/platform/tenant/infrastructure"
)

func main() {
\tctx := context.Background()
\tlog := slog.New(slog.NewJSONHandler(os.Stdout, nil))

\tpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
\tif err != nil {
\t\tlog.Error("db connect failed", "err", err)
\t\tos.Exit(1)
\t}

\t// 1. Run all module migrations in dependency order.
\tif err := migrator.Run(pool,
\t\taudit.Migrations,
\t\tauth.Migrations,
\t\ttenant.Migrations,
\t\tmembership.Migrations,
\t); err != nil {
\t\tlog.Error("migration failed", "err", err)
\t\tos.Exit(1)
\t}

\t// 2. Build the outbox event bus.
\tbus := events.NewBus(log)

\t// 3. Instantiate modules in dependency order.
\t//    audit first — its EventLogger is injected into other modules.
\tauditMod, err := audit.New(audit.Config{
\t\tDB:        pool,
\t\tRepo:      auditinfra.NewPostgresAuditRepository(pool),
\t\tPublisher: bus.Publisher(),
\t\tLogger:    log,
\t})
\tif err != nil {
\t\tlog.Error("audit init failed", "err", err)
\t\tos.Exit(1)
\t}
\tauditMod.Subscribe(bus) // persist audit.log_created events

\ttenantRepo := tenantinfra.NewPostgresTenantRepository(pool)
\tsettingsRepo := tenantinfra.NewPostgresTenantSettingsRepository(pool)
\tmemberRepo := memberinfra.NewPostgresMembershipRepository(pool)

\ttenantMod := tenant.New(tenant.Config{
\t\tPool:           pool,
\t\tRepo:           tenantRepo,
\t\tSettingsRepo:   settingsRepo,
\t\tMembershipRepo: memberRepo,
\t\tAuditLogger:    auditMod.EventLogger(),
\t\tLogger:         log,
\t})

\tauthMod := auth.New(auth.Config{
\t\tPool:             pool,
\t\tUserRepo:         authinfra.NewPostgresUserRepository(pool),
\t\tTenantRepo:       tenantRepo,
\t\tTenantSettingsRepo: settingsRepo,
\t\tMembershipRepo:   memberRepo,
\t\tInviteTokenRepo:  memberinfra.NewPostgresInviteTokenRepository(pool),
\t\tRefreshTokenRepo: authinfra.NewPostgresRefreshTokenRepository(pool),
\t\tEmailVerifRepo:   authinfra.NewPostgresEmailVerifRepository(pool),
\t\tAuditLogger:      auditMod.EventLogger(),
\t\tPublisher:        bus.Publisher(),
\t\tLogger:           log,
\t\tJWTSecret:        os.Getenv("JWT_SECRET"),
\t\tTokenLifespan:    15 * time.Minute,
\t})

\t// 4. Mount HTTP routes.
\tmux := http.NewServeMux()
\tauditMod.RegisterRoutes(mux)
\ttenantMod.RegisterRoutes(mux)
\tauthMod.RegisterRoutes(mux)

\t// 5. Start the outbox bus.
\tgo bus.Run(ctx)

\tlog.Info("listening", "addr", ":8080")
\tlog.Fatal(http.ListenAndServe(":8080", mux))
}`

const wiringOrderSnippet = `// Correct dependency order:
// 1. audit   — no deps on other kernel modules
// 2. tenant  — deps: memberRepo (to gate PATCH/DELETE)
// 3. auth    — deps: tenantRepo, settingsRepo, memberRepo, auditMod.EventLogger()
// 4. membership — deps: tenantRepo, userRepo
// 5. billing — deps: memberRepo (permission.Checker)
// 6. notification — deps: none (subscribe to bus)
// 7. user    — deps: memberRepo
// 8. storage — deps: none
// B2B2C:
// 9. end_customers  — deps: memberRepo
// 10. white_label   — deps: memberRepo
// 11. customer_auth — deps: none
// 12. customer_notifications — deps: brand resolver, customer resolver`

const sharedServicesSnippet = `// JWTService is built once inside auth.New and exported.
// Inject into middlewares or other modules — don't rebuild it.
jwtSvc := authMod.JWTService()

// permission.Checker is stateless; build once and share.
checker := permission.New(memberRepo)

// EventLogger from audit is the interface other modules consume.
// Never import audit use cases directly from another module.
auditLogger := auditMod.EventLogger()

// Inject both into a custom module.
myMod := mymodule.New(mymodule.Config{
\tJWT:         jwtSvc,
\tChecker:     checker,
\tAuditLogger: auditLogger,
\tPublisher:   bus.Publisher(),
})`

// ── Building a custom module ──────────────────────────────────────────────────

const structureSnippet = `platform/
└── invoices/               ← your module root
    ├── domain/
    │   ├── invoice.go      ← entity + value objects
    │   └── repository.go   ← repository interface
    ├── usecase/
    │   ├── create.go
    │   ├── get.go
    │   └── list.go
    ├── infrastructure/
    │   └── postgres_repository.go
    ├── handler/
    │   └── handler.go
    ├── migrations/
    │   ├── embed.go
    │   └── 0001_create_invoices.up.sql
    ├── events.go           ← event type constants
    └── module.go           ← Config + New() + RegisterRoutes`

const domainSnippet = `// platform/invoices/domain/invoice.go
package domain

import "time"

type InvoiceStatus string

const (
\tStatusDraft  InvoiceStatus = "draft"
\tStatusIssued InvoiceStatus = "issued"
\tStatusPaid   InvoiceStatus = "paid"
)

type Invoice struct {
\tID         string
\tTenantID   string
\tCustomerID string
\tAmount     int64  // cents
\tCurrency   string
\tStatus     InvoiceStatus
\tIssuedAt   *time.Time
\tCreatedAt  time.Time
}

// platform/invoices/domain/repository.go
package domain

import "context"

type InvoiceRepository interface {
\tCreate(ctx context.Context, inv *Invoice) error
\tFindByID(ctx context.Context, id, tenantID string) (*Invoice, error)
\tList(ctx context.Context, tenantID string, cursor string, limit int) ([]*Invoice, string, error)
}`

const usecaseSnippet = `// platform/invoices/usecase/create.go
package usecase

import (
\t"context"
\t"fmt"
\t"time"

\t"github.com/agile-factory/kernel/platform/events"
\t"github.com/agile-factory/kernel/platform/invoices/domain"
)

type CreateInvoiceInput struct {
\tTenantID   string
\tCustomerID string
\tAmount     int64
\tCurrency   string
}

type CreateInvoiceUseCase struct {
\trepo      domain.InvoiceRepository
\tpublisher events.Publisher
}

func NewCreateInvoiceUseCase(repo domain.InvoiceRepository, pub events.Publisher) *CreateInvoiceUseCase {
\treturn &CreateInvoiceUseCase{repo: repo, publisher: pub}
}

func (uc *CreateInvoiceUseCase) Execute(ctx context.Context, in CreateInvoiceInput) (*domain.Invoice, error) {
\tinv := &domain.Invoice{
\t\tID:         generateID(), // ulid or uuid
\t\tTenantID:   in.TenantID,
\t\tCustomerID: in.CustomerID,
\t\tAmount:     in.Amount,
\t\tCurrency:   in.Currency,
\t\tStatus:     domain.StatusDraft,
\t\tCreatedAt:  time.Now(),
\t}
\tif err := uc.repo.Create(ctx, inv); err != nil {
\t\treturn nil, fmt.Errorf("create invoice: %w", err)
\t}
\t// Publish via the transactional outbox.
\t_ = uc.publisher.Publish(ctx, "invoice.created", map[string]any{
\t\t"invoice_id": inv.ID,
\t\t"tenant_id":  inv.TenantID,
\t\t"amount":     inv.Amount,
\t\t"currency":   inv.Currency,
\t})
\treturn inv, nil
}`

const infraSnippet = `// platform/invoices/infrastructure/postgres_repository.go
package infrastructure

import (
\t"context"
\t"fmt"

\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/agile-factory/kernel/platform/invoices/domain"
)

type PostgresInvoiceRepository struct {
\tpool *pgxpool.Pool
}

func NewPostgresInvoiceRepository(pool *pgxpool.Pool) *PostgresInvoiceRepository {
\treturn &PostgresInvoiceRepository{pool: pool}
}

func (r *PostgresInvoiceRepository) Create(ctx context.Context, inv *domain.Invoice) error {
\t_, err := r.pool.Exec(ctx,
\t\t\`INSERT INTO invoices (id, tenant_id, customer_id, amount, currency, status, created_at)
\t\t VALUES ($1, $2, $3, $4, $5, $6, $7)\`,
\t\tinv.ID, inv.TenantID, inv.CustomerID, inv.Amount, inv.Currency, inv.Status, inv.CreatedAt,
\t)
\treturn err
}

func (r *PostgresInvoiceRepository) FindByID(ctx context.Context, id, tenantID string) (*domain.Invoice, error) {
\trow := r.pool.QueryRow(ctx,
\t\t"SELECT id, tenant_id, customer_id, amount, currency, status, created_at FROM invoices WHERE id=$1 AND tenant_id=$2",
\t\tid, tenantID,
\t)
\tinv := &domain.Invoice{}
\tif err := row.Scan(&inv.ID, &inv.TenantID, &inv.CustomerID, &inv.Amount, &inv.Currency, &inv.Status, &inv.CreatedAt); err != nil {
\t\treturn nil, fmt.Errorf("find invoice: %w", err)
\t}
\treturn inv, nil
}`

const handlerSnippet = `// platform/invoices/handler/handler.go
package handler

import (
\t"encoding/json"
\t"net/http"

\t"github.com/agile-factory/kernel/platform/invoices/usecase"
\t"github.com/agile-factory/kernel/platform/permission"
)

type Handler struct {
\tcreate  *usecase.CreateInvoiceUseCase
\tlist    *usecase.ListInvoicesUseCase
\tchecker permission.Checker
}

func New(create *usecase.CreateInvoiceUseCase, list *usecase.ListInvoicesUseCase, checker permission.Checker) *Handler {
\treturn &Handler{create: create, list: list, checker: checker}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
\tmux.HandleFunc("POST /api/v1/tenants/{tenantId}/invoices", h.handleCreate)
\tmux.HandleFunc("GET /api/v1/tenants/{tenantId}/invoices", h.handleList)
}

func (h *Handler) handleCreate(w http.ResponseWriter, r *http.Request) {
\ttenantID := r.PathValue("tenantId")

\t// Gate to tenant members using the shared permission.Checker.
\tif !h.checker.IsMember(r.Context(), tenantID) {
\t\thttp.Error(w, \`{"error":"forbidden"}\`, http.StatusForbidden)
\t\treturn
\t}

\tvar body struct {
\t\tCustomerID string \`json:"customer_id"\`
\t\tAmount     int64  \`json:"amount"\`
\t\tCurrency   string \`json:"currency"\`
\t}
\tif err := json.NewDecoder(r.Body).Decode(&body); err != nil {
\t\thttp.Error(w, \`{"error":"bad_request"}\`, http.StatusBadRequest)
\t\treturn
\t}

\tinv, err := h.create.Execute(r.Context(), usecase.CreateInvoiceInput{
\t\tTenantID:   tenantID,
\t\tCustomerID: body.CustomerID,
\t\tAmount:     body.Amount,
\t\tCurrency:   body.Currency,
\t})
\tif err != nil {
\t\thttp.Error(w, \`{"error":"internal"}\`, http.StatusInternalServerError)
\t\treturn
\t}

\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(http.StatusCreated)
\tjson.NewEncoder(w).Encode(inv)
}`

const migrationsSnippet = `-- platform/invoices/migrations/0001_create_invoices.up.sql
CREATE TABLE IF NOT EXISTS invoices (
    id          TEXT PRIMARY KEY,
    tenant_id   TEXT NOT NULL,
    customer_id TEXT NOT NULL,
    amount      BIGINT NOT NULL,
    currency    TEXT NOT NULL DEFAULT 'USD',
    status      TEXT NOT NULL DEFAULT 'draft',
    issued_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_invoices_tenant_id ON invoices (tenant_id);`

const embedSnippet = `// platform/invoices/migrations/embed.go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS`

const eventsSnippet = `// platform/invoices/events.go
package invoices

// Subscribe wires the module to consume events from other modules.
// Example: listen for billing.invoice_paid to auto-update invoice status.
func (m *Module) Subscribe(sub events.Subscriber) {
\tsub.Subscribe("billing.invoice_paid", func(ctx context.Context, payload []byte) error {
\t\tvar p struct {
\t\t\tInvoiceID string \`json:"invoice_id"\`
\t\t}
\t\tif err := json.Unmarshal(payload, &p); err != nil {
\t\t\treturn err
\t\t}
\t\treturn m.markPaid.Execute(ctx, p.InvoiceID)
\t})
}`

const moduleGoSnippet = `// platform/invoices/module.go
package invoices

import (
\t"embed"
\t"errors"
\t"log/slog"
\t"net/http"

\t"github.com/jackc/pgx/v5/pgxpool"

\t"github.com/agile-factory/kernel/platform/events"
\t"github.com/agile-factory/kernel/platform/invoices/domain"
\t"github.com/agile-factory/kernel/platform/invoices/handler"
\t"github.com/agile-factory/kernel/platform/invoices/infrastructure"
\t"github.com/agile-factory/kernel/platform/invoices/usecase"
\t"github.com/agile-factory/kernel/platform/migrator"
\t"github.com/agile-factory/kernel/platform/permission"
)

//go:embed migrations/*.sql
var sqlFiles embed.FS

// Migrations is the slice passed to migrator.Run.
var Migrations = migrator.FromFS(sqlFiles, "migrations")

// Config holds all dependencies — no globals, no init().
type Config struct {
\tPool      *pgxpool.Pool
\tRepo      domain.InvoiceRepository  // inject infra or a test stub
\tChecker   permission.Checker
\tPublisher events.Publisher
\tLogger    *slog.Logger
}

type Module struct {
\thandler *handler.Handler
\t// Expose use cases for cross-module injection if needed.
\tCreateUC *usecase.CreateInvoiceUseCase
}

func New(cfg Config) (*Module, error) {
\tvar errs []error
\tif cfg.Pool == nil      { errs = append(errs, errors.New("Pool is required")) }
\tif cfg.Repo == nil      { errs = append(errs, errors.New("Repo is required")) }
\tif cfg.Checker == nil   { errs = append(errs, errors.New("Checker is required")) }
\tif cfg.Publisher == nil { errs = append(errs, errors.New("Publisher is required")) }
\tif cfg.Logger == nil    { errs = append(errs, errors.New("Logger is required")) }
\tif len(errs) > 0        { return nil, errors.Join(errs...) }

\tcreateUC := usecase.NewCreateInvoiceUseCase(cfg.Repo, cfg.Publisher)
\tlistUC   := usecase.NewListInvoicesUseCase(cfg.Repo)

\treturn &Module{
\t\thandler:  handler.New(createUC, listUC, cfg.Checker),
\t\tCreateUC: createUC,
\t}, nil
}

func (m *Module) RegisterRoutes(mux *http.ServeMux) {
\tm.handler.RegisterRoutes(mux)
}`

const wireCustomSnippet = `// In main.go — wiring your custom module alongside kernel modules.
invoiceRepo := infrastructure.NewPostgresInvoiceRepository(pool)

invoiceMod, err := invoices.New(invoices.Config{
\tPool:      pool,
\tRepo:      invoiceRepo,
\tChecker:   permission.New(memberRepo),
\tPublisher: bus.Publisher(),
\tLogger:    log,
})
if err != nil {
\tlog.Error("invoices init failed", "err", err)
\tos.Exit(1)
}

// Subscribe BEFORE bus.Run.
invoiceMod.Subscribe(bus)
invoiceMod.RegisterRoutes(mux)

// Add your migrations to the run list.
migrator.Run(pool,
\taudit.Migrations,
\tauth.Migrations,
\ttenant.Migrations,
\tmembership.Migrations,
\tinvoices.Migrations,   // ← yours
)`

function Bullet({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-2.5">
      <span className="mt-2 size-1.5 shrink-0 rounded-full bg-primary" />
      <span>{children}</span>
    </li>
  )
}

function CheckItem({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-2.5">
      <span className="mt-0.5 text-primary">✓</span>
      <span>{children}</span>
    </li>
  )
}

export default function CustomModulePage() {
  return (
    <DocPage
      toc={toc}
      prev={{ title: 'Getting Started', href: '/docs/getting-started' }}
      next={{ title: 'Architecture (ADRs)', href: '/docs/architecture' }}
    >
      <PageHeader
        eyebrow="Guides"
        title="Building a Custom Module"
        description="How to wire existing kernel modules into your project and how to build your own following the same conventions."
      />

      {/* ── Using existing modules ─────────────────────────────────────────── */}

      <Section id="using-modules">
        <H2 id="using-modules">Using existing modules</H2>
        <P>
          Every kernel module follows the same pattern: a typed{' '}
          <code>Config</code> struct, a <code>New()</code> constructor that
          validates dependencies, <code>RegisterRoutes(mux)</code> for HTTP, and
          optionally <code>Subscribe(bus)</code> for event consumers. Wire them
          in{' '}
          <code>main.go</code> — no service locator, no global registry.
        </P>
        <CodeBlock code={fullWiringSnippet} lang="go" filename="main.go" />

        <H3 id="wiring-order">Wiring order</H3>
        <P>
          Build modules in dependency order. <code>audit</code> has no
          kernel-module dependencies, so it comes first. Its{' '}
          <code>EventLogger()</code> is then injected into every module that
          needs audit coverage.
        </P>
        <CodeBlock code={wiringOrderSnippet} lang="go" />

        <H3 id="injecting-services">Injecting shared services</H3>
        <P>
          Several kernel objects are built once and shared across modules.
          Never re-create them — share the instance.
        </P>
        <CodeBlock code={sharedServicesSnippet} lang="go" />
        <Callout variant="tip">
          <p>
            <code>permission.Checker</code> and <code>audit.EventLogger</code>{' '}
            are interfaces, not concrete types. Your custom module can depend on
            the interface only — no import of the concrete kernel package.
          </p>
        </Callout>
      </Section>

      {/* ── Building a custom module ───────────────────────────────────────── */}

      <Section id="custom-module">
        <H2 id="custom-module">Building a custom module</H2>
        <P>
          Custom modules live alongside kernel modules and follow the same
          layout. The example below builds a minimal <code>invoices</code>{' '}
          module from scratch.
        </P>

        <H3 id="structure">Directory structure</H3>
        <CodeBlock code={structureSnippet} lang="bash" />
        <Callout variant="warning" title="One package per layer">
          <p>
            Each subdirectory is its own Go package.{' '}
            <code>domain</code> must not import <code>usecase</code> or{' '}
            <code>infrastructure</code>. Dependency arrows always point inward:
            handler → usecase → domain.
          </p>
        </Callout>

        <H3 id="domain">1 · Domain</H3>
        <P>
          Define your entity and the repository interface. No framework
          imports — pure Go structs and interfaces.
        </P>
        <CodeBlock code={domainSnippet} lang="go" filename="domain/invoice.go" />

        <H3 id="usecase">2 · Use cases</H3>
        <P>
          Each use case is a struct with an <code>Execute</code> method.
          Dependencies are injected via the constructor — no global state,
          no <code>init()</code>. Publish outbox events at the end of a
          successful operation.
        </P>
        <CodeBlock code={usecaseSnippet} lang="go" filename="usecase/create.go" />

        <H3 id="infra">3 · Infrastructure</H3>
        <P>
          Implement the domain repository interface against Postgres. The
          implementation is the only place that imports <code>pgxpool</code>.
        </P>
        <CodeBlock code={infraSnippet} lang="go" filename="infrastructure/postgres_repository.go" />

        <H3 id="handler">4 · HTTP handler</H3>
        <P>
          The handler decodes JSON, gates access via{' '}
          <code>permission.Checker</code>, calls the use case, and encodes the
          response. It has no business logic.
        </P>
        <CodeBlock code={handlerSnippet} lang="go" filename="handler/handler.go" />

        <H3 id="migrations">5 · Migrations</H3>
        <P>
          Embed SQL files using <code>embed.FS</code> and expose a{' '}
          <code>Migrations</code> variable. The <code>migrator</code> package
          tracks applied files so re-running is idempotent.
        </P>
        <CodeBlock
          code={migrationsSnippet}
          lang="sql"
          filename="migrations/0001_create_invoices.up.sql"
        />
        <CodeBlock code={embedSnippet} lang="go" filename="migrations/embed.go" />
        <Callout variant="tip">
          <p>
            Every <code>.up.sql</code> file must have a paired{' '}
            <code>.down.sql</code> rollback. The CI check fails otherwise.
          </p>
        </Callout>

        <H3 id="events">6 · Events (outbox)</H3>
        <P>
          Subscribe to events from other modules in a{' '}
          <code>Subscribe(sub events.Subscriber)</code> method on your module.
          Use{' '}
          <code>publisher.Publish</code> in your use cases to emit events into
          the outbox. Both are at-least-once — handlers must be idempotent.
        </P>
        <CodeBlock code={eventsSnippet} lang="go" filename="events.go" />

        <H3 id="module-go">7 · module.go</H3>
        <P>
          The root file exposes <code>Config</code>, <code>New()</code>,{' '}
          <code>Migrations</code>, and <code>RegisterRoutes</code>. Validate
          every required field in <code>New()</code> using{' '}
          <code>errors.Join</code> — fail fast at startup.
        </P>
        <CodeBlock code={moduleGoSnippet} lang="go" filename="module.go" />

        <P>Wire it into <code>main.go</code> like any kernel module:</P>
        <CodeBlock code={wireCustomSnippet} lang="go" filename="main.go" />
      </Section>

      {/* ── Checklist ─────────────────────────────────────────────────────── */}

      <Section id="checklist">
        <H2 id="checklist">Checklist</H2>
        <P>Before shipping a custom module, verify:</P>
        <ul className="mt-3 space-y-2 text-sm text-muted-foreground">
          <CheckItem>
            <code>domain/</code> has no imports from <code>usecase</code>,{' '}
            <code>infrastructure</code>, or any kernel package other than shared
            value types.
          </CheckItem>
          <CheckItem>
            <code>New()</code> validates every required Config field and returns
            a descriptive error listing all missing deps.
          </CheckItem>
          <CheckItem>
            Every <code>.up.sql</code> has a paired <code>.down.sql</code>.
          </CheckItem>
          <CheckItem>
            Event handlers are idempotent — the outbox delivers at-least-once.
          </CheckItem>
          <CheckItem>
            <code>Subscribe(bus)</code> is called before <code>bus.Run(ctx)</code>.
          </CheckItem>
          <CheckItem>
            HTTP handlers use <code>permission.Checker</code> — never inspect
            JWT claims directly.
          </CheckItem>
          <CheckItem>
            Email templates use <code>html/template</code>, never{' '}
            <code>text/template</code>.
          </CheckItem>
          <CheckItem>No global variables, no <code>init()</code> side effects.</CheckItem>
        </ul>
      </Section>
    </DocPage>
  )
}
