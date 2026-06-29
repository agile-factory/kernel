import type { ReactNode } from 'react'
import { SidebarNav } from '@/components/docs/sidebar-nav'
import { TopBar } from '@/components/docs/top-bar'

export default function DocsLayout({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-background">
      <TopBar />
      <div className="mx-auto flex max-w-[100rem] px-4 sm:px-6">
        <aside className="sticky top-16 hidden h-[calc(100vh-4rem)] w-64 shrink-0 overflow-y-auto py-8 pr-4 scrollbar-thin lg:block">
          <SidebarNav />
        </aside>
        <main className="min-w-0 flex-1">{children}</main>
      </div>
    </div>
  )
}
