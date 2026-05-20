import type { RouteLocationNormalizedLoaded } from 'vue-router'
import {
  CAPABILITY_CATALOG,
  HUB_CATEGORY_ORDER,
  type CapabilityCategory
} from '../config/capabilityCatalog'

export const ZONE_TO_CAP: Record<string, string> = {
  k8s: 'k8s_delivery',
  services: 'service_deploy',
  linux: 'linux_hosts',
  init: 'init_tools',
  proxy: 'proxy',
  mirror: 'k8s_mirror'
}

export type HubTab = 'overview' | 'packs' | CapabilityCategory

const VALID_SECTIONS = new Set<string>(HUB_CATEGORY_ORDER)

export function parseHubSection(raw: unknown): CapabilityCategory {
  const s = String(raw || '').trim()
  if (VALID_SECTIONS.has(s)) return s as CapabilityCategory
  return 'delivery'
}

export function categoryForCapId(capId: string): CapabilityCategory | null {
  if (!capId) return null
  const item = CAPABILITY_CATALOG.find((c) => c.id === capId)
  return item?.category ?? null
}

export function parseHubCapId(route: RouteLocationNormalizedLoaded): string {
  const cap = String(route.query.cap || '').trim()
  if (cap) return cap
  const zone = String(route.query.zone || '').trim()
  if (!zone) return ''
  return ZONE_TO_CAP[zone] || (zone.includes('_') ? zone : '')
}

export function parseHubTab(route: RouteLocationNormalizedLoaded): HubTab {
  if (route.hash === '#packs') return 'packs'

  const tab = String(route.query.tab || '').trim()
  if (tab === 'overview' || tab === 'packs') return tab
  if (VALID_SECTIONS.has(tab)) return tab as CapabilityCategory

  const section = String(route.query.section || '').trim()
  if (VALID_SECTIONS.has(section)) return section as CapabilityCategory

  const capId = parseHubCapId(route)
  const cat = categoryForCapId(capId)
  if (cat) return cat

  return 'overview'
}

export function hubWorkloadsPath(tab: HubTab, cap?: string): string {
  const query: Record<string, string> = { tab }
  if (cap) query.cap = cap
  const qs = new URLSearchParams(query).toString()
  return `/app/workloads?${qs}`
}

export function isCapabilityTab(tab: HubTab): tab is CapabilityCategory {
  return tab !== 'overview' && tab !== 'packs'
}
