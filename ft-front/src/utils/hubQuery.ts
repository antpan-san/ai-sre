import type { RouteLocationNormalizedLoaded } from 'vue-router'

export const ZONE_TO_CAP: Record<string, string> = {
  k8s: 'k8s_delivery',
  services: 'service_deploy',
  linux: 'linux_hosts',
  init: 'init_tools',
  mirror: 'k8s_mirror'
}

export function parseHubCapId(route: RouteLocationNormalizedLoaded): string {
  const cap = String(route.query.cap || '').trim()
  if (cap) return cap
  const zone = String(route.query.zone || '').trim()
  if (!zone) return ''
  return ZONE_TO_CAP[zone] || (zone.includes('_') ? zone : '')
}

export function shouldExpandSubscribe(route: RouteLocationNormalizedLoaded): boolean {
  return String(route.query.expand || '').trim() === 'subscribe'
}

export const CAP_TO_DEPLOY_HASH: Record<string, string> = {
  k8s_delivery: 'cluster',
  service_deploy: 'services',
  linux_hosts: 'cluster',
  init_tools: 'init-tools',
  k8s_mirror: 'cluster',
}

export function hubDeployPath(cap?: string, expandSubscribe = false): string {
  const query: Record<string, string> = {}
  if (cap) query.cap = cap
  if (expandSubscribe) query.expand = 'subscribe'
  const qs = new URLSearchParams(query).toString()
  const hash = cap ? CAP_TO_DEPLOY_HASH[cap] : ''
  const base = qs ? `/app/workloads?${qs}` : '/app/workloads'
  return hash ? `${base}#${hash}` : base
}

/** @deprecated use hubDeployPath */
export function hubWorkloadsPath(_tab?: string, cap?: string): string {
  return hubDeployPath(cap)
}
