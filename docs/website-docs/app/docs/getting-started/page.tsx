import type { Metadata } from 'next'
import { Callout } from '@/components/docs/callout'
import { CodeBlock } from '@/components/docs/code-block'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import { DocLink, H2, H3, P, Section } from '@/components/docs/content'
import type { TocItem } from '@/components/docs/toc'

export const metadata: Metadata = {
  title: 'Getting Started',
  description:
    'Install AgileFactory/kernel, wire your first modules in main.go, run migrations, and make your first authenticated request.',
}

const toc: TocItem[] = [
  { id: 'prerequisites', title: 'Prerequisites', level: 2 },
  { id: 'install', title: 'Installation', level: 2 },
  { id: 'wiring', title: 'Wiring main.go', level: 2 },
  { id: 'migrations', title: 'Running migrations', level: 3 },
  { id: 'first-request', title: 'Your first request', level: 2 },
  { id: 'next-steps', title: 'Next steps', level: 2 },
]

const installSnippet = `go get github.com/agile-factory/kernel`

const wiringSnippet = `package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/agile-factory/kernel/audit"
	"github.com/agile-factory/kernel/auth"
	"github.com/agile-factory/kernel/tenant"
)

func main() {
	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// Compose modules in dependency order.
	auditMod := audit.New(audit.Config{DB: db})
	authMod := auth.New(auth.Config{
		DB:        db,
		Audit:     auditMod,
		JWTSecret: os.Getenv("JWT_SECRET"),
	})
	tenantMod := tenant.New(tenant.Config{DB: db, Audit: auditMod})

	mux := http.NewServeMux()
	authMod.Mount(mux, "/api/v1/auth")
	tenantMod.Mount(mux, "/api/v1/tenants")

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}`

const migrateSnippet = `import "github.com/agile-factory/kernel/migrator"

// Run every module's embedded migrations in one transaction-safe pass.
func migrate(db *sql.DB) error {
	return migrator.Run(db,
		audit.Migrations,
		auth.Migrations,
		tenant.Migrations,
	)
}`

const requestSnippet = `# Sign up the first owner — this also creates their tenant.
curl -s -X POST http://localhost:8080/api/v1/auth/signup \\
  -H 'Content-Type: application/json' \\
  -d '{
    "email": "owner@acme.io",
    "password": "s3cret-passphrase",
    "tenant_name": "Acme, Inc."
  }'`

const responseSnippet = `{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "rt_8f2c1d...",
  "user": {
    "id": "usr_01HF...",
    "email": "owner@acme.io",
    "tenant_id": "tnt_01HF...",
    "role": "owner"
  }
}`

export default function GettingStartedPage() {
  return (
    <DocPage
      toc={toc}
      prev={{ title: 'Introduction', href: '/docs' }}
      next={{ title: 'auth module', href: '/docs/modules/auth' }}
    >
      <PageHeader
        eyebrow="Getting Started"
        title="Getting Started"
        description="Install the kernel, wire your first modules, run migrations, and make an authenticated request — in about five minutes."
      />

      <Section id="prerequisites">
        <H2 id="prerequisites">Prerequisites</H2>
        <P>
          kernel targets <code>Go 1.22+</code> and PostgreSQL 14+. You bring your
          own <code>*sql.DB</code> — any driver works, though the examples use{' '}
          <code>pgx</code>.
        </P>
      </Section>

      <Section id="install">
        <H2 id="install">Installation</H2>
        <P>Add the module to your project:</P>
        <CodeBlock code={installSnippet} lang="bash" />
        <Callout variant="tip">
          <p>
            kernel has no required third-party runtime dependencies beyond the
            standard library and your SQL driver. Optional providers (Stripe,
            S3, SMS) are pulled in only when you import their module.
          </p>
        </Callout>
      </Section>

      <Section id="wiring">
        <H2 id="wiring">Wiring main.go</H2>
        <P>
          Construct each module with its typed <code>Config</code>, passing
          shared dependencies explicitly. The <code>audit</code> module is a
          common dependency, so build it first.
        </P>
        <CodeBlock code={wiringSnippet} lang="go" filename="main.go" />

        <H3 id="migrations">Running migrations</H3>
        <P>
          Each module embeds its own SQL migration files. Use the{' '}
          <code>migrator</code> package to run them all in dependency order.
        </P>
        <CodeBlock code={migrateSnippet} lang="go" filename="migrate.go" />
        <Callout variant="warning" title="Order matters">
          <p>
            Run migrations in the same dependency order you wire modules. The{' '}
            <code>migrator</code> records applied versions, so re-running is
            idempotent.
          </p>
        </Callout>
      </Section>

      <Section id="first-request">
        <H2 id="first-request">Your first request</H2>
        <P>
          With the server running, sign up the first owner. This single call
          provisions the user, their tenant, and an audit event.
        </P>
        <CodeBlock code={requestSnippet} lang="bash" />
        <P>You will receive an access token, refresh token, and the new user:</P>
        <CodeBlock code={responseSnippet} lang="json" />
      </Section>

      <Section id="next-steps">
        <H2 id="next-steps">Next steps</H2>
        <P>
          Explore the <DocLink href="/docs/modules/auth">auth</DocLink> and{' '}
          <DocLink href="/docs/modules/audit">audit</DocLink> module references,
          or read the{' '}
          <DocLink href="/docs/architecture">Architecture (ADRs)</DocLink> to
          understand the decisions behind the design.
        </P>
      </Section>
    </DocPage>
  )
}
