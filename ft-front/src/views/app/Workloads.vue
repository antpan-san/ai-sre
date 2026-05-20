<template>
  <div class="workloads page-shell page-shell--crud-wide">
    <AppPageHeader
      title="工作负载"
      description="Kubernetes、应用服务、Linux 主机与初始化工具；执行受订阅状态控制。"
    >
      <template #actions>
        <el-button size="small" link type="primary" @click="router.push('/app/capabilities')">能力中心</el-button>
        <el-button size="small" :loading="dashLoading" @click="refresh">刷新</el-button>
      </template>
    </AppPageHeader>

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

    <section v-if="recentDelivery.length" class="app-recent-strip">
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

    <div class="app-workload-grid">
      <WorkloadZoneCard
        v-for="item in primaryZones"
        :key="item.id"
        :item="item"
        :highlighted="highlightId === item.id"
        @open="openItem"
        @subscribe="subscribeItem"
        @executions="goExecForItem"
      />
    </div>
    <div class="app-workload-grid app-workload-grid--secondary">
      <WorkloadZoneCard
        v-for="item in secondaryZones"
        :key="item.id"
        :item="item"
        :highlighted="highlightId === item.id"
        @open="openItem"
        @subscribe="subscribeItem"
        @executions="goExecForItem"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import WorkloadZoneCard from '../../components/app/WorkloadZoneCard.vue'
import '../../assets/app-workbench.css'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { useDashboardStore } from '../../stores/dashboard'
import type { DashboardData } from '../../types/dashboard'
import { openCapability } from '../../utils/capabilityNavigation'

const ZONE_TO_ID: Record<string, string> = {
  k8s: 'k8s_delivery',
  services: 'service_deploy',
  linux: 'linux_hosts',
  init: 'init_tools',
  proxy: 'proxy',
  mirror: 'k8s_mirror'
}

const route = useRoute()
const router = useRouter()
const dashboardStore = useDashboardStore()
const { deliveryCapabilities, shellPrefix, load: loadCaps, subscribe } = useCapabilityCatalog()

const highlightId = ref('')

const dashLoading = computed(() => dashboardStore.loading)
const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const errorStopped = computed(() => {
  const s = dash.value?.serviceStatusStats
  return (s?.error ?? 0) + (s?.stopped ?? 0)
})

const primaryZones = computed(() => deliveryCapabilities.value.filter((c) => c.workload_tier !== 'secondary'))
const secondaryZones = computed(() => deliveryCapabilities.value.filter((c) => c.workload_tier === 'secondary'))

const DELIVERY_SOURCES = new Set(['k8s', 'cli', 'job'])

const recentDelivery = computed(() => {
  const rows = dash.value?.recentExecutions ?? []
  return rows
    .filter((r) => (r.source ? DELIVERY_SOURCES.has(r.source) : true))
    .slice(0, 5)
})

const applyZoneQuery = async () => {
  const zone = String(route.query.zone || '')
  const id = ZONE_TO_ID[zone] || (zone.includes('_') ? zone : '')
  highlightId.value = id
  if (id) {
    await nextTick()
    document.querySelector('.zone-card--highlight')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

watch(
  () => route.query.zone,
  () => {
    void applyZoneQuery()
  }
)

const refresh = async () => {
  await Promise.all([dashboardStore.fetchDashboardData(), loadCaps(true)])
}

const openItem = (item: ResolvedCapability) => openCapability(router, item)

const subscribeItem = (item: ResolvedCapability) => {
  void subscribe(item)
}

const goExec = (source?: string) => {
  const query = source ? { source } : undefined
  router.push({ path: `${shellPrefix.value}/execution-records`, query })
}

const goExecForItem = (item: ResolvedCapability) => {
  const source = item.execution_source
  goExec(source || undefined)
}

const goExecution = (id: string) => {
  router.push(`${shellPrefix.value}/executions/${id}`)
}

onMounted(async () => {
  await Promise.all([dashboardStore.fetchDashboardData(), loadCaps()])
  await applyZoneQuery()
})
</script>
