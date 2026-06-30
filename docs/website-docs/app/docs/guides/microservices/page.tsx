import type { Metadata } from 'next'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import { Callout } from '@/components/docs/callout'
import { CodeBlock } from '@/components/docs/code-block'
import { H2, H3, P, Section } from '@/components/docs/content'
import type { TocItem } from '@/components/docs/toc'

export const metadata: Metadata = {
  title: 'Architecture microservices',
  description:
    'Comment déployer le kernel AgileFactory en architecture microservices : découpage par domaine, communication inter-services via outbox, et déploiement Docker.',
}

const toc: TocItem[] = [
  { id: 'architecture-choice', title: 'Monolith-first ou microservices ?', level: 2 },
  { id: 'service-mapping', title: 'Découpage en services', level: 2 },
  { id: 'core-service', title: 'Service core', level: 3 },
  { id: 'billing-service', title: 'Service billing', level: 3 },
  { id: 'portal-service', title: 'Service portal B2B2C', level: 3 },
  { id: 'inter-service', title: 'Communication inter-services via events', level: 2 },
  { id: 'deployment', title: 'Déploiement', level: 2 },
]

// ── Snippets ──────────────────────────────────────────────────────────────────

const coreMainSnippet = `// cmd/core/main.go — service core : auth + tenant + membership + audit
package main

import (
\t"context"
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

\t// Migrations : uniquement les modules de ce service.
\tif err := migrator.Run(pool,
\t\taudit.Migrations,
\t\tauth.Migrations,
\t\ttenant.Migrations,
\t\tmembership.Migrations,
\t); err != nil {
\t\tlog.Error("migration failed", "err", err)
\t\tos.Exit(1)
\t}

\tbus := events.NewBus(log)

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
\tauditMod.Subscribe(bus)

\ttenantRepo   := tenantinfra.NewPostgresTenantRepository(pool)
\tsettingsRepo := tenantinfra.NewPostgresTenantSettingsRepository(pool)
\tmemberRepo   := memberinfra.NewPostgresMembershipRepository(pool)

\ttenantMod := tenant.New(tenant.Config{
\t\tPool:           pool,
\t\tRepo:           tenantRepo,
\t\tSettingsRepo:   settingsRepo,
\t\tMembershipRepo: memberRepo,
\t\tAuditLogger:    auditMod.EventLogger(),
\t\tLogger:         log,
\t})

\tauthMod := auth.New(auth.Config{
\t\tPool:               pool,
\t\tUserRepo:           authinfra.NewPostgresUserRepository(pool),
\t\tTenantRepo:         tenantRepo,
\t\tTenantSettingsRepo: settingsRepo,
\t\tMembershipRepo:     memberRepo,
\t\tInviteTokenRepo:    memberinfra.NewPostgresInviteTokenRepository(pool),
\t\tRefreshTokenRepo:   authinfra.NewPostgresRefreshTokenRepository(pool),
\t\tEmailVerifRepo:     authinfra.NewPostgresEmailVerifRepository(pool),
\t\tAuditLogger:        auditMod.EventLogger(),
\t\tPublisher:          bus.Publisher(),
\t\tLogger:             log,
\t\tJWTSecret:          os.Getenv("JWT_SECRET"),
\t\tTokenLifespan:      15 * time.Minute,
\t})

\tmemberMod := membership.New(membership.Config{
\t\tPool:       pool,
\t\tRepo:       memberRepo,
\t\tTenantRepo: tenantRepo,
\t\tUserRepo:   authinfra.NewPostgresUserRepository(pool),
\t\tPublisher:  bus.Publisher(),
\t\tLogger:     log,
\t})

\tmux := http.NewServeMux()
\tauditMod.RegisterRoutes(mux)
\ttenantMod.RegisterRoutes(mux)
\tauthMod.RegisterRoutes(mux)
\tmemberMod.RegisterRoutes(mux)

\tgo bus.Run(ctx)

\tlog.Info("core service listening", "addr", ":8080")
\tif err := http.ListenAndServe(":8080", mux); err != nil {
\t\tlog.Error("server error", "err", err)
\t\tos.Exit(1)
\t}
}`

const billingMainSnippet = `// cmd/billing/main.go — service billing
package main

import (
\t"context"
\t"log/slog"
\t"net/http"
\t"os"

\t"github.com/jackc/pgx/v5/pgxpool"

\t"github.com/agile-factory/kernel/platform/billing"
\tbillinginfra "github.com/agile-factory/kernel/platform/billing/infrastructure"
\t"github.com/agile-factory/kernel/platform/events"
\t"github.com/agile-factory/kernel/platform/membership"
\tmemberinfra "github.com/agile-factory/kernel/platform/membership/infrastructure"
\t"github.com/agile-factory/kernel/platform/migrator"
\t"github.com/agile-factory/kernel/platform/permission"
)

func main() {
\tctx := context.Background()
\tlog := slog.New(slog.NewJSONHandler(os.Stdout, nil))

\tpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
\tif err != nil {
\t\tlog.Error("db connect failed", "err", err)
\t\tos.Exit(1)
\t}

\t// Billing apporte ses propres migrations.
\t// membership.Migrations est déjà appliqué par le service core
\t// mais migrator.Run est idempotent — le relancer ici est sûr.
\tif err := migrator.Run(pool,
\t\tmembership.Migrations,
\t\tbilling.Migrations,
\t); err != nil {
\t\tlog.Error("migration failed", "err", err)
\t\tos.Exit(1)
\t}

\tbus := events.NewBus(log)

\tmemberRepo := memberinfra.NewPostgresMembershipRepository(pool)
\tchecker    := permission.New(memberRepo)

\tbillingMod, err := billing.New(billing.Config{
\t\tPool:           pool,
\t\tRepo:           billinginfra.NewPostgresBillingRepository(pool),
\t\tChecker:        checker,
\t\tStripeKey:      os.Getenv("STRIPE_SECRET_KEY"),
\t\tStripeWebhook:  os.Getenv("STRIPE_WEBHOOK_SECRET"),
\t\tPublisher:      bus.Publisher(),
\t\tLogger:         log,
\t})
\tif err != nil {
\t\tlog.Error("billing init failed", "err", err)
\t\tos.Exit(1)
\t}
\tbillingMod.Subscribe(bus)

\tmux := http.NewServeMux()
\tbillingMod.RegisterRoutes(mux)

\tgo bus.Run(ctx)

\tlog.Info("billing service listening", "addr", ":8081")
\tif err := http.ListenAndServe(":8081", mux); err != nil {
\t\tlog.Error("server error", "err", err)
\t\tos.Exit(1)
\t}
}`

const portalMainSnippet = `// cmd/portal/main.go — service portal B2B2C
// Modules : end_customers + customer_auth + white_label + customer_notifications
package main

import (
\t"context"
\t"log/slog"
\t"net/http"
\t"os"

\t"github.com/jackc/pgx/v5/pgxpool"

\t"github.com/agile-factory/kernel/platform/customer_auth"
\tcustomerauthinfra "github.com/agile-factory/kernel/platform/customer_auth/infrastructure"
\t"github.com/agile-factory/kernel/platform/customer_notifications"
\t"github.com/agile-factory/kernel/platform/end_customers"
\tendcustinfra "github.com/agile-factory/kernel/platform/end_customers/infrastructure"
\t"github.com/agile-factory/kernel/platform/events"
\t"github.com/agile-factory/kernel/platform/membership"
\tmemberinfra "github.com/agile-factory/kernel/platform/membership/infrastructure"
\t"github.com/agile-factory/kernel/platform/migrator"
\t"github.com/agile-factory/kernel/platform/permission"
\t"github.com/agile-factory/kernel/platform/white_label"
\twlinfra "github.com/agile-factory/kernel/platform/white_label/infrastructure"
)

func main() {
\tctx := context.Background()
\tlog := slog.New(slog.NewJSONHandler(os.Stdout, nil))

\tpool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
\tif err != nil {
\t\tlog.Error("db connect failed", "err", err)
\t\tos.Exit(1)
\t}

\tif err := migrator.Run(pool,
\t\tmembership.Migrations,
\t\tend_customers.Migrations,
\t\tcustomer_auth.Migrations,
\t\twhite_label.Migrations,
\t\tcustomer_notifications.Migrations,
\t); err != nil {
\t\tlog.Error("migration failed", "err", err)
\t\tos.Exit(1)
\t}

\tbus := events.NewBus(log)

\tmemberRepo  := memberinfra.NewPostgresMembershipRepository(pool)
\tchecker     := permission.New(memberRepo)
\tbrandRepo   := wlinfra.NewPostgresBrandRepository(pool)
\tendCustRepo := endcustinfra.NewPostgresEndCustomerRepository(pool)

\twlMod := white_label.New(white_label.Config{
\t\tPool:      pool,
\t\tRepo:      brandRepo,
\t\tChecker:   checker,
\t\tPublisher: bus.Publisher(),
\t\tLogger:    log,
\t})

\tendCustMod, err := end_customers.New(end_customers.Config{
\t\tPool:      pool,
\t\tRepo:      endCustRepo,
\t\tChecker:   checker,
\t\tPublisher: bus.Publisher(),
\t\tLogger:    log,
\t})
\tif err != nil {
\t\tlog.Error("end_customers init failed", "err", err)
\t\tos.Exit(1)
\t}

\tcustAuthMod := customer_auth.New(customer_auth.Config{
\t\tPool:          pool,
\t\tRepo:          customerauthinfra.NewPostgresCustomerAuthRepository(pool),
\t\tCustomerRepo:  endCustRepo,
\t\tBrandResolver: wlMod.BrandResolver(),
\t\tJWTSecret:     os.Getenv("CUSTOMER_JWT_SECRET"),
\t\tPublisher:     bus.Publisher(),
\t\tLogger:        log,
\t})

\tcustNotifMod := customer_notifications.New(customer_notifications.Config{
\t\tPool:          pool,
\t\tBrandResolver: wlMod.BrandResolver(),
\t\tCustomerRepo:  endCustRepo,
\t\tLogger:        log,
\t})
\tcustNotifMod.Subscribe(bus)

\tmux := http.NewServeMux()
\twlMod.RegisterRoutes(mux)
\tendCustMod.RegisterRoutes(mux)
\tcustAuthMod.RegisterRoutes(mux)
\tcustNotifMod.RegisterRoutes(mux)

\tgo bus.Run(ctx)

\tlog.Info("portal service listening", "addr", ":8082")
\tif err := http.ListenAndServe(":8082", mux); err != nil {
\t\tlog.Error("server error", "err", err)
\t\tos.Exit(1)
\t}
}`

const interServiceSnippet = `// ── Publier un event depuis le service billing ──────────────────────────────
// Dans un use case billing, après une opération réussie :
_ = publisher.Publish(ctx, "billing.subscription_activated", map[string]any{
\t"tenant_id":       tenantID,
\t"subscription_id": sub.ID,
\t"plan":            sub.Plan,
})
// L'event est écrit dans la table outbox Postgres en même temps que
// la transaction métier (at-least-once, livraison garantie).

// ── Consommer cet event dans le service portal ───────────────────────────────
// Dans portal/main.go, après avoir instancié les modules :
bus.Subscribe("billing.subscription_activated", func(ctx context.Context, payload []byte) error {
\tvar p struct {
\t\tTenantID string \`json:"tenant_id"\`
\t\tPlan     string \`json:"plan"\`
\t}
\tif err := json.Unmarshal(payload, &p); err != nil {
\t\treturn err
\t}
\t// Activer les features B2B2C selon le plan souscrit.
\treturn endCustMod.ActivatePlan(ctx, p.TenantID, p.Plan)
})

// IMPORTANT : les deux services doivent partager la MÊME base Postgres
// pour que l'outbox soit visible cross-service.
// La table \`outbox_events\` est créée par audit.Migrations (service core).
// Les autres services relisent la même table via leur propre bus.Run().`

const dockerComposeSnippet = `# docker-compose.yml
version: '3.9'

services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: kernel
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: kernel
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U kernel"]
      interval: 5s
      retries: 5

  core:
    build:
      context: .
      dockerfile: cmd/core/Dockerfile
    environment:
      DATABASE_URL: postgres://kernel:secret@postgres:5432/kernel
      JWT_SECRET: \${JWT_SECRET}
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy

  billing:
    build:
      context: .
      dockerfile: cmd/billing/Dockerfile
    environment:
      DATABASE_URL: postgres://kernel:secret@postgres:5432/kernel
      STRIPE_SECRET_KEY: \${STRIPE_SECRET_KEY}
      STRIPE_WEBHOOK_SECRET: \${STRIPE_WEBHOOK_SECRET}
    ports:
      - "8081:8081"
    depends_on:
      postgres:
        condition: service_healthy

  portal:
    build:
      context: .
      dockerfile: cmd/portal/Dockerfile
    environment:
      DATABASE_URL: postgres://kernel:secret@postgres:5432/kernel
      CUSTOMER_JWT_SECRET: \${CUSTOMER_JWT_SECRET}
    ports:
      - "8082:8082"
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:`

const serviceMappingTable = `# Groupe logique → service → modules kernel

| Service     | Port | Modules kernel                                                  |
|-------------|------|-----------------------------------------------------------------|
| core        | 8080 | audit, auth, tenant, membership                                 |
| billing     | 8081 | billing (+ membership en lecture seule)                         |
| portal      | 8082 | end_customers, customer_auth, white_label, customer_notifications |
| (optionnel) | 8083 | notification, user, storage  ← peuvent rejoindre core ou vivre seuls |`

// ── Component ─────────────────────────────────────────────────────────────────

export default function MicroservicesPage() {
  return (
    <DocPage
      toc={toc}
      prev={{ title: 'Building a Custom Module', href: '/docs/guides/custom-module' }}
      next={{ title: 'Architecture (ADRs)', href: '/docs/architecture' }}
    >
      <PageHeader
        eyebrow="Guides"
        title="Architecture microservices"
        description="Déployez le kernel AgileFactory en plusieurs services indépendants : découpage par domaine, communication via l'outbox Postgres, et orchestration Docker."
      />

      {/* ── Architecture choice ───────────────────────────────────────────── */}

      <Section id="architecture-choice">
        <H2 id="architecture-choice">Monolith-first ou microservices ?</H2>
        <P>
          Le kernel est conçu pour démarrer en monolithe : tous les modules
          sont câblés dans un seul <code>main.go</code>, une seule DB, un seul
          processus. C&apos;est la configuration recommandée pour les équipes de
          moins de 5 développeurs ou en phase de MVP.
        </P>
        <P>
          Décomposer en services indépendants apporte de la valeur uniquement
          quand l&apos;un de ces signaux apparaît :
        </P>
        <ul className="mt-3 space-y-2 text-sm text-muted-foreground">
          <li className="flex items-start gap-2.5">
            <span className="mt-2 size-1.5 shrink-0 rounded-full bg-primary" />
            <span>
              Besoin de <strong>déployer billing séparément</strong> pour
              répondre à des exigences PCI-DSS sans impacter le reste.
            </span>
          </li>
          <li className="flex items-start gap-2.5">
            <span className="mt-2 size-1.5 shrink-0 rounded-full bg-primary" />
            <span>
              Le portail B2B2C (<code>end_customers</code>,{' '}
              <code>customer_auth</code>) est exposé à des millions
              d&apos;utilisateurs et doit scaler indépendamment du back-office.
            </span>
          </li>
          <li className="flex items-start gap-2.5">
            <span className="mt-2 size-1.5 shrink-0 rounded-full bg-primary" />
            <span>
              Des équipes distinctes possèdent des domaines distincts et
              veulent des cycles de déploiement indépendants.
            </span>
          </li>
        </ul>
        <Callout variant="tip" title="Conseil">
          <p>
            Commencez toujours en monolithe. La frontière entre modules kernel
            est déjà propre (pas de dépendances circulaires, events via outbox).
            La décomposition en services est mécanique une fois que les
            frontières sont stabilisées.
          </p>
        </Callout>
      </Section>

      {/* ── Service mapping ───────────────────────────────────────────────── */}

      <Section id="service-mapping">
        <H2 id="service-mapping">Découpage en services</H2>
        <P>
          Le kernel se divise naturellement en trois groupes logiques selon les
          dépendances entre modules. Le tableau ci-dessous montre la
          décomposition recommandée.
        </P>
        <CodeBlock code={serviceMappingTable} lang="markdown" />
        <Callout variant="warning" title="Base Postgres partagée">
          <p>
            Les trois services pointent vers la <strong>même base Postgres</strong>.
            C&apos;est intentionnel : l&apos;outbox transactionnelle repose sur une
            seule table <code>outbox_events</code> visible par tous. Si vous
            avez besoin de bases séparées, remplacez le bus in-process par un
            broker externe (NATS, Kafka) — voir la section inter-services.
          </p>
        </Callout>

        {/* ── Core service ─────────────────────────────────────────────────── */}

        <H3 id="core-service">Service core (auth + tenant + membership + audit)</H3>
        <P>
          Le service core est la fondation : il gère l&apos;identité des
          opérateurs SaaS, les tenants, les membres, et l&apos;audit. Tous les
          autres services lisent la table <code>memberships</code> pour vérifier
          les permissions — c&apos;est pourquoi ils partagent la même DB.
        </P>
        <CodeBlock code={coreMainSnippet} lang="go" filename="cmd/core/main.go" />

        {/* ── Billing service ──────────────────────────────────────────────── */}

        <H3 id="billing-service">Service billing</H3>
        <P>
          Le service billing n&apos;embarque que <code>billing</code>. Il lit
          la table <code>memberships</code> (créée par core) pour le{' '}
          <code>permission.Checker</code>, mais ne monte aucune route membership.
          Les migrations <code>membership.Migrations</code> sont relancées ici —
          <code>migrator.Run</code> est idempotent.
        </P>
        <CodeBlock code={billingMainSnippet} lang="go" filename="cmd/billing/main.go" />

        {/* ── Portal service ───────────────────────────────────────────────── */}

        <H3 id="portal-service">Service portal B2B2C</H3>
        <P>
          Le portail B2B2C regroupe les quatre modules destinés aux
          utilisateurs finaux de vos clients. <code>white_label</code> est
          instancié en premier car son <code>BrandResolver()</code> est injecté
          dans <code>customer_auth</code> et <code>customer_notifications</code>.
        </P>
        <CodeBlock code={portalMainSnippet} lang="go" filename="cmd/portal/main.go" />
      </Section>

      {/* ── Inter-service communication ───────────────────────────────────── */}

      <Section id="inter-service">
        <H2 id="inter-service">Communication inter-services via events</H2>
        <P>
          Le bus <code>events.Bus</code> utilise une table{' '}
          <code>outbox_events</code> Postgres. Quand deux services partagent la
          même DB, un event publié par billing est visible par le bus du service
          portal — il suffit que les deux bus aient souscrit à l&apos;event avant
          d&apos;appeler <code>bus.Run(ctx)</code>.
        </P>
        <CodeBlock code={interServiceSnippet} lang="go" filename="inter-service events" />
        <Callout variant="warning" title="At-least-once">
          <p>
            L&apos;outbox garantit la livraison <em>au moins une fois</em>.
            Tous les handlers d&apos;events doivent être <strong>idempotents</strong> :
            vérifiez l&apos;existence de la ressource avant de la créer, utilisez
            des clés d&apos;idempotence, ou ignorez les doublons gracieusement.
          </p>
        </Callout>
        <Callout variant="tip" title="Broker externe (optionnel)">
          <p>
            Si vous passez à des bases séparées, remplacez{' '}
            <code>events.NewBus(log)</code> par une implémentation NATS ou Kafka
            qui satisfait l&apos;interface <code>events.Publisher</code> /{' '}
            <code>events.Subscriber</code>. Aucun use case ni handler n&apos;a
            besoin de changer — seul <code>main.go</code> évolue.
          </p>
        </Callout>
      </Section>

      {/* ── Deployment ────────────────────────────────────────────────────── */}

      <Section id="deployment">
        <H2 id="deployment">Déploiement</H2>
        <P>
          Voici un <code>docker-compose.yml</code> minimal qui orchestre les
          trois services et une base Postgres partagée. Chaque service a son
          propre <code>Dockerfile</code> dans <code>cmd/&lt;service&gt;/</code>.
        </P>
        <CodeBlock code={dockerComposeSnippet} lang="yaml" filename="docker-compose.yml" />
        <Callout variant="tip" title="Ordre des migrations">
          <p>
            Chaque service appelle <code>migrator.Run</code> au démarrage.
            Puisque le migrator est idempotent, il n&apos;y a pas de coordination
            à orchestrer : le premier service qui démarre applique les
            migrations, les suivants ne font rien. En production, préférez un
            job d&apos;init dédié (<code>cmd/migrate/main.go</code>) qui run
            toutes les migrations avant de démarrer les services.
          </p>
        </Callout>
        <Callout variant="warning" title="Secrets">
          <p>
            Ne committez jamais les secrets dans{' '}
            <code>docker-compose.yml</code>. Utilisez un fichier{' '}
            <code>.env</code> (hors VCS) ou un gestionnaire de secrets (Vault,
            AWS SSM, Doppler). Les variables comme <code>JWT_SECRET</code> et{' '}
            <code>STRIPE_SECRET_KEY</code> doivent être injectées à
            l&apos;exécution.
          </p>
        </Callout>
      </Section>
    </DocPage>
  )
}
