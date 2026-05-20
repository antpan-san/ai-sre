import { computed, ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  CAPABILITY_CATALOG,
  CAPABILITY_CATEGORY_LABELS,
  categoryOrder,
  type CatalogCapability,
  type CapabilityCategory,
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

export function useCapabilityCatalog() {
  const route = useRoute()
  const loading = ref(false)
  const caps = ref<BillingCapabilities | null>(null)
  const shellPrefix = computed(() => (route.path.startsWith('/admin') ? '/admin' : '/app'))
  const role = computed(() => userRole())

  const load = async () => {
    loading.value = true
    try {
      caps.value = await getBillingCapabilities()
    } catch {
      caps.value = null
    } finally {
      loading.value = false
    }
  }

  const resolved = computed<ResolvedCapability[]>(() => {
    const prefix = shellPrefix.value
    const packages = caps.value?.packages || []
    return CAPABILITY_CATALOG.filter((item) => {
      if (item.super_admin_only && role.value !== 'super_admin') return false
      return true
    }).map((item) => {
      const sub = resolveSubscriptionStatus(item, caps.value, role.value)
      const pkg = packages.find((p: BillingPackageRow) => p.pack_key === item.pack_key)
      return {
        ...item,
        ...sub,
        pack_display_name: pkg?.display_name || item.pack_key,
        open_path: `${prefix}${item.route_suffix || ''}`
      }
    })
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

  const subscribe = async (item: ResolvedCapability) => {
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

  onMounted(() => {
    void load()
  })

  return {
    loading,
    caps,
    resolved,
    byCategory,
    categoryLabels: CAPABILITY_CATEGORY_LABELS,
    categoryOrder,
    shellPrefix,
    load,
    subscribe
  }
}
