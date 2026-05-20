<template>
  <div class="workloads-hub page-shell page-shell--crud-wide">
    <AppPageHeader
      title="工作负载"
      description="浏览平台能力、订阅状态与运行态；按分类进入控制台或发起订阅。"
    >
      <template #actions>
        <el-button size="small" :loading="loading || dashLoading" @click="refresh">刷新</el-button>
      </template>
    </AppPageHeader>

    <section v-loading="loading" class="app-stats-row">
      <article class="app-stat-tile">
        <span class="app-stat-label">已开通</span>
        <strong>{{ summary.entitled }}</strong>
      </article>
      <article class="app-stat-tile">
        <span class="app-stat-label">可订阅</span>
        <strong>{{ summary.subscribeable }}</strong>
      </article>
      <article class="app-stat-tile app-stat-tile--ok">
        <span class="app-stat-label">免费可用</span>
        <strong>{{ summary.free }}</strong>
      </article>
      <article class="app-stat-tile">
        <span class="app-stat-label">能力总数</span>
        <strong>{{ summary.total }}</strong>
      </article>
    </section>

    <section v-loading="dashLoading" class="app-stats-row">
      <article class="app-stat-tile app-stat-tile--link app-stat-tile--ok" @click="goExec()">
        <span class="app-stat-label">运行中服务</span>
        <strong>{{ dash?.serviceStatusStats?.running ?? 0 }}</strong>
      </article>
      <article class="app-stat-tile app-stat-tile--link" @click="goExec()">
        <span class="app-stat-label">部署中</span>
        <strong>{{ dash?.serviceStatusStats?.deploying ?? 0 }}</strong>
      </article>
      <article class="app-stat-tile app-stat-tile--link app-stat-tile--warn" @click="goExec()">
        <span class="app-stat-label">异常/停止</span>
        <strong>{{ errorStopped }}</strong>
      </article>
      <article class="app-stat-tile app-stat-tile--link" @click="goExec('k8s')">
        <span class="app-stat-label">近 24h K8s 执行</span>
        <strong>{{ dash?.platformSummary?.executionsBySourceLast24h?.k8s ?? 0 }}</strong>
      </article>
    </section>

    <section v-if="packsWithCapabilities.length" id="packs" class="hub-packs">
      <h3 class="hub-section-title">我的功能包</h3>
      <div class="app-pack-row">
        <PackCard
          v-for="pack in packsWithCapabilities"
          :key="pack.pack_key"
          :pack="pack"
          @manage="scrollToPackSection"
          @subscribe="subscribePack"
        />
      </div>
    </section>

    <div class="hub-layout">
      <nav class="hub-nav">
        <button
          v-for="cat in visibleCategories"
          :key="cat"
          type="button"
          :class="['hub-nav__item', { 'hub-nav__item--active': activeSection === cat }]"
          @click="selectSection(cat)"
        >
          <el-icon v-if="categoryIcon(cat)" class="hub-nav__icon"><component :is="categoryIcon(cat)" /></el-icon>
          <div class="hub-nav__copy">
            <span class="hub-nav__label">{{ categoryLabels[cat] }}</span>
            <span class="hub-nav__desc">{{ categoryDesc[cat] }}</span>
          </div>
          <el-badge :value="categoryCount(cat)" type="info" />
        </button>
      </nav>

      <section class="hub-main">
        <header class="hub-main__head">
          <div>
            <h3 class="hub-section-title">{{ categoryLabels[activeSection] }}</h3>
            <p class="hub-main__desc">{{ categoryDesc[activeSection] }}</p>
          </div>
          <el-button
            v-if="activeSection === 'troubleshoot'"
            size="small"
            type="primary"
            link
            @click="router.push('/app/troubleshooting')"
          >
            打开问题排查页
          </el-button>
        </header>

        <div class="hub-main__toolbar">
          <el-input v-model="searchQ" size="small" clearable placeholder="搜索本分类能力…" style="width: 200px" />
          <el-select v-model="statusFilter" size="small" style="width: 120px">
            <el-option label="全部状态" value="all" />
            <el-option label="已开通" value="entitled" />
            <el-option label="未订阅" value="unsubscribed" />
            <el-option label="免费" value="free" />
          </el-select>
        </div>

        <section v-if="activeSection === 'delivery' && recentDelivery.length" class="app-recent-strip">
          <strong>最近动态</strong>
          <ul>
            <li v-for="row in recentDelivery" :key="row.id" @click="goExecution(row.id)">
              <span>{{ row.name }}</span>
              <el-tag size="small" :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'info'">
                {{ row.status }}
              </el-tag>
            </li>
          </ul>
        </section>

        <div v-loading="loading" class="app-cap-grid">
          <CapabilityHubCard
            v-for="item in sectionItems"
            :key="item.id"
            :item="item"
            :highlighted="highlightCapId === item.id"
            :pack-info="packInfoFor(item.pack_key)"
            @open="openItem"
            @subscribe="subscribeItem"
            @subscribe-pack="subscribePackKey"
            @contact-admin="contactAdmin"
            @executions="goExecForItem"
          />
        </div>
        <el-empty v-if="!sectionItems.length && !loading" description="没有匹配的能力" />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import CapabilityHubCard from '../../components/app/CapabilityHubCard.vue'
import PackCard from '../../components/app/PackCard.vue'
import '../../assets/app-workbench.css'
import {
  CAPABILITY_CATEGORY_DESC,
  CAPABILITY_CATEGORY_ICON,
  HUB_CATEGORY_ORDER,
  type CapabilityCategory
} from '../../config/capabilityCatalog'
import {
  useCapabilityCatalog,
  type PackWithCapabilities,
  type ResolvedCapability,
  type StatusFilter
} from '../../composables/useCapabilityCatalog'
import { useDashboardStore } from '../../stores/dashboard'
import type { DashboardData } from '../../types/dashboard'
import { openCapability } from '../../utils/capabilityNavigation'
import { parseHubCapId, parseHubSection } from '../../utils/hubQuery'
import { resolveCatalogIcon } from '../../utils/catalogIcons'

const route = useRoute()
const router = useRouter()
const dashboardStore = useDashboardStore()
const {
  loading,
  summary,
  packsWithCapabilities,
  categoryLabels,
  load: loadCaps,
  subscribe,
  filterCapabilities,
  byCategory
} = useCapabilityCatalog()

const categoryDesc = CAPABILITY_CATEGORY_DESC
const searchQ = ref('')
const statusFilter = ref<StatusFilter>('all')
const activeSection = ref<CapabilityCategory>('delivery')
const highlightCapId = ref('')

const dashLoading = computed(() => dashboardStore.loading)
const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const errorStopped = computed(() => {
  const s = dash.value?.serviceStatusStats
  return (s?.error ?? 0) + (s?.stopped ?? 0)
})

const packMap = computed(() => {
  const m = new Map<string, PackWithCapabilities>()
  for (const p of packsWithCapabilities.value) m.set(p.pack_key, p)
  return m
})

const visibleCategories = computed(() =>
  HUB_CATEGORY_ORDER.filter((cat) => (byCategory.value.get(cat) || []).length > 0)
)

const categoryCount = (cat: CapabilityCategory) => (byCategory.value.get(cat) || []).length

const categoryIcon = (cat: CapabilityCategory) => resolveCatalogIcon(CAPABILITY_CATEGORY_ICON[cat])

const sectionItems = computed(() =>
  filterCapabilities({ q: searchQ.value, status: statusFilter.value, category: activeSection.value })
)

const DELIVERY_SOURCES = new Set(['k8s', 'cli', 'job'])

const recentDelivery = computed(() => {
  const rows = dash.value?.recentExecutions ?? []
  return rows.filter((r) => (r.source ? DELIVERY_SOURCES.has(r.source) : true)).slice(0, 5)
})

const packInfoFor = (packKey?: string) => {
  if (!packKey) return undefined
  const p = packMap.value.get(packKey)
  if (!p) return undefined
  return { entitled: p.entitled, can_subscribe: p.can_subscribe, display_name: p.display_name }
}

const syncFromRoute = async () => {
  activeSection.value = parseHubSection(route.query.section)
  highlightCapId.value = parseHubCapId(route)
  if (highlightCapId.value) {
    await nextTick()
    document.querySelector('.hub-card--highlight')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
  if (route.hash === '#packs') {
    await nextTick()
    document.getElementById('packs')?.scrollIntoView({ behavior: 'smooth' })
  }
}

const selectSection = (cat: CapabilityCategory) => {
  activeSection.value = cat
  searchQ.value = ''
  statusFilter.value = 'all'
  void router.replace({ path: '/app/workloads', query: { section: cat }, hash: route.hash || undefined })
}

const scrollToPackSection = (pack: PackWithCapabilities) => {
  const first = pack.capabilities[0]
  if (first) {
    activeSection.value = first.category
    void router.replace({ path: '/app/workloads', query: { section: first.category, cap: first.id }, hash: '#packs' })
  }
}

watch(
  () => [route.query.section, route.query.cap, route.query.zone, route.hash],
  () => {
    void syncFromRoute()
  }
)

const refresh = async () => {
  await Promise.all([dashboardStore.fetchDashboardData(), loadCaps(true)])
}

const openItem = (item: ResolvedCapability) => openCapability(router, item)

const subscribeItem = (item: ResolvedCapability) => {
  void subscribe(item)
}

const subscribePack = (pack: PackWithCapabilities) => {
  void subscribe({ pack_key: pack.pack_key, can_subscribe: pack.can_subscribe })
}

const subscribePackKey = (packKey: string) => {
  const p = packMap.value.get(packKey)
  if (p) subscribePack(p)
}

const contactAdmin = () => {
  ElMessage.info('请联系管理员开通此能力')
}

const goExec = (source?: string) => {
  router.push({ path: '/app/execution-records', query: source ? { source } : undefined })
}

const goExecForItem = (item: ResolvedCapability) => {
  goExec(item.execution_source || undefined)
}

const goExecution = (id: string) => {
  router.push(`/app/executions/${id}`)
}

onMounted(async () => {
  await Promise.all([dashboardStore.fetchDashboardData(), loadCaps()])
  await syncFromRoute()
})
</script>

<style scoped>
.hub-packs {
  margin-bottom: 20px;
}
.hub-section-title {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 600;
}
.hub-main__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding-left: 12px;
  border-left: 3px solid var(--el-color-primary);
}
.hub-main__desc {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.hub-main__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 14px;
}
</style>
