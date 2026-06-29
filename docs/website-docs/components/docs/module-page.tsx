import { ApiEndpoint } from '@/components/docs/api-endpoint'
import { CodeBlock } from '@/components/docs/code-block'
import { H2, H3, P, Section, Table } from '@/components/docs/content'
import { DocPage, PageHeader } from '@/components/docs/doc-page'
import type { TocItem } from '@/components/docs/toc'
import type { ModuleDoc } from '@/lib/modules-data'
import { ArrowDownToLine, ArrowUpFromLine } from 'lucide-react'

type PageLink = { title: string; href: string }

export function ModulePage({
  mod,
  prev,
  next,
}: {
  mod: ModuleDoc
  prev?: PageLink
  next?: PageLink
}) {
  const toc: TocItem[] = [
    { id: 'overview', title: 'Overview', level: 2 },
    { id: 'installation', title: 'Installation', level: 2 },
    { id: 'configuration', title: 'Configuration', level: 2 },
    { id: 'usage', title: 'Usage', level: 2 },
  ]
  if (mod.methods.length) toc.push({ id: 'api-reference', title: 'API Reference', level: 2 })
  if (mod.events.length) toc.push({ id: 'events', title: 'Events', level: 2 })
  if (mod.endpoints.length)
    toc.push({ id: 'http-endpoints', title: 'HTTP Endpoints', level: 2 })
  toc.push({ id: 'migrations', title: 'Migrations', level: 2 })

  return (
    <DocPage toc={toc} prev={prev} next={next}>
      <PageHeader
        eyebrow={`Module · ${mod.name}`}
        title={mod.title}
        description={mod.description}
      />

      <Section id="overview">
        <H2 id="overview">Overview</H2>
        <P>{mod.overview}</P>
      </Section>

      <Section id="installation">
        <H2 id="installation">Installation</H2>
        <P>
          Import the module and register its embedded migrations with the{' '}
          <code>migrator</code>.
        </P>
        <CodeBlock code={mod.install} lang="go" filename="migrate.go" />
      </Section>

      <Section id="configuration">
        <H2 id="configuration">Configuration</H2>
        <P>
          <code>{mod.name}.Config</code> accepts the following fields:
        </P>
        <Table
          head={['Field', 'Type', 'Required', 'Description']}
          rows={mod.config.map((f) => [
            <code key="n">{f.name}</code>,
            <code key="t">{f.type}</code>,
            f.required ? (
              <span key="r" className="font-medium text-primary">
                Required
              </span>
            ) : (
              <span key="r" className="text-muted-foreground">
                Optional
              </span>
            ),
            f.description,
          ])}
        />
      </Section>

      <Section id="usage">
        <H2 id="usage">Usage</H2>
        <P>Construct the module and mount its handlers on your router:</P>
        <CodeBlock code={mod.usage} lang="go" />
      </Section>

      {mod.methods.length ? (
        <Section id="api-reference">
          <H2 id="api-reference">API Reference</H2>
          <P>Exported methods on the module value:</P>
          <div className="space-y-8">
            {mod.methods.map((method) => (
              <div key={method.name} className="space-y-3">
                <H3 id={`method-${method.name.toLowerCase()}`}>
                  <code className="text-primary">{method.name}</code>
                </H3>
                <div className="overflow-x-auto rounded-lg border border-border bg-muted/30 px-4 py-2 scrollbar-thin">
                  <code className="whitespace-pre font-mono text-xs text-foreground">
                    {method.signature}
                  </code>
                </div>
                <P>{method.description}</P>
                {method.example ? (
                  <CodeBlock code={method.example} lang="go" />
                ) : null}
              </div>
            ))}
          </div>
        </Section>
      ) : null}

      {mod.events.length ? (
        <Section id="events">
          <H2 id="events">Events</H2>
          <P>
            Events this module publishes or consumes. Published events are
            delivered at-least-once via the audit outbox.
          </P>
          <div className="space-y-4">
            {mod.events.map((evt) => (
              <div
                key={evt.name}
                className="overflow-hidden rounded-xl border border-border bg-card"
              >
                <div className="flex flex-wrap items-center gap-2 border-b border-border bg-muted/30 px-4 py-3">
                  <code className="font-mono text-sm text-foreground">
                    {evt.name}
                  </code>
                  <span
                    className={
                      'ml-auto inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-medium ' +
                      (evt.direction === 'Published'
                        ? 'border-primary/30 bg-primary/10 text-primary'
                        : 'border-emerald-500/30 bg-emerald-500/10 text-emerald-500')
                    }
                  >
                    {evt.direction === 'Published' ? (
                      <ArrowUpFromLine className="size-3" aria-hidden="true" />
                    ) : (
                      <ArrowDownToLine className="size-3" aria-hidden="true" />
                    )}
                    {evt.direction}
                  </span>
                </div>
                <div className="space-y-3 px-4 py-3">
                  <p className="text-sm text-muted-foreground">{evt.description}</p>
                  <CodeBlock code={evt.payload} lang="json" />
                </div>
              </div>
            ))}
          </div>
        </Section>
      ) : null}

      {mod.endpoints.length ? (
        <Section id="http-endpoints">
          <H2 id="http-endpoints">HTTP Endpoints</H2>
          <P>Handlers mounted by this module:</P>
          {mod.endpoints.map((ep) => (
            <ApiEndpoint
              key={`${ep.method}-${ep.path}`}
              method={ep.method}
              path={ep.path}
              auth={ep.auth}
              body={ep.body}
              responses={ep.responses}
            />
          ))}
        </Section>
      ) : null}

      <Section id="migrations">
        <H2 id="migrations">Migrations</H2>
        <P>Migration files embedded in this module, applied in order:</P>
        <Table
          head={['File', 'Description']}
          rows={mod.migrations.map((m) => [
            <code key="f">{m.file}</code>,
            m.description,
          ])}
        />
      </Section>
    </DocPage>
  )
}
