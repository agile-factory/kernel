import type { Metadata } from 'next'
import { TopBar } from '@/components/docs/top-bar'
import { SiteFooter } from '@/components/site-footer'

export const metadata: Metadata = {
  title: 'Changelog',
  description:
    'Release notes for AgileFactory/kernel — new modules, breaking changes, and fixes, following semantic versioning.',
}

type ChangeKind = 'Added' | 'Changed' | 'Fixed' | 'Breaking' | 'Deprecated'

type Release = {
  version: string
  date: string
  highlight?: string
  changes: { kind: ChangeKind; text: string }[]
}

const releases: Release[] = [
  {
    version: 'v1.2.0',
    date: '2026-06-22',
    highlight: 'B2B2C suite goes stable',
    changes: [
      {
        kind: 'Added',
        text: 'customer-notifications module with per-tenant template and sender overrides.',
      },
      {
        kind: 'Added',
        text: 'white-label custom-domain DNS verification with automatic TLS provisioning hooks.',
      },
      {
        kind: 'Changed',
        text: 'end-customers list endpoints now default to v2 cursor pagination.',
      },
      {
        kind: 'Fixed',
        text: 'Race in the outbox worker that could delay delivery under sustained write load.',
      },
    ],
  },
  {
    version: 'v1.1.0',
    date: '2026-05-30',
    changes: [
      {
        kind: 'Added',
        text: 'storage module with S3-compatible drivers for R2, MinIO, AWS, and Backblaze B2.',
      },
      {
        kind: 'Added',
        text: 'audit ListV2 cursor pagination alongside the existing v1 offset reads.',
      },
      {
        kind: 'Deprecated',
        text: 'audit.List (v1 offset) is deprecated in favor of ListV2; it will be removed in v2.0.0.',
      },
    ],
  },
  {
    version: 'v1.0.0',
    date: '2026-04-18',
    highlight: 'First stable release',
    changes: [
      {
        kind: 'Breaking',
        text: 'All module constructors now take a typed Config struct instead of positional arguments.',
      },
      {
        kind: 'Added',
        text: 'Core modules: auth, tenant, membership, audit, billing, notifications, and user.',
      },
      {
        kind: 'Added',
        text: 'Transactional outbox guaranteeing at-least-once delivery of domain events.',
      },
      {
        kind: 'Changed',
        text: 'Migrations are now embedded per-module and applied through a shared migrator.',
      },
    ],
  },
  {
    version: 'v0.9.0',
    date: '2026-03-05',
    changes: [
      {
        kind: 'Added',
        text: 'customer-auth with OTP password reset over SMS or email (RFC-0006).',
      },
      { kind: 'Fixed', text: 'JWT refresh tokens were not rotated on reuse detection.' },
    ],
  },
]

const kindStyles: Record<ChangeKind, string> = {
  Added: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-500',
  Changed: 'border-primary/30 bg-primary/10 text-primary',
  Fixed: 'border-sky-500/30 bg-sky-500/10 text-sky-500',
  Breaking: 'border-destructive/30 bg-destructive/10 text-destructive',
  Deprecated: 'border-amber-500/30 bg-amber-500/10 text-amber-500',
}

export default function ChangelogPage() {
  return (
    <div className="min-h-screen bg-background">
      <TopBar />
      <main className="mx-auto max-w-3xl px-4 py-14 sm:px-6">
        <header className="space-y-3 pb-10">
          <p className="font-mono text-sm font-medium text-primary">Reference</p>
          <h1 className="text-balance text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
            Changelog
          </h1>
          <p className="text-pretty text-lg leading-relaxed text-muted-foreground">
            Every release of AgileFactory/kernel, following semantic versioning.
            Breaking changes are only introduced on major versions.
          </p>
        </header>

        <div className="relative space-y-12 border-l border-border pl-8">
          {releases.map((release) => (
            <section key={release.version} className="relative">
              <span className="absolute -left-[2.55rem] top-1 flex size-4 items-center justify-center">
                <span className="size-3 rounded-full border-2 border-primary bg-background" />
              </span>

              <div className="flex flex-wrap items-baseline gap-3">
                <h2 className="font-mono text-xl font-semibold text-foreground">
                  {release.version}
                </h2>
                <time
                  dateTime={release.date}
                  className="text-sm text-muted-foreground"
                >
                  {new Date(release.date).toLocaleDateString('en-US', {
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric',
                  })}
                </time>
              </div>

              {release.highlight ? (
                <p className="mt-1 text-sm font-medium text-primary">
                  {release.highlight}
                </p>
              ) : null}

              <ul className="mt-4 space-y-2.5">
                {release.changes.map((change, i) => (
                  <li key={i} className="flex items-start gap-3 text-sm leading-relaxed">
                    <span
                      className={
                        'mt-0.5 inline-flex w-20 shrink-0 items-center justify-center rounded-md border px-2 py-0.5 text-xs font-medium ' +
                        kindStyles[change.kind]
                      }
                    >
                      {change.kind}
                    </span>
                    <span className="text-muted-foreground">{change.text}</span>
                  </li>
                ))}
              </ul>
            </section>
          ))}
        </div>
      </main>
      <SiteFooter />
    </div>
  )
}
