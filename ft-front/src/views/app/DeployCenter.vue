<template>
  <div class="deploy-center page-shell page-shell--crud-wide">
    <AppPageHeader
      title="部署中心"
      description="安装与管理 Kubernetes、应用服务、Linux 主机及初始化工具。"
    >
      <template #actions>
        <el-button size="small" :loading="loading || dashLoading" @click="refresh">刷新</el-button>
      </template>
    </AppPageHeader>

    <ul v-if="recentDelivery.length" class="deploy-recent">
      <li v-for="row in recentDelivery" :key="row.id" @click="goExecution(row.id)">
        <span>{{ row.name }}</span>
        <span class="deploy-recent__status">{{ row.status }}</span>
      </li>
    </ul>

    <section v-loading="loading" class="deploy-section">
      <h3 class="deploy-section__title">已开通</h3>
      <div v-if="entitledRows.length" class="deploy-list">
        <DeployEntryRow
          v-for="item in entitledRows"
          :key="item.id"
          :item="item"
          mode="entitled"
          :highlighted="highlightCapId === item.id"
          @open="openItem"
        />
      </div>
      <p v-else class="deploy-section__empty">暂无已开通的部署能力</p>
    </section>

    <el-collapse v-model="subscribeCollapse" class="deploy-subscribe-collapse">
      <el-collapse-item name="subscribe">
        <template #title>
          <span class="deploy-section__title deploy-section__title--collapse">
            可订阅
            <span v-if="subscribeRows.length" class="deploy-section__count">（{{ subscribeRows.length }}）</span>
          </span>
        </template>
        <div v-if="subscribeRows.length" class="deploy-list">
          <DeployEntryRow
            v-for="item in subscribeRows"
            :key="item.id"
            :item="item"
            mode="subscribe"
            :highlighted="highlightCapId === item.id"
            @subscribe="subscribeItem"
            @contact-admin="contactAdmin"
          />
        </div>
        <p v-else class="deploy-section__empty">全部部署能力已开通</p>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import AppPageHeader from '../../components/app/AppPageHeader.vue'
import DeployEntryRow from '../../components/app/DeployEntryRow.vue'
import '../../assets/app-workbench.css'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { useDashboardStore } from '../../stores/dashboard'
import type { DashboardData } from '../../types/dashboard'
import { openCapability } from '../../utils/capabilityNavigation'
import { parseHubCapId, shouldExpandSubscribe } from '../../utils/hubQuery'

const route = useRoute()
const router = useRouter()
const dashboardStore = useDashboardStore()
const { loading, load: loadCaps, subscribe, filterCapabilities, isEntitledStatus } = useCapabilityCatalog()

const highlightCapId = ref('')
const subscribeCollapse = ref<string[]>([])

const dashLoading = computed(() => dashboardStore.loading)
const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const sortDelivery = (items: ResolvedCapability[]) =>
  [...items].sort((a, b) => {
    const tier = (t?: string) => (t === 'primary' ? 0 : 1)
    const d = tier(a.workload_tier) - tier(b.workload_tier)
    return d !== 0 ? d : a.name.localeCompare(b.name, 'zh-CN')
  })

const deliveryItems = computed(() =>
  sortDelivery(filterCapabilities({ category: 'delivery', status: 'all' }))
)

const entitledRows = computed(() => deliveryItems.value.filter((c) => isEntitledStatus(c.status)))

const subscribeRows = computed(() => deliveryItems.value.filter((c) => !isEntitledStatus(c.status)))

const DELIVERY_SOURCES = new Set(['k8s', 'cli', 'job'])

const recentDelivery = computed(() => {
  const rows = dash.value?.recentExecutions ?? []
  return rows.filter((r) => (r.source ? DELIVERY_SOURCES.has(r.source) : true)).slice(0, 3)
})

const syncFromRoute = async () => {
  highlightCapId.value = parseHubCapId(route)
  if (shouldExpandSubscribe(route)) {
    subscribeCollapse.value = ['subscribe']
  }
  if (highlightCapId.value) {
    await nextTick()
    document.querySelector('.deploy-row--highlight')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

watch(
  () => [route.query.cap, route.query.zone, route.query.expand],
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

const contactAdmin = () => {
  ElMessage.info('请联系管理员开通此能力')
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
.deploy-section {
  margin-bottom: 8px;
}
.deploy-section__title {
  margin: 0 0 4px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.deploy-section__title--collapse {
  margin: 0;
}
.deploy-section__count {
  font-weight: 400;
  color: var(--el-text-color-secondary);
}
.deploy-section__empty {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.deploy-subscribe-collapse {
  border: none;
}
.deploy-subscribe-collapse :deep(.el-collapse-item__header) {
  border-bottom: none;
  height: auto;
  line-height: 1.4;
  padding: 8px 0;
}
.deploy-subscribe-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}
.deploy-subscribe-collapse :deep(.el-collapse-item__content) {
  padding-bottom: 0;
}
</style>
