import type { RouteLocationNormalizedLoaded } from 'vue-router'
import type { CapabilityCategory } from '../config/capabilityCatalog'
import { HUB_CATEGORY_ORDER } from '../config/capabilityCatalog'

export const ZONE_TO_CAP: Record<string, string> = {
  k8s: 'k8s_delivery',
  services: 'service_deploy',
  linux: 'linux_hosts',
  init: 'init_tools',
  proxy: 'proxy',
  mirror: 'k8s_mirror'
}

const VALID_SECTIONS = new Set<string>(HUB_CATEGORY_ORDER)

export function parseHubSection(raw: unknown): CapabilityCategory {
  const s = String(raw || '').trim()
  if (VALID_SECTIONS.has(s)) return s as CapabilityCategory
  return 'delivery'
}

export function parseHubCapId(route: RouteLocationNormalizedLoaded): string {
  const cap = String(route.query.cap || '').trim()
  if (cap) return cap
  const zone = String(route.query.zone || '').trim()
  if (!zone) return ''
  return ZONE_TO_CAP[zone] || (zone.includes('_') ? zone : '')
}

export function hubWorkloadsPath(section: CapabilityCategory, cap?: string): string {
  const query: Record<string, string> = { section }
  if (cap) query.cap = cap
  const qs = new URLSearchParams(query).toString()
  return `/app/workloads?${qs}`
}
