import Link from 'next/link'
import { ArrowLeft, ArrowRight } from 'lucide-react'
import type { ReactNode } from 'react'
import { TableOfContents, type TocItem } from './toc'

type PageLink = { title: string; href: string }

export function DocPage({
  toc = [],
  children,
  prev,
  next,
}: {
  toc?: TocItem[]
  children: ReactNode
  prev?: PageLink
  next?: PageLink
}) {
  return (
    <div className="flex gap-8 lg:gap-12">
      <article className="min-w-0 flex-1 py-10 lg:max-w-3xl">
        {children}
        {(prev || next) && (
          <div className="mt-12 grid grid-cols-1 gap-4 border-t border-border pt-8 sm:grid-cols-2">
            {prev ? (
              <Link
                href={prev.href}
                className="group flex flex-col gap-1.5 rounded-xl border border-border bg-card p-5 transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md hover:shadow-primary/5"
              >
                <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <ArrowLeft
                    className="size-3.5 transition-transform group-hover:-translate-x-0.5"
                    aria-hidden="true"
                  />
                  Previous
                </span>
                <span className="font-medium text-foreground transition-colors group-hover:text-primary">
                  {prev.title}
                </span>
              </Link>
            ) : (
              <span />
            )}
            {next ? (
              <Link
                href={next.href}
                className="group flex flex-col gap-1.5 rounded-xl border border-border bg-card p-5 text-right transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:shadow-md hover:shadow-primary/5 sm:items-end"
              >
                <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  Next
                  <ArrowRight
                    className="size-3.5 transition-transform group-hover:translate-x-0.5"
                    aria-hidden="true"
                  />
                </span>
                <span className="font-medium text-foreground transition-colors group-hover:text-primary">
                  {next.title}
                </span>
              </Link>
            ) : (
              <span />
            )}
          </div>
        )}
      </article>
      <aside className="sticky top-16 hidden h-[calc(100vh-4rem)] w-56 shrink-0 overflow-y-auto py-10 scrollbar-thin xl:block">
        <TableOfContents items={toc} />
      </aside>
    </div>
  )
}

export function PageHeader({
  eyebrow,
  title,
  description,
}: {
  eyebrow?: string
  title: string
  description: string
}) {
  return (
    <header className="space-y-3 pb-2">
      {eyebrow ? (
        <p className="font-mono text-sm font-medium text-primary">{eyebrow}</p>
      ) : null}
      <h1 className="text-balance text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
        {title}
      </h1>
      <p className="text-pretty text-lg leading-relaxed text-muted-foreground">
        {description}
      </p>
    </header>
  )
}
