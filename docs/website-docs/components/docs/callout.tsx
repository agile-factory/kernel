import { AlertTriangle, Info, Lightbulb, OctagonAlert } from 'lucide-react'
import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

type Variant = 'info' | 'warning' | 'tip' | 'danger'

const config: Record<
  Variant,
  { icon: typeof Info; label: string; className: string; iconClass: string }
> = {
  info: {
    icon: Info,
    label: 'Note',
    className: 'border-primary/30 bg-primary/5',
    iconClass: 'text-primary',
  },
  tip: {
    icon: Lightbulb,
    label: 'Tip',
    className: 'border-emerald-500/30 bg-emerald-500/5',
    iconClass: 'text-emerald-500',
  },
  warning: {
    icon: AlertTriangle,
    label: 'Warning',
    className: 'border-amber-500/30 bg-amber-500/5',
    iconClass: 'text-amber-500',
  },
  danger: {
    icon: OctagonAlert,
    label: 'Danger',
    className: 'border-destructive/40 bg-destructive/5',
    iconClass: 'text-destructive',
  },
}

export function Callout({
  variant = 'info',
  title,
  children,
}: {
  variant?: Variant
  title?: string
  children: ReactNode
}) {
  const { icon: Icon, label, className, iconClass } = config[variant]

  return (
    <div
      className={cn(
        'my-6 flex gap-3 rounded-xl border px-4 py-3 text-sm leading-relaxed',
        className,
      )}
      role="note"
    >
      <Icon className={cn('mt-0.5 size-4 shrink-0', iconClass)} aria-hidden="true" />
      <div className="min-w-0 [&>p]:m-0 [&>p+p]:mt-2">
        <p className="mb-1 font-semibold text-foreground">{title ?? label}</p>
        <div className="text-muted-foreground [&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[0.85em] [&_code]:text-foreground">
          {children}
        </div>
      </div>
    </div>
  )
}
