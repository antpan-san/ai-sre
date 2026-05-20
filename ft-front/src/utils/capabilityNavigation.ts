import type { Router } from 'vue-router'
import type { ResolvedCapability } from '../composables/useCapabilityCatalog'

export function openCapability(router: Router, item: ResolvedCapability) {
  if (!item.can_open || !item.open_path) return
  const [path, queryStr] = item.open_path.split('?')
  const query: Record<string, string> = {}
  if (queryStr) {
    for (const part of queryStr.split('&')) {
      const [k, v] = part.split('=')
      if (k) query[k] = decodeURIComponent(v || '')
    }
  }
  void router.push({ path, query: Object.keys(query).length ? query : undefined })
}
