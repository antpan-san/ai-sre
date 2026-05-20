<template>
  <div class="workloads-hub page-shell page-shell--crud-wide">
    <AppPageHeader title="工作负载" description="概览运行态、管理功能包订阅，按分类进入控制台。">
      <template #actions>
        <el-button size="small" :loading="loading || dashLoading" @click="refresh">刷新</el-button>
      </template>
    </AppPageHeader>

    <el-tabs v-model="activeTab" class="hub-tabs" @tab-change="onTabChange">
      <el-tab-pane label="概览" name="overview">
        <section v-loading="loading || dashLoading" class="hub-pane">
          <h3 class="hub-pane__subtitle">订阅状态</h3>
          <div class="app-stats-row">
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
          </div>

          <h3 class="hub-pane__subtitle">运行态</h3>
          <div class="app-stats-row">
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
          </div>

          <section v-if="recentDelivery.length" class="app-recent-strip">
            <div class="hub-pane__strip-head">
              <strong>最近动态</strong>
              <el-button size="small" link type="primary" @click="switchTab('delivery')">查看交付部署</el-button>
            </div>
            <ul>
              <li v-for="row in recentDelivery" :key="row.id" @click="goExecution(row.id)">
                <span>{{ row.name }}</span>
                <el-tag size="small" :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'info'">
                  {{ row.status }}
                </el-tag>
              </li>
            </ul>
          </section>
        </section>
      </el-tab-pane>

      <el-tab-pane label="我的功能包" name="packs">
        <section v-loading="loading" class="hub-pane">
          <p class="hub-pane__desc">按功能包订阅或查看包内能力；订阅后可从各分类页打开控制台。</p>
          <div v-if="packsWithCapabilities.length" class="app-pack-row">
            <PackCard
              v-for="pack in packsWithCapabilities"
              :key="pack.pack_key"
              :pack="pack"
              @manage="goPackCategory"
              @subscribe="subscribePack"
            />
          </div>
          <el-empty v-else description="暂无功能包" />
        </section>
      </el-tab-pane>

      <el-tab-pane
        v-for="cat in visibleCategories"
        :key="cat"
        :label="categoryLabels[cat]"
        :name="cat"
      >
        <section v-loading="loading" class="hub-pane">
          <p class="hub-pane__desc">{{ categoryDesc[cat] }}</p>

          <div class="hub-pane__toolbar">
            <el-input v-model="searchQ" size="small" clearable placeholder="搜索…" style="width: 200px" />
            <el-select v-model="statusFilter" size="small" style="width: 120px">
              <el-option label="全部状态" value="all" />
              <el-option label="已开通" value="entitled" />
              <el-option label="未订阅" value="unsubscribed" />
              <el-option label="免费" value="free" />
            </el-select>
            <el-button
              v-if="cat === 'troubleshoot'"
              size="small"
              type="primary"
              link
              @click="router.push('/app/troubleshooting')"
            >
              打开问题排查页
            </el-button>
          </div>

          <div class="app-cap-grid">
            <CapabilityHubCard
              v-for="item in itemsForTab(cat)"
              :key="item.id"
              :item="item"
              :highlighted="highlightCapId === item.id && activeTab === cat"
              :pack-info="packInfoFor(item.pack_key)"
              @open="openItem"
              @subscribe="subscribeItem"
              @subscribe-pack="subscribePackKey"
              @contact-admin="contactAdmin"
              @executions="goExecForItem"
            />
          </div>
          <el-empty v-if="!itemsForTab(cat).length && !loading" description="没有匹配的能力" />
        </section>
      </el-tab-pane>
    </el-tabs>
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
import {
  isCapabilityTab,
  parseHubCapId,
  parseHubTab,
  type HubTab
} from '../../utils/hubQuery'

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
const activeTab = ref<HubTab>('overview')
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

const itemsForTab = (cat: CapabilityCategory) =>
  filterCapabilities({ q: searchQ.value, status: statusFilter.value, category: cat })

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

const buildQuery = (tab: HubTab) => {
  const query: Record<string, string> = { tab }
  if (isCapabilityTab(tab) && highlightCapId.value) {
    query.cap = highlightCapId.value
  }
  return query
}

const syncFromRoute = async () => {
  activeTab.value = parseHubTab(route)
  highlightCapId.value = parseHubCapId(route)
  if (highlightCapId.value && activeTab.value !== 'overview' && activeTab.value !== 'packs') {
    await nextTick()
    document.querySelector('.hub-card--highlight')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

const switchTab = (tab: HubTab) => {
  activeTab.value = tab
  searchQ.value = ''
  statusFilter.value = 'all'
  void router.replace({ path: '/app/workloads', query: buildQuery(tab) })
}

const onTabChange = (name: string | number) => {
  switchTab(String(name) as HubTab)
}

const goPackCategory = (pack: PackWithCapabilities) => {
  const first = pack.capabilities[0]
  if (!first) return
  highlightCapId.value = first.id
  switchTab(first.category)
}

watch(
  () => [route.query.tab, route.query.section, route.query.cap, route.query.zone, route.hash],
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
.hub-tabs {
  margin-top: 4px;
}
.hub-pane {
  padding-top: 12px;
}
.hub-pane__subtitle {
  margin: 0 0 10px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.hub-pane__subtitle:not(:first-child) {
  margin-top: 8px;
}
.hub-pane__desc {
  margin: 0 0 14px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.hub-pane__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-bottom: 14px;
}
.hub-pane__strip-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
</style>
