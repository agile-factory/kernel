import Link from 'next/link'
import {
  ArrowRight,
  Bell,
  Building2,
  CreditCard,
  KeyRound,
  ScrollText,
  Users,
} from 'lucide-react'

const features = [
  {
    icon: KeyRound,
    title: 'Authentication',
    href: '/docs/modules/auth',
    description:
      'Owner signup, login, refresh, logout, email verification, and invite acceptance — JWT-based with rotating refresh tokens.',
  },
  {
    icon: Building2,
    title: 'Multi-Tenancy',
    href: '/docs/modules/tenant',
    description:
      'First-class tenant isolation with per-tenant settings, membership management, and zero global state across requests.',
  },
  {
    icon: ScrollText,
    title: 'Audit Trails',
    href: '/docs/modules/audit',
    description:
      'Append-only event logging with v1 offset and v2 cursor pagination, plus at-least-once outbox delivery.',
  },
  {
    icon: CreditCard,
    title: 'Billing',
    href: '/docs/modules/billing',
    description:
      'Stripe webhook handling, a Wave stub for African rails, plans, and subscription lifecycle management.',
  },
  {
    icon: Bell,
    title: 'Notifications',
    href: '/docs/modules/notifications',
    description:
      'Event-driven email and SMS dispatch with per-tenant template overrides and pluggable providers.',
  },
  {
    icon: Users,
    title: 'B2B2C End-Customers',
    href: '/docs/modules/end-customers',
    description:
      'Your tenants get their own customers: CRUD, suspend/restore, customer-auth, and white-label domains.',
  },
]

export function FeatureGrid() {
  return (
    <section className="mx-auto max-w-[100rem] px-4 py-20 sm:px-6">
      <div className="mx-auto max-w-2xl text-center">
        <h2 className="text-balance text-3xl font-bold tracking-tight text-foreground sm:text-4xl">
          Every B2B SaaS primitive, composable
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-pretty leading-relaxed text-muted-foreground">
          Wire only what you need. Each module ships its own migrations, config
          struct, HTTP handlers, and events.
        </p>
      </div>

      <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {features.map((feature) => (
          <Link
            key={feature.title}
            href={feature.href}
            className="group flex flex-col gap-3 rounded-xl border border-border bg-card p-6 transition-all duration-200 hover:-translate-y-1 hover:border-primary/40 hover:shadow-lg hover:shadow-primary/5"
          >
            <span className="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary ring-1 ring-primary/20 transition-all group-hover:bg-primary/15">
              <feature.icon className="size-5" aria-hidden="true" />
            </span>
            <h3 className="font-semibold text-foreground group-hover:text-primary">
              {feature.title}
            </h3>
            <p className="flex-1 text-sm leading-relaxed text-muted-foreground">
              {feature.description}
            </p>
            <span className="flex items-center gap-1 text-xs font-medium text-primary opacity-0 transition-opacity group-hover:opacity-100">
              Explore module
              <ArrowRight className="size-3" aria-hidden="true" />
            </span>
          </Link>
        ))}
      </div>
    </section>
  )
}
