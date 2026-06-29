'use client'

import Link from 'next/link'
import { Menu, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { GitHubIcon, Logo } from '@/components/logo'
import { SearchTrigger } from '@/components/search-dialog'
import { ThemeToggle } from '@/components/theme-toggle'
import { SidebarNav } from './sidebar-nav'

const NAV_LINKS = [
  { href: '/docs', label: 'Docs' },
  { href: '/docs/modules/auth', label: 'Modules' },
  { href: '/docs/architecture', label: 'Architecture' },
  { href: '/changelog', label: 'Changelog' },
]

export function TopBar() {
  const [drawerOpen, setDrawerOpen] = useState(false)

  useEffect(() => {
    document.body.style.overflow = drawerOpen ? 'hidden' : ''
    return () => {
      document.body.style.overflow = ''
    }
  }, [drawerOpen])

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-[100rem] items-center gap-4 px-4 sm:px-6">
        <button
          type="button"
          onClick={() => setDrawerOpen(true)}
          aria-label="Open navigation menu"
          className="inline-flex size-9 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground lg:hidden"
        >
          <Menu className="size-4" aria-hidden="true" />
        </button>

        <Logo />

        {/* Desktop nav links */}
        <nav className="ml-4 hidden items-center gap-1 lg:flex" aria-label="Site navigation">
          {NAV_LINKS.map(({ href, label }) => (
            <Link
              key={href}
              href={href}
              className="rounded-md px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
            >
              {label}
            </Link>
          ))}
        </nav>

        <div className="ml-auto flex items-center gap-2">
          <div className="hidden sm:block">
            <SearchTrigger />
          </div>
          <a
            href="https://github.com/agile-factory/kernel"
            target="_blank"
            rel="noreferrer"
            aria-label="GitHub repository"
            className="inline-flex size-9 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <GitHubIcon className="size-4" />
          </a>
          <ThemeToggle />
        </div>
      </div>

      {/* Mobile search row */}
      <div className="border-t border-border px-4 py-2 sm:hidden">
        <SearchTrigger />
      </div>

      {/* Mobile drawer */}
      {drawerOpen ? (
        <div className="fixed inset-0 z-50 lg:hidden">
          <button
            type="button"
            aria-label="Close navigation menu"
            className="fixed inset-0 cursor-default bg-background/70 backdrop-blur-sm"
            onClick={() => setDrawerOpen(false)}
          />
          <div className="fixed inset-y-0 left-0 z-10 w-72 max-w-[85vw] overflow-y-auto border-r border-border bg-sidebar p-4 scrollbar-thin">
            <div className="mb-6 flex items-center justify-between">
              <Logo />
              <button
                type="button"
                onClick={() => setDrawerOpen(false)}
                aria-label="Close menu"
                className="inline-flex size-8 items-center justify-center rounded-md text-muted-foreground hover:bg-accent hover:text-foreground"
              >
                <X className="size-4" aria-hidden="true" />
              </button>
            </div>
            <SidebarNav onNavigate={() => setDrawerOpen(false)} />
          </div>
        </div>
      ) : null}
    </header>
  )
}
