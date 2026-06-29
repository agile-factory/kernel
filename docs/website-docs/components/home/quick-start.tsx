import Link from 'next/link'
import { ArrowRight } from 'lucide-react'
import { CodeBlock } from '@/components/docs/code-block'

const wiring = `package main

import (
	"net/http"

	"github.com/agile-factory/kernel/auth"
	"github.com/agile-factory/kernel/audit"
	"github.com/agile-factory/kernel/tenant"
)

func main() {
	db := mustConnect()                  // *sql.DB you own
	mux := http.NewServeMux()

	// Each module is wired explicitly — no global state, no magic.
	auditMod := audit.New(audit.Config{DB: db})
	authMod := auth.New(auth.Config{
		DB:       db,
		Audit:    auditMod,
		JWTSecret: env("JWT_SECRET"),
	})
	tenantMod := tenant.New(tenant.Config{DB: db, Audit: auditMod})

	// Mount handlers under any prefix you like.
	authMod.Mount(mux, "/api/v1/auth")
	tenantMod.Mount(mux, "/api/v1/tenants")

	http.ListenAndServe(":8080", mux)
}`

export async function QuickStart() {
  return (
    <section className="border-t border-border bg-card/30">
      <div className="mx-auto grid max-w-[100rem] items-center gap-10 px-4 py-20 sm:px-6 lg:grid-cols-2">
        <div>
          <p className="font-mono text-sm font-medium text-primary">
            Quick start
          </p>
          <h2 className="mt-3 text-balance text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Wire it explicitly. Own it completely.
          </h2>
          <p className="mt-4 text-pretty leading-relaxed text-muted-foreground">
            No hidden lifecycle, no service container, no reflection. You pass
            your own <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.85em] text-foreground">*sql.DB</code>,
            compose modules in dependency order, and mount their handlers on the
            router you control.
          </p>
          <ul className="mt-6 space-y-3 text-sm">
            {[
              'Zero global state — everything is an explicit dependency',
              'Each module ships its own SQL migrations',
              'At-least-once delivery via the outbox pattern',
              'Cursor pagination on every list endpoint',
            ].map((item) => (
              <li key={item} className="flex items-start gap-2.5">
                <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary" />
                <span className="text-muted-foreground">{item}</span>
              </li>
            ))}
          </ul>
          <Link
            href="/docs/getting-started"
            className="mt-8 inline-flex items-center gap-2 text-sm font-semibold text-primary transition-opacity hover:opacity-80"
          >
            Read the Getting Started guide
            <ArrowRight className="size-4" aria-hidden="true" />
          </Link>
        </div>
        <div className="min-w-0">
          <CodeBlock code={wiring} lang="go" filename="main.go" />
        </div>
      </div>
    </section>
  )
}
