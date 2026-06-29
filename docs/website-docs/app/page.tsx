import { TopBar } from '@/components/docs/top-bar'
import { FeatureGrid } from '@/components/home/feature-grid'
import { Hero } from '@/components/home/hero'
import { QuickStart } from '@/components/home/quick-start'
import { SiteFooter } from '@/components/site-footer'

export default function HomePage() {
  return (
    <div className="min-h-screen bg-background">
      <TopBar />
      <main>
        <Hero />
        <FeatureGrid />
        <QuickStart />
      </main>
      <SiteFooter />
    </div>
  )
}
