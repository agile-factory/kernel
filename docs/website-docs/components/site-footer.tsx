import Link from 'next/link'
import { Logo } from '@/components/logo'

const cols: { title: string; links: { label: string; href: string }[] }[] = [
  {
    title: 'Documentation',
    links: [
      { label: 'Getting Started', href: '/docs/getting-started' },
      { label: 'Modules', href: '/docs/modules/audit' },
      { label: 'Architecture', href: '/docs/architecture' },
      { label: 'Changelog', href: '/changelog' },
    ],
  },
  {
    title: 'Modules',
    links: [
      { label: 'auth', href: '/docs/modules/auth' },
      { label: 'audit', href: '/docs/modules/audit' },
      { label: 'billing', href: '/docs/modules/billing' },
      { label: 'notifications', href: '/docs/modules/notifications' },
    ],
  },
  {
    title: 'Project',
    links: [
      { label: 'GitHub', href: 'https://github.com/agile-factory/kernel' },
      { label: 'RFCs', href: '/docs/rfcs' },
      { label: 'pkg.go.dev', href: 'https://pkg.go.dev' },
    ],
  },
]

export function SiteFooter() {
  return (
    <footer className="border-t border-border bg-background">
      <div className="mx-auto grid max-w-[100rem] gap-8 px-4 py-12 sm:px-6 md:grid-cols-[1.5fr_1fr_1fr_1fr]">
        <div className="space-y-3">
          <Logo />
          <p className="max-w-xs text-sm leading-relaxed text-muted-foreground">
            A composable Go library for B2B SaaS. Wire it explicitly in{' '}
            <code className="font-mono text-foreground">main.go</code> and own it
            completely.
          </p>
        </div>
        {cols.map((col) => (
          <div key={col.title}>
            <p className="mb-3 text-sm font-semibold text-foreground">
              {col.title}
            </p>
            <ul className="space-y-2 text-sm">
              {col.links.map((link) => (
                <li key={link.label}>
                  {link.href.startsWith('http') ? (
                    <a
                      href={link.href}
                      target="_blank"
                      rel="noreferrer"
                      className="text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {link.label}
                    </a>
                  ) : (
                    <Link
                      href={link.href}
                      className="text-muted-foreground transition-colors hover:text-foreground"
                    >
                      {link.label}
                    </Link>
                  )}
                </li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <div className="border-t border-border">
        <div className="mx-auto flex max-w-[100rem] flex-col gap-2 px-4 py-6 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-6">
          <p>© {new Date().getFullYear()} AgileFactory. Apache-2.0 Licensed.</p>
          <p className="font-mono">github.com/agile-factory/kernel</p>
        </div>
      </div>
    </footer>
  )
}
