import DOMPurify from 'dompurify'
import { marked } from 'marked'

const MD_OPTS: Parameters<typeof marked.use>[0] = {
  gfm: true,
  breaks: true,
}

let configured = false
function ensureMarked() {
  if (configured) return
  marked.use(MD_OPTS)
  configured = true
}

/** User-facing markdown (AI / diagnose): sanitized HTML for v-html. */
export function markdownToSafeHtml(source: string | null | undefined): string {
  const raw = (source ?? '').trim()
  if (!raw) return ''
  ensureMarked()
  const html = marked.parse(raw, { async: false }) as string
  return DOMPurify.sanitize(html, {
    USE_PROFILES: { html: true },
  })
}
