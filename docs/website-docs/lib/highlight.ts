import 'server-only'
import { codeToHtml } from 'shiki'

export type SupportedLang = 'go' | 'sql' | 'json' | 'bash' | 'ts' | 'tsx'

/**
 * Server-side syntax highlighting using Shiki with dual themes.
 * The light/dark swap is handled in globals.css via CSS variables.
 */
export async function highlight(
  code: string,
  lang: SupportedLang = 'go',
): Promise<string> {
  return codeToHtml(code.trim(), {
    lang,
    themes: {
      light: 'github-light',
      dark: 'github-dark',
    },
    defaultColor: false,
  })
}
