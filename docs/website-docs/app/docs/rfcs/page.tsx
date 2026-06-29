import type { Metadata } from 'next'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import { H2, P, Section } from '@/components/docs/content'
import type { TocItem } from '@/components/docs/toc'

export const metadata: Metadata = {
  title: 'RFCs',
  description:
    'Requests for Comments for AgileFactory/kernel — proposals under discussion, accepted designs, and historical records.',
}

type RfcStatus = 'Draft' | 'In Review' | 'Accepted' | 'Implemented' | 'Withdrawn'

type Rfc = {
  id: string
  title: string
  status: RfcStatus
  author: string
  updated: string
  summary: string
}

const rfcs: Rfc[] = [
  {
    id: 'RFC-0009',
    title: 'Tenant-scoped rate limiting',
    status: 'In Review',
    author: 'm.ouattara',
    updated: '2026-06-18',
    summary:
      'Introduce a pluggable rate-limit middleware keyed by tenant_id with a token-bucket backend, configurable per route group.',
  },
  {
    id: 'RFC-0008',
    title: 'Pluggable secrets provider',
    status: 'Draft',
    author: 'a.kone',
    updated: '2026-06-10',
    summary:
      'Abstract secret resolution behind a SecretsProvider interface so deployments can back config with Vault, SSM, or env vars.',
  },
  {
    id: 'RFC-0007',
    title: 'Outbox partitioning for high-volume tenants',
    status: 'Accepted',
    author: 's.traore',
    updated: '2026-05-28',
    summary:
      'Partition the outbox table by tenant bucket to keep drain queries fast once a tenant exceeds ~10M events.',
  },
  {
    id: 'RFC-0006',
    title: 'customer-auth OTP channel abstraction',
    status: 'Implemented',
    author: 'm.ouattara',
    updated: '2026-05-02',
    summary:
      'Allow OTP delivery over SMS or email through a single Channel interface, shipped in v0.9.0.',
  },
  {
    id: 'RFC-0005',
    title: 'Drop v1 offset pagination',
    status: 'Withdrawn',
    author: 'a.kone',
    updated: '2026-04-14',
    summary:
      'Proposed removing v1 list endpoints. Withdrawn: several integrators still depend on offset reads for export jobs.',
  },
]

const statusStyles: Record<RfcStatus, string> = {
  Draft: 'border-border bg-muted/40 text-muted-foreground',
  'In Review': 'border-amber-500/30 bg-amber-500/10 text-amber-500',
  Accepted: 'border-primary/30 bg-primary/10 text-primary',
  Implemented: 'border-emerald-500/30 bg-emerald-500/10 text-emerald-500',
  Withdrawn: 'border-destructive/30 bg-destructive/10 text-destructive',
}

const toc: TocItem[] = [
  { id: 'process', title: 'The RFC process', level: 2 },
  { id: 'index', title: 'RFC index', level: 2 },
]

export default function RfcsPage() {
  return (
    <DocPage
      toc={toc}
      prev={{ title: 'Architecture (ADRs)', href: '/docs/architecture' }}
      next={{ title: 'Changelog', href: '/changelog' }}
    >
      <PageHeader
        eyebrow="Reference"
        title="RFCs"
        description="Substantial changes to the kernel start as a Request for Comments. RFCs capture the proposal, the discussion, and the resolution before any code lands."
      />

      <Section id="process">
        <H2 id="process">The RFC process</H2>
        <P>
          An RFC moves through <code>Draft</code> → <code>In Review</code> →{' '}
          <code>Accepted</code> → <code>Implemented</code>. Anything that changes a
          public module interface, adds a table, or alters delivery semantics
          requires an RFC. Smaller changes go straight to a pull request.
        </P>
      </Section>

      <Section id="index">
        <H2 id="index">RFC index</H2>
        <div className="space-y-3">
          {rfcs.map((rfc) => (
            <article
              key={rfc.id}
              className="rounded-xl border border-border bg-card p-5 transition-colors hover:border-primary/40"
            >
              <div className="flex flex-wrap items-center gap-3">
                <span className="font-mono text-xs font-semibold text-muted-foreground">
                  {rfc.id}
                </span>
                <h3 className="text-base font-semibold text-foreground">
                  {rfc.title}
                </h3>
                <span
                  className={
                    'ml-auto inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-medium ' +
                    statusStyles[rfc.status]
                  }
                >
                  {rfc.status}
                </span>
              </div>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                {rfc.summary}
              </p>
              <div className="mt-3 flex items-center gap-3 text-xs text-muted-foreground">
                <span className="font-mono">@{rfc.author}</span>
                <span aria-hidden="true">·</span>
                <span>
                  Updated{' '}
                  <time dateTime={rfc.updated}>
                    {new Date(rfc.updated).toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'short',
                      day: 'numeric',
                    })}
                  </time>
                </span>
              </div>
            </article>
          ))}
        </div>
      </Section>
    </DocPage>
  )
}
