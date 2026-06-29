export type NavLink = {
  title: string
  href: string
  badge?: string
}

export type NavGroup = {
  title: string
  items: NavLink[]
}

export const moduleList: { slug: string; title: string; summary: string }[] = [
  {
    slug: 'auth',
    title: 'auth',
    summary: 'Owner signup, login, refresh, logout, email verification, invites.',
  },
  {
    slug: 'tenant',
    title: 'tenant',
    summary: 'Tenant CRUD and per-tenant settings.',
  },
  {
    slug: 'membership',
    title: 'membership',
    summary: 'Invite, revoke, and list members of a tenant.',
  },
  {
    slug: 'audit',
    title: 'audit',
    summary: 'Append-only event logging with cursor pagination and outbox delivery.',
  },
  {
    slug: 'billing',
    title: 'billing',
    summary: 'Stripe webhooks, Wave stub, plans, and subscriptions.',
  },
  {
    slug: 'notifications',
    title: 'notifications',
    summary: 'Event-driven email/SMS dispatch with template overrides.',
  },
  {
    slug: 'user',
    title: 'user',
    summary: 'Lazy user-profile creation with GET/PATCH.',
  },
  {
    slug: 'storage',
    title: 'storage',
    summary: 'S3-compatible object storage (R2, MinIO, AWS, B2).',
  },
  {
    slug: 'end-customers',
    title: 'end-customers',
    summary: 'B2B2C end-customer CRUD with suspend/restore.',
  },
  {
    slug: 'white-label',
    title: 'white-label',
    summary: 'Per-tenant branding and custom domain DNS.',
  },
  {
    slug: 'customer-auth',
    title: 'customer-auth',
    summary: 'End-customer register/login/OTP reset with aud:"customer".',
  },
  {
    slug: 'customer-notifications',
    title: 'customer-notifications',
    summary: 'End-customer email + SMS with tenant overrides.',
  },
]

export const sidebar: NavGroup[] = [
  {
    title: 'Getting Started',
    items: [
      { title: 'Introduction', href: '/docs' },
      { title: 'Getting Started', href: '/docs/getting-started' },
    ],
  },
  {
    title: 'Core Modules',
    items: [
      { title: 'auth', href: '/docs/modules/auth' },
      { title: 'tenant', href: '/docs/modules/tenant' },
      { title: 'membership', href: '/docs/modules/membership' },
      { title: 'audit', href: '/docs/modules/audit' },
      { title: 'billing', href: '/docs/modules/billing' },
      { title: 'notifications', href: '/docs/modules/notifications' },
      { title: 'user', href: '/docs/modules/user' },
      { title: 'storage', href: '/docs/modules/storage' },
    ],
  },
  {
    title: 'B2B2C Modules',
    items: [
      { title: 'end-customers', href: '/docs/modules/end-customers' },
      { title: 'white-label', href: '/docs/modules/white-label' },
      { title: 'customer-auth', href: '/docs/modules/customer-auth' },
      {
        title: 'customer-notifications',
        href: '/docs/modules/customer-notifications',
      },
    ],
  },
  {
    title: 'Guides',
    items: [
      { title: 'Building a Custom Module', href: '/docs/guides/custom-module', badge: 'New' },
    ],
  },
  {
    title: 'Reference',
    items: [
      { title: 'Architecture (ADRs)', href: '/docs/architecture' },
      { title: 'RFCs', href: '/docs/rfcs' },
      { title: 'Changelog', href: '/changelog' },
    ],
  },
]

/** Flat list of every searchable page for the ⌘K command palette. */
export const searchIndex: { title: string; href: string; section: string }[] = [
  { title: 'Introduction', href: '/docs', section: 'Getting Started' },
  {
    title: 'Getting Started',
    href: '/docs/getting-started',
    section: 'Getting Started',
  },
  ...moduleList.map((m) => ({
    title: `${m.title} module`,
    href: `/docs/modules/${m.slug}`,
    section: 'Modules',
  })),
  {
    title: 'Architecture (ADRs)',
    href: '/docs/architecture',
    section: 'Reference',
  },
  { title: 'Building a Custom Module', href: '/docs/guides/custom-module', section: 'Guides' },
  { title: 'RFCs', href: '/docs/rfcs', section: 'Reference' },
  { title: 'Changelog', href: '/changelog', section: 'Reference' },
]
