import Link from 'next/link'
import { ArrowRight } from 'lucide-react'
import type { Metadata } from 'next'
import { Callout } from '@/components/docs/callout'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import { DocLink, H2, P, Section } from '@/components/docs/content'
import type { TocItem } from '@/components/docs/toc'
import { moduleList } from '@/lib/navigation'

export const metadata: Metadata = {
  title: 'Introduction',
  description:
    'AgileFactory/kernel is a composable Go library for building B2B SaaS products with auth, multi-tenancy, audit, billing, and notifications.',
}

const toc: TocItem[] = [
  { id: 'what-is-kernel', title: 'What is kernel?', level: 2 },
  { id: 'philosophy', title: 'Design philosophy', level: 2 },
  { id: 'modules', title: 'Available modules', level: 2 },
]

export default function DocsIndexPage() {
  return (
    <DocPage toc={toc} next={{ title: 'Getting Started', href: '/docs/getting-started' }}>
      <PageHeader
        eyebrow="Introduction"
        title="AgileFactory/kernel"
        description="A production-ready Go library that gives your B2B SaaS product the boring-but-critical infrastructure, so you can focus on your product."
      />

      <Section id="what-is-kernel">
        <H2 id="what-is-kernel">What is kernel?</H2>
        <P>
          <code>kernel</code> powers multiple production B2B SaaS products —
          Kay-Point, 380 Solutions, BRVM, and others. It bundles auth,
          multi-tenancy, audit trails, billing, and notifications as composable
          modules with at-least-once reliability, cursor pagination, and zero
          global state.
        </P>
        <Callout variant="info" title="It is not a framework">
          <p>
            kernel is a set of composable modules you wire explicitly in{' '}
            <code>main.go</code> and own completely. There is no service
            container, no lifecycle magic, and no reflection.
          </p>
        </Callout>
      </Section>

      <Section id="philosophy">
        <H2 id="philosophy">Design philosophy</H2>
        <P>
          Every module follows the same contract: it accepts a typed{' '}
          <code>Config</code>, ships its own SQL migrations, exposes exported Go
          methods, and mounts HTTP handlers under a prefix you choose. You pass
          your own <code>*sql.DB</code> and compose modules in dependency order.
        </P>
        <ul className="space-y-2 text-sm text-muted-foreground">
          <li className="flex items-start gap-2.5">
            <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary" />
            <span>
              <strong className="text-foreground">Explicit over implicit.</strong>{' '}
              Dependencies are constructor arguments, never globals.
            </span>
          </li>
          <li className="flex items-start gap-2.5">
            <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary" />
            <span>
              <strong className="text-foreground">At-least-once delivery.</strong>{' '}
              Side effects go through a transactional outbox.
            </span>
          </li>
          <li className="flex items-start gap-2.5">
            <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary" />
            <span>
              <strong className="text-foreground">Cursor pagination.</strong>{' '}
              Every list endpoint is stable under concurrent writes.
            </span>
          </li>
        </ul>
        <P>
          Ready to wire your first module? Head to the{' '}
          <DocLink href="/docs/getting-started">Getting Started</DocLink> guide.
        </P>
      </Section>

      <Section id="modules">
        <H2 id="modules">Available modules</H2>
        <P>
          kernel ships {moduleList.length} modules across core SaaS and B2B2C
          concerns. Browse any of them for full API, event, and endpoint
          references.
        </P>
        <div className="grid gap-3 sm:grid-cols-2">
          {moduleList.map((m) => (
            <Link
              key={m.slug}
              href={`/docs/modules/${m.slug}`}
              className="group flex flex-col gap-1 rounded-xl border border-border bg-card p-4 transition-colors hover:border-primary/40"
            >
              <span className="flex items-center gap-2 font-mono text-sm font-semibold text-foreground group-hover:text-primary">
                {m.title}
                <ArrowRight
                  className="size-3.5 opacity-0 transition-opacity group-hover:opacity-100"
                  aria-hidden="true"
                />
              </span>
              <span className="text-sm leading-relaxed text-muted-foreground">
                {m.summary}
              </span>
            </Link>
          ))}
        </div>
      </Section>
    </DocPage>
  )
}
