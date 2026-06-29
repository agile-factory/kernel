import Link from 'next/link'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export function H2({ id, children }: { id: string; children: ReactNode }) {
  return (
    <h2
      id={id}
      className="group scroll-mt-24 border-b border-border pb-2 pt-2 text-2xl font-semibold tracking-tight text-foreground"
    >
      <a href={`#${id}`} className="no-underline">
        {children}
        <span className="ml-2 text-primary opacity-0 transition-opacity group-hover:opacity-100">
          #
        </span>
      </a>
    </h2>
  )
}

export function H3({ id, children }: { id: string; children: ReactNode }) {
  return (
    <h3
      id={id}
      className="group scroll-mt-24 text-lg font-semibold tracking-tight text-foreground"
    >
      <a href={`#${id}`} className="no-underline">
        {children}
        <span className="ml-2 text-primary opacity-0 transition-opacity group-hover:opacity-100">
          #
        </span>
      </a>
    </h3>
  )
}

export function Lead({ children }: { children: ReactNode }) {
  return (
    <p className="text-pretty text-base leading-relaxed text-muted-foreground">
      {children}
    </p>
  )
}

export function P({ children }: { children: ReactNode }) {
  return (
    <p className="text-pretty leading-relaxed text-muted-foreground [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.85em] [&_code]:text-foreground">
      {children}
    </p>
  )
}

export function InlineCode({ children }: { children: ReactNode }) {
  return (
    <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[0.85em] text-foreground">
      {children}
    </code>
  )
}

export function DocLink({ href, children }: { href: string; children: ReactNode }) {
  const isExternal = href.startsWith('http')
  if (isExternal) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noreferrer"
        className="font-medium text-primary underline-offset-4 hover:underline"
      >
        {children}
      </a>
    )
  }
  return (
    <Link
      href={href}
      className="font-medium text-primary underline-offset-4 hover:underline"
    >
      {children}
    </Link>
  )
}

export function Table({
  head,
  rows,
  className,
}: {
  head: string[]
  rows: ReactNode[][]
  className?: string
}) {
  return (
    <div
      className={cn(
        'my-6 overflow-x-auto rounded-xl border border-border scrollbar-thin',
        className,
      )}
    >
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-border bg-muted/40 text-left">
            {head.map((h) => (
              <th
                key={h}
                className="px-4 py-2.5 font-semibold text-foreground whitespace-nowrap"
              >
                {h}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr
              key={i}
              className="border-b border-border last:border-0 align-top"
            >
              {row.map((cell, j) => (
                <td
                  key={j}
                  className="px-4 py-2.5 text-muted-foreground [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.85em] [&_code]:text-foreground"
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export function Section({
  id,
  children,
  className,
}: {
  id?: string
  children: ReactNode
  className?: string
}) {
  return (
    <section id={id} className={cn('space-y-4 pt-8', className)}>
      {children}
    </section>
  )
}
