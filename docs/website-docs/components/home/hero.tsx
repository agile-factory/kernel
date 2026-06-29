import Link from 'next/link'
import { ArrowRight } from 'lucide-react'
import { GitHubIcon } from '@/components/logo'
import { InstallCommand } from '@/components/install-command'

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border">
      {/* subtle grid backdrop */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute inset-0 opacity-[0.4] [background-image:linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [background-size:64px_64px] [mask-image:radial-gradient(ellipse_at_top,black,transparent_70%)]"
      />
      <div className="relative mx-auto max-w-[100rem] px-4 py-20 sm:px-6 sm:py-28">
        <div className="mx-auto max-w-3xl text-center">
          <Link
            href="/changelog"
            className="inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1 text-xs font-medium text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground"
          >
            <span className="rounded-full bg-primary/15 px-1.5 py-0.5 font-mono text-[10px] text-primary">
              v0.8.0
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
              className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-primary px-6 text-sm font-semibold text-primary-foreground transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:w-auto"
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

          <div className="mx-auto mt-8 max-w-md">
            <InstallCommand />
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
