import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  CAPABILITY_CATALOG,
  CAPABILITY_CATEGORY_LABELS,
  CAPABILITY_CATEGORY_SHORT,
  catalogRoutePath,
  categoryOrder,
  type CapabilityCategory,
  type CatalogCapability,
  type SubscriptionStatusLabel
} from '../config/capabilityCatalog'
import {
  createCheckoutSession,
  getBillingCapabilities,
  type BillingCapabilities,
  type BillingCapabilityFeature,
  type BillingPackageRow
} from '../api/billing'

export interface ResolvedCapability extends CatalogCapability {
  status: SubscriptionStatusLabel
  can_open: boolean
  can_subscribe: boolean
  stripe_ready: boolean
  pack_display_name?: string
  open_path: string
}

export interface CapabilitySummary {
  entitled: number
  subscribeable: number
  free: number
  total: number
}

export interface PackWithCapabilities {
  pack_key: string
  display_name: string
  entitled: boolean
  stripe_ready: boolean
  capabilities: ResolvedCapability[]
  status: SubscriptionStatusLabel
  can_subscribe: boolean
}

export type StatusFilter = 'all' | 'entitled' | 'unsubscribed' | 'free'

const ENTITLED_STATUSES: SubscriptionStatusLabel[] = ['已订阅', '免费可用', '管理员已开通']

const capsCache = ref<BillingCapabilities | null>(null)
const capsLoading = ref(false)
let capsLoadPromise: Promise<void> | null = null

function userRole(): string {
  try {
    return String((JSON.parse(localStorage.getItem('userInfo') || '{}') as { role?: string }).role ?? '')
  } catch {
    return ''
  }
}

export function resolveSubscriptionStatus(
  item: CatalogCapability,
  caps: BillingCapabilities | null,
  role: string
): { status: SubscriptionStatusLabel; can_open: boolean; can_subscribe: boolean; stripe_ready: boolean } {
  if (item.super_admin_only) {
    const ok = role === 'super_admin'
    return { status: ok ? '管理员已开通' : '暂不可用', can_open: ok, can_subscribe: false, stripe_ready: false }
  }
  if (item.always_free) {
    return { status: '免费可用', can_open: true, can_subscribe: false, stripe_ready: false }
  }
  if (!caps) {
    return { status: '暂不可用', can_open: false, can_subscribe: false, stripe_ready: false }
  }
  if (caps.billing_exempt || role === 'super_admin') {
    return { status: '管理员已开通', can_open: true, can_subscribe: false, stripe_ready: true }
  }

  const packKey = item.pack_key || ''
  const pkg = (caps.packages || []).find((p) => p.pack_key === packKey)
  const stripeReady = pkg?.stripe_ready === true

  let feature: BillingCapabilityFeature | undefined
  if (item.feature_key) {
    feature = (caps.features || []).find((f) => f.feature_key === item.feature_key)
  }

  if (pkg?.entitled) {
    return { status: '已订阅', can_open: true, can_subscribe: false, stripe_ready: stripeReady }
  }
  if (feature?.can_execute) {
    return { status: feature.billing_enabled ? '已订阅' : '免费可用', can_open: true, can_subscribe: false, stripe_ready: stripeReady }
  }
  if (feature && !feature.billing_enabled) {
    return { status: '免费可用', can_open: feature.can_view !== false, can_subscribe: false, stripe_ready: false }
  }
  if (feature && !feature.visible_enabled) {
    return { status: '暂不可用', can_open: false, can_subscribe: false, stripe_ready: stripeReady }
  }
  if (!stripeReady && feature?.billing_enabled) {
    return { status: '联系管理员开通', can_open: feature.can_view !== false, can_subscribe: false, stripe_ready: false }
  }
  return {
    status: '未订阅',
    can_open: feature?.can_view !== false,
    can_subscribe: stripeReady && feature?.billing_enabled === true,
    stripe_ready: stripeReady
  }
}

function resolveItem(item: CatalogCapability, caps: BillingCapabilities | null, prefix: string, role: string): ResolvedCapability {
  const sub = resolveSubscriptionStatus(item, caps, role)
  const pkg = (caps?.packages || []).find((p: BillingPackageRow) => p.pack_key === item.pack_key)
  const routePath = catalogRoutePath(item)
  return {
    ...item,
    ...sub,
    pack_display_name: pkg?.display_name || item.pack_key,
    open_path: routePath ? `${prefix}${routePath.startsWith('/') ? routePath : '/' + routePath}` : ''
  }
}

export function useCapabilityCatalog() {
  const route = useRoute()
  const shellPrefix = computed(() => (route.path.startsWith('/admin') ? '/admin' : '/app'))
  const role = computed(() => userRole())

  const load = async (force = false) => {
    if (capsCache.value && !force) return
    if (capsLoadPromise && !force) {
      await capsLoadPromise
      return
    }
    capsLoading.value = true
    capsLoadPromise = (async () => {
      try {
        capsCache.value = await getBillingCapabilities()
      } catch {
        capsCache.value = null
      } finally {
        capsLoading.value = false
        capsLoadPromise = null
      }
    })()
    await capsLoadPromise
  }

  const loading = computed(() => capsLoading.value)
  const caps = computed(() => capsCache.value)

  const resolved = computed<ResolvedCapability[]>(() => {
    const prefix = shellPrefix.value
    return CAPABILITY_CATALOG.filter((item) => {
      if (item.super_admin_only && role.value !== 'super_admin') return false
      return true
    }).map((item) => resolveItem(item, capsCache.value, prefix, role.value))
  })

  const deliveryCapabilities = computed(() => resolved.value.filter((c) => c.category === 'delivery'))

  const summary = computed<CapabilitySummary>(() => {
    const list = resolved.value
    return {
      entitled: list.filter((c) => ENTITLED_STATUSES.includes(c.status)).length,
      subscribeable: list.filter((c) => c.can_subscribe).length,
      free: list.filter((c) => c.status === '免费可用').length,
      total: list.length
    }
  })

  const packsWithCapabilities = computed<PackWithCapabilities[]>(() => {
    const packages = capsCache.value?.packages || []
    const byPack = new Map<string, ResolvedCapability[]>()
    for (const item of resolved.value) {
      if (!item.pack_key) continue
      const list = byPack.get(item.pack_key) || []
      list.push(item)
      byPack.set(item.pack_key, list)
    }
    const seen = new Set<string>()
    const out: PackWithCapabilities[] = []
    for (const pkg of packages) {
      const key = pkg.pack_key || ''
      if (!key || seen.has(key)) continue
      seen.add(key)
      const capabilities = byPack.get(key) || []
      if (!capabilities.length) continue
      const entitled = pkg.entitled === true || capabilities.every((c) => ENTITLED_STATUSES.includes(c.status))
      const canSubscribe = !entitled && pkg.stripe_ready === true && capabilities.some((c) => c.can_subscribe)
      let status: SubscriptionStatusLabel = '未订阅'
      if (entitled) status = '已订阅'
      else if (capabilities.some((c) => c.status === '免费可用')) status = '免费可用'
      else if (!pkg.stripe_ready) status = '联系管理员开通'
      out.push({
        pack_key: key,
        display_name: pkg.display_name || key,
        entitled,
        stripe_ready: pkg.stripe_ready === true,
        capabilities,
        status,
        can_subscribe: canSubscribe
      })
    }
    for (const [key, capabilities] of byPack) {
      if (seen.has(key)) continue
      out.push({
        pack_key: key,
        display_name: capabilities[0]?.pack_display_name || key,
        entitled: capabilities.every((c) => ENTITLED_STATUSES.includes(c.status)),
        stripe_ready: false,
        capabilities,
        status: capabilities[0]?.status || '未订阅',
        can_subscribe: capabilities.some((c) => c.can_subscribe)
      })
    }
    return out.sort((a, b) => Number(b.entitled) - Number(a.entitled))
  })

  const byCategory = computed(() => {
    const map = new Map<CapabilityCategory, ResolvedCapability[]>()
    for (const cat of categoryOrder()) {
      map.set(cat, [])
    }
    for (const item of resolved.value) {
      const list = map.get(item.category) || []
      list.push(item)
      map.set(item.category, list)
    }
    return map
  })

  const filterCapabilities = (opts: { q?: string; status?: StatusFilter; category?: CapabilityCategory | 'all' }) => {
    let list = resolved.value
    const q = (opts.q || '').trim().toLowerCase()
    if (q) {
      list = list.filter((item) => {
        const hay = [item.name, item.description, ...(item.keywords || []), item.pack_key || '', item.cli_topic || '']
          .join(' ')
          .toLowerCase()
        return hay.includes(q)
      })
    }
    if (opts.status && opts.status !== 'all') {
      if (opts.status === 'entitled') {
        list = list.filter((c) => ENTITLED_STATUSES.includes(c.status))
      } else if (opts.status === 'unsubscribed') {
        list = list.filter((c) => c.status === '未订阅' || c.status === '联系管理员开通')
      } else if (opts.status === 'free') {
        list = list.filter((c) => c.status === '免费可用')
      }
    }
    if (opts.category && opts.category !== 'all') {
      list = list.filter((c) => c.category === opts.category)
    }
    return list
  }

  const subscribe = async (item: Pick<ResolvedCapability, 'pack_key' | 'can_subscribe'>) => {
    const key = item.pack_key
    if (!key) {
      ElMessage.info('该能力无需单独订阅')
      return
    }
    if (!item.can_subscribe) {
      ElMessage.info('请联系管理员开通此能力')
      return
    }
    try {
      const resp = await createCheckoutSession({ pack_key: key })
      if (resp?.url) {
        window.location.href = resp.url
      }
    } catch {
      ElMessage.info('暂无法在线订阅，请联系管理员开通')
    }
  }

  return {
    loading,
    caps,
    resolved,
    deliveryCapabilities,
    summary,
    packsWithCapabilities,
    byCategory,
    categoryLabels: CAPABILITY_CATEGORY_LABELS,
    categoryShort: CAPABILITY_CATEGORY_SHORT,
    categoryOrder,
    shellPrefix,
    load,
    subscribe,
    filterCapabilities,
    isEntitledStatus: (s: SubscriptionStatusLabel) => ENTITLED_STATUSES.includes(s)
  }
}
