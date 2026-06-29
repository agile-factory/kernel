import Link from 'next/link'
import { ArrowRight } from 'lucide-react'
import { GitHubIcon } from '@/components/logo'
import { InstallCommand } from '@/components/install-command'

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border">
      {/* Decorative orb glow */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute left-1/2 top-0 -translate-x-1/2 -translate-y-1/4 h-[500px] w-[700px] rounded-full bg-primary opacity-[0.08] blur-3xl"
      />
      {/* Grid backdrop */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-[0.35] [background-image:linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [background-size:64px_64px] [mask-image:radial-gradient(ellipse_at_top,black,transparent_70%)]"
      />
      <div className="relative mx-auto max-w-[100rem] px-4 py-24 sm:px-6 sm:py-32">
        <div className="mx-auto max-w-3xl text-center">
          <Link
            href="/changelog"
            className="inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
          >
            <span className="flex items-center gap-1.5">
              <span className="relative flex size-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-500 opacity-60" />
                <span className="relative inline-flex size-2 rounded-full bg-emerald-500" />
              </span>
              <span className="rounded-full bg-primary/15 px-1.5 py-0.5 font-mono text-[10px] text-primary">
                v0.8.0
              </span>
            </span>
            Cursor pagination + outbox delivery now stable
            <ArrowRight className="size-3" aria-hidden="true" />
          </Link>

          <h1 className="mt-6 text-balance text-4xl font-bold tracking-tight text-foreground sm:text-6xl">
            Go SaaS Accelerator.{' '}
            <span className="text-primary">Ship B2B features in days.</span>
          </h1>

          <p className="mx-auto mt-6 max-w-2xl text-pretty text-lg leading-relaxed text-muted-foreground">
            AgileFactory/kernel is a production-ready Go library that gives your
            B2B SaaS product auth, multi-tenancy, audit trails, billing, and
            notifications — with at-least-once reliability, cursor pagination,
            and zero global state.
          </p>

          <div className="mx-auto mt-8 flex max-w-md flex-col items-center justify-center gap-3 sm:flex-row">
            <Link
              href="/docs/getting-started"
              className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-lg shadow-primary/20 transition-all hover:opacity-90 hover:shadow-primary/30 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:w-auto"
            >
              Get Started
              <ArrowRight className="size-4" aria-hidden="true" />
            </Link>
            <a
              href="https://github.com/agile-factory/kernel"
              target="_blank"
              rel="noreferrer"
              className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg border border-border bg-card px-6 text-sm font-semibold text-foreground transition-colors hover:bg-accent sm:w-auto"
            >
              <GitHubIcon className="size-4" />
              View on GitHub
            </a>
          </div>

          <div className="mx-auto mt-6 max-w-md">
            <InstallCommand />
          </div>

          {/* Stats row */}
          <div className="mt-8 flex flex-wrap items-center justify-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
            <span className="font-mono font-medium text-foreground">12 modules</span>
            <span aria-hidden="true" className="text-border">·</span>
            <span>Go 1.21+</span>
            <span aria-hidden="true" className="text-border">·</span>
            <span>Apache-2.0</span>
            <span aria-hidden="true" className="text-border">·</span>
            <span>Zero global state</span>
          </div>

          <p className="mt-6 text-sm text-muted-foreground">
            It is not a framework. It is a set of composable modules you wire
            explicitly in{' '}
            <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.85em] text-foreground">
              main.go
            </code>{' '}
            and own completely.
          </p>
        </div>
      </div>
    </section>
  )
}
