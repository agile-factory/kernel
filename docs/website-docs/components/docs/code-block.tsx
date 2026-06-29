import { highlight, type SupportedLang } from '@/lib/highlight'
import { cn } from '@/lib/utils'
import { CopyButton } from './copy-button'

const langLabels: Record<SupportedLang, string> = {
  go: 'Go',
  sql: 'SQL',
  json: 'JSON',
  bash: 'Shell',
  ts: 'TypeScript',
  tsx: 'TSX',
}

export async function CodeBlock({
  code,
  lang = 'go',
  filename,
  showLineNumbers = false,
  className,
}: {
  code: string
  lang?: SupportedLang
  filename?: string
  showLineNumbers?: boolean
  className?: string
}) {
  const html = await highlight(code, lang)

  return (
    <figure
      className={cn(
        'group relative my-6 overflow-hidden rounded-xl border border-border bg-card',
        className,
      )}
    >
      <figcaption className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-2">
        <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
          {filename ? (
            <span className="font-mono text-foreground">{filename}</span>
          ) : (
            <span className="rounded bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wide text-primary">
              {langLabels[lang]}
            </span>
          )}
        </div>
        <CopyButton value={code.trim()} />
      </figcaption>
      <div
        className={cn(
          'overflow-x-auto p-4 text-[13px] leading-relaxed scrollbar-thin [&_code]:font-mono [&_pre]:!bg-transparent',
          showLineNumbers && 'code-line-numbers',
        )}
        // Shiki output is generated server-side from trusted, static doc content.
        dangerouslySetInnerHTML={{ __html: html }}
      />
    </figure>
  )
}
