/** Resolve a human-readable execution target for list/detail views. */
export function displayExecutionTarget(row: Record<string, unknown> | null | undefined): string {
  if (!row) return '-'
  const host = String(row.target_host ?? row.targetHost ?? '').trim()
  if (host) return host
  const resource = String(row.resource_name ?? row.resourceName ?? '').trim()
  if (resource) return resource
  const meta = recordMeta(row)
  const direct = String(meta.diagnosis_target ?? '').trim()
  if (direct) return direct
  const ctx = meta.context
  if (ctx && typeof ctx === 'object') {
    const c = ctx as Record<string, unknown>
    const fromCtx = String(c.diagnosis_target ?? '').trim()
    if (fromCtx) return fromCtx
    const scalars = c.scalars
    if (scalars && typeof scalars === 'object') {
      for (const key of ['addr', 'target', 'domain', 'bootstrap', 'url', 'dsn', 'host']) {
        const v = String((scalars as Record<string, unknown>)[key] ?? '').trim()
        if (v) return v
      }
    }
  }
  const cmd = String(row.command ?? '').trim()
  const m = cmd.match(/\b(?:check|probe|analyze)\s+\S+\s+(\S+)/)
  if (m?.[1]) return m[1]
  return '-'
}

function recordMeta(row: Record<string, unknown>): Record<string, unknown> {
  const m = row.metadata
  return m && typeof m === 'object' ? (m as Record<string, unknown>) : {}
}
