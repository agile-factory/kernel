'use client'

import Fuse from 'fuse.js'
import { CornerDownLeft, FileText, Search } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useEffect, useMemo, useRef, useState } from 'react'
import { searchIndex } from '@/lib/navigation'
import { cn } from '@/lib/utils'

export function SearchTrigger() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setOpen((v) => !v)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="inline-flex h-9 w-full items-center gap-2 rounded-md border border-border bg-background px-3 text-sm text-muted-foreground transition-colors hover:border-primary/40 hover:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring sm:w-56"
      >
        <Search className="size-4 shrink-0" aria-hidden="true" />
        <span className="flex-1 text-left">Search docs...</span>
        <kbd className="hidden items-center gap-0.5 rounded border border-border bg-muted px-1.5 font-mono text-[10px] text-muted-foreground sm:inline-flex">
          ⌘K
        </kbd>
      </button>
      {open ? <SearchModal onClose={() => setOpen(false)} /> : null}
    </>
  )
}

function SearchModal({ onClose }: { onClose: () => void }) {
  const router = useRouter()
  const [query, setQuery] = useState('')
  const [active, setActive] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)

  const fuse = useMemo(
    () =>
      new Fuse(searchIndex, {
        keys: ['title', 'section'],
        threshold: 0.4,
      }),
    [],
  )

  const results = useMemo(() => {
    if (!query.trim()) return searchIndex.slice(0, 8)
    return fuse.search(query).map((r) => r.item)
  }, [query, fuse])

  // Group results by section
  const grouped = useMemo(() => {
    const map = new Map<string, typeof results>()
    for (const item of results) {
      const list = map.get(item.section) ?? []
      list.push(item)
      map.set(item.section, list)
    }
    return map
  }, [results])

  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  useEffect(() => {
    setActive(0)
  }, [query])

  function go(href: string) {
    router.push(href)
    onClose()
  }

  function onKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Escape') onClose()
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setActive((a) => Math.min(a + 1, results.length - 1))
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setActive((a) => Math.max(a - 1, 0))
    }
    if (e.key === 'Enter' && results[active]) {
      e.preventDefault()
      go(results[active].href)
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center p-4 pt-[12vh]"
      role="dialog"
      aria-modal="true"
      aria-label="Search documentation"
    >
      <button
        type="button"
        aria-label="Close search"
        className="fixed inset-0 cursor-default bg-background/70 backdrop-blur-sm"
        onClick={onClose}
      />
      <div
        className="relative z-10 w-full max-w-lg overflow-hidden rounded-xl border border-border bg-popover shadow-2xl shadow-black/20"
        onKeyDown={onKeyDown}
      >
        <div className="flex items-center gap-2 border-b border-border px-4">
          <Search className="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Search documentation..."
            className="h-12 flex-1 bg-transparent text-sm text-foreground outline-none placeholder:text-muted-foreground"
          />
          <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground">
            ESC
          </kbd>
        </div>
        <div className="max-h-80 overflow-y-auto p-2 scrollbar-thin">
          {results.length === 0 ? (
            <div className="px-3 py-8 text-center">
              <p className="text-sm text-muted-foreground">
                No results for{' '}
                <span className="font-medium text-foreground">
                  &ldquo;{query}&rdquo;
                </span>
              </p>
              <p className="mt-1 text-xs text-muted-foreground">
                Try searching for a module name or concept
              </p>
            </div>
          ) : (
            Array.from(grouped.entries()).map(([section, items]) => (
              <div key={section} className="mb-2 last:mb-0">
                <p className="mb-1 px-3 py-1 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                  {section}
                </p>
                {items.map((item) => {
                  const flatIdx = results.indexOf(item)
                  return (
                    <button
                      key={item.href}
                      type="button"
                      onMouseEnter={() => setActive(flatIdx)}
                      onClick={() => go(item.href)}
                      className={cn(
                        'flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors',
                        active === flatIdx
                          ? 'bg-accent text-accent-foreground'
                          : 'text-foreground hover:bg-accent/50',
                      )}
                    >
                      <FileText
                        className="size-4 shrink-0 text-muted-foreground"
                        aria-hidden="true"
                      />
                      <span className="flex-1">{item.title}</span>
                      {active === flatIdx ? (
                        <CornerDownLeft
                          className="size-3.5 shrink-0 text-muted-foreground"
                          aria-hidden="true"
                        />
                      ) : null}
                    </button>
                  )
                })}
              </div>
            ))
          )}
        </div>
        {results.length > 0 && (
          <div className="border-t border-border px-4 py-2">
            <p className="text-[10px] text-muted-foreground">
              <kbd className="rounded border border-border bg-muted px-1 font-mono">
                ↑↓
              </kbd>{' '}
              navigate{' '}
              <kbd className="rounded border border-border bg-muted px-1 font-mono">
                ↵
              </kbd>{' '}
              open{' '}
              <kbd className="rounded border border-border bg-muted px-1 font-mono">
                esc
              </kbd>{' '}
              close
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
