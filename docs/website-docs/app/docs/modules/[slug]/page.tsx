import type { Metadata } from 'next'
import { notFound } from 'next/navigation'
import { ModulePage } from '@/components/docs/module-page'
import { getModule, modules } from '@/lib/modules-data'
import { moduleList } from '@/lib/navigation'

export function generateStaticParams() {
  return Object.keys(modules).map((slug) => ({ slug }))
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>
}): Promise<Metadata> {
  const { slug } = await params
  const mod = getModule(slug)
  if (!mod) return { title: 'Module not found' }
  return {
    title: `${mod.title} module`,
    description: mod.description,
  }
}

export default async function ModuleRoute({
  params,
}: {
  params: Promise<{ slug: string }>
}) {
  const { slug } = await params
  const mod = getModule(slug)
  if (!mod) notFound()

  const index = moduleList.findIndex((m) => m.slug === slug)
  const prevMod = index > 0 ? moduleList[index - 1] : undefined
  const nextMod =
    index >= 0 && index < moduleList.length - 1
      ? moduleList[index + 1]
      : undefined

  return (
    <ModulePage
      mod={mod}
      prev={
        prevMod
          ? { title: `${prevMod.title} module`, href: `/docs/modules/${prevMod.slug}` }
          : { title: 'Getting Started', href: '/docs/getting-started' }
      }
      next={
        nextMod
          ? { title: `${nextMod.title} module`, href: `/docs/modules/${nextMod.slug}` }
          : { title: 'Architecture (ADRs)', href: '/docs/architecture' }
      }
    />
  )
}
