import type { Metadata } from 'next'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import { DocLink, H2, P, Section } from '@/components/docs/content'
import { Callout } from '@/components/docs/callout'
import { CodeBlock } from '@/components/docs/code-block'
import type { TocItem } from '@/components/docs/toc'

export const metadata: Metadata = {
  title: 'Architecture (ADRs)',
  description:
    'Architecture Decision Records for AgileFactory/kernel — the constraints and trade-offs behind module isolation, the event bus, the outbox, and cursor pagination.',
}

const toc: TocItem[] = [
  { id: 'principles', title: 'Design principles', level: 2 },
  { id: 'adr-001', title: 'ADR-001 · Module isolation', level: 2 },
  { id: 'adr-002', title: 'ADR-002 · Event bus', level: 2 },
  { id: 'adr-003', title: 'ADR-003 · Transactional outbox', level: 2 },
  { id: 'adr-004', title: 'ADR-004 · Cursor pagination', level: 2 },
  { id: 'adr-005', title: 'ADR-005 · No global state', level: 2 },
]

function Bullet({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-2.5">
      <span className="mt-2 size-1.5 shrink-0 rounded-full bg-primary" />
      <span>{children}</span>
    </li>
  )
}

export default function ArchitecturePage() {
  return (
    <DocPage
      toc={toc}
      prev={{
        title: 'customer-notifications',
        href: '/docs/modules/customer-notifications',
      }}
      next={{ title: 'RFCs', href: '/docs/rfcs' }}
    >
      <PageHeader
        eyebrow="Reference"
        title="Architecture (ADRs)"
        description="The decisions that shape the kernel, recorded as immutable ADRs. Each one captures the context, the choice, and the consequences you live with."
      />

      <Section id="principles">
        <H2 id="principles">Design principles</H2>
        <ul className="space-y-2.5 text-sm leading-relaxed text-muted-foreground">
          <Bullet>
            <strong className="text-foreground">Composition over framework.</strong>{' '}
            The kernel is a library, not a runtime. You own <code>main()</code>,
            the router, and the database handle.
          </Bullet>
          <Bullet>
            <strong className="text-foreground">Reliability by default.</strong>{' '}
            Side effects flow through a transactional outbox, so a committed
            change is never silently lost.
          </Bullet>
          <Bullet>
            <strong className="text-foreground">Tenancy is non-negotiable.</strong>{' '}
            Every query is scoped by <code>tenant_id</code>; there is no global
            read path.
          </Bullet>
        </ul>
      </Section>

      <Section id="adr-001">
        <H2 id="adr-001">ADR-001 · Module isolation</H2>
        <P>
          <strong className="text-foreground">Status: Accepted.</strong> Each
          module owns its own tables, migrations, and public Go interface. Modules
          never import one another&apos;s internal packages — they communicate
          through the event bus or through explicit constructor dependencies.
        </P>
        <Callout variant="info" title="Consequence">
          <p>
            A module can be removed by deleting its constructor call and running
            its down-migrations. Nothing else holds a hard reference to its
            internals.
          </p>
        </Callout>
      </Section>

      <Section id="adr-002">
        <H2 id="adr-002">ADR-002 · In-process event bus</H2>
        <P>
          Modules publish domain events to a synchronous, in-process bus. We chose
          in-process delivery over an external broker because the kernel targets
          single-binary B2B SaaS deployments where operational simplicity beats
          horizontal fan-out.
        </P>
        <CodeBlock
          lang="go"
          filename="bus.go"
          code={`bus := events.NewBus()

// notifications subscribes to auth events
bus.Subscribe("owner.invited", notifications.OnOwnerInvited)

// auth publishes without knowing who listens
bus.Publish(ctx, events.Event{
	Name:     "owner.invited",
	TenantID: tenantID,
	Payload:  payload,
})`}
        />
        <Callout variant="warning" title="Trade-off">
          <p>
            Subscribers run in the publisher&apos;s process. Long-running handlers
            should hand off to the outbox rather than block the request path.
          </p>
        </Callout>
      </Section>

      <Section id="adr-003">
        <H2 id="adr-003">ADR-003 · Transactional outbox</H2>
        <P>
          Durable side effects — emails, webhooks, audit fan-out — are written to
          an <code>outbox</code> table inside the same transaction as the change
          that triggered them. A background worker drains the outbox and delivers
          at-least-once.
        </P>
        <ul className="space-y-2.5 text-sm leading-relaxed text-muted-foreground">
          <Bullet>If the transaction rolls back, the side effect never happened.</Bullet>
          <Bullet>If delivery fails, the row stays and is retried with backoff.</Bullet>
          <Bullet>
            Consumers must be idempotent — at-least-once means a message can arrive
            more than once.
          </Bullet>
        </ul>
      </Section>

      <Section id="adr-004">
        <H2 id="adr-004">ADR-004 · Cursor pagination</H2>
        <P>
          List endpoints expose <code>v2</code> cursor pagination. Offset
          pagination (<code>v1</code>) drifts when rows are inserted between page
          reads; an opaque cursor encodes the last-seen{' '}
          <code>(created_at, id)</code> tuple and stays stable under concurrent
          writes.
        </P>
        <CodeBlock
          lang="json"
          filename="response.json"
          code={`{
  "items": [ /* ... */ ],
  "next_cursor": "eyJ0IjoiMjAyNi0wNi0yOFQxMjowMDowMFoiLCJpZCI6ImV2dF8wMUhGIn0",
  "has_more": true
}`}
        />
      </Section>

      <Section id="adr-005">
        <H2 id="adr-005">ADR-005 · No global state</H2>
        <P>
          There are no package-level singletons, no <code>init()</code> side
          effects, and no hidden global database handle. Every module is
          constructed explicitly and receives its dependencies as arguments. This
          keeps tests parallelizable and multi-tenant isolation auditable.
        </P>
        <Callout variant="tip" title="Why this matters">
          <p>
            Because nothing is global, you can spin up two fully independent kernel
            instances in a single test binary — one per tenant fixture — with zero
            cross-talk. See the related{' '}
            <DocLink href="/docs/rfcs">RFCs</DocLink> for proposals that build on
            this guarantee.
          </p>
        </Callout>
      </Section>
    </DocPage>
  )
}
