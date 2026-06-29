import { Lock, LockOpen } from 'lucide-react'
import { cn } from '@/lib/utils'

type Method = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

const methodColor: Record<Method, string> = {
  GET: 'text-emerald-500 bg-emerald-500/10 border-emerald-500/30',
  POST: 'text-primary bg-primary/10 border-primary/30',
  PUT: 'text-amber-500 bg-amber-500/10 border-amber-500/30',
  PATCH: 'text-amber-500 bg-amber-500/10 border-amber-500/30',
  DELETE: 'text-destructive bg-destructive/10 border-destructive/30',
}

export type EndpointResponse = { status: string; body: string }

export function ApiEndpoint({
  method,
  path,
  auth,
  body,
  responses,
}: {
  method: Method
  path: string
  auth: string
  body?: string
  responses: EndpointResponse[]
}) {
  const isPublic = auth.toLowerCase() === 'none'

  return (
    <div className="my-6 overflow-hidden rounded-xl border border-border bg-card">
      <div className="flex flex-wrap items-center gap-3 border-b border-border bg-muted/30 px-4 py-3">
        <span
          className={cn(
            'rounded-md border px-2 py-0.5 font-mono text-xs font-semibold',
            methodColor[method],
          )}
        >
          {method}
        </span>
        <code className="font-mono text-sm text-foreground">{path}</code>
        <span
          className={cn(
            'ml-auto inline-flex items-center gap-1.5 rounded-md border border-border bg-background px-2 py-0.5 text-xs',
            isPublic ? 'text-muted-foreground' : 'text-foreground',
          )}
        >
          {isPublic ? (
            <LockOpen className="size-3" aria-hidden="true" />
          ) : (
            <Lock className="size-3 text-primary" aria-hidden="true" />
          )}
          <span className="font-medium">{auth}</span>
        </span>
      </div>

      <dl className="divide-y divide-border">
        {body ? (
          <div className="grid grid-cols-[5rem_1fr] gap-3 px-4 py-3 text-sm sm:grid-cols-[7rem_1fr]">
            <dt className="pt-0.5 font-medium text-muted-foreground">Body</dt>
            <dd>
              <code className="block whitespace-pre-wrap break-words font-mono text-xs text-foreground">
                {body}
              </code>
            </dd>
          </div>
        ) : null}
        {responses.map((r) => (
          <div
            key={r.status}
            className="grid grid-cols-[5rem_1fr] gap-3 px-4 py-3 text-sm sm:grid-cols-[7rem_1fr]"
          >
            <dt className="pt-0.5 font-mono font-medium text-muted-foreground">
              {r.status}
            </dt>
            <dd>
              <code className="block whitespace-pre-wrap break-words font-mono text-xs text-foreground">
                {r.body}
              </code>
            </dd>
          </div>
        ))}
      </dl>
    </div>
  )
}
