<template>
  <div class="deploy-center deploy-config-page page-shell page-shell--crud-wide">
    <AppPageHeader
      title="工作负载"
      description="Kubernetes、应用服务、Linux 主机与初始化工具统一入口；未订阅能力可先查看说明再开通。"
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

    <section class="workload-capability-grid" aria-label="工作负载能力状态">
      <article v-for="item in workloadCaps" :key="item.id" class="workload-capability-card">
        <div>
          <h3>{{ item.name }}</h3>
          <p>{{ item.description }}</p>
          <span>{{ item.pack_display_name || item.pack_key || '免费能力' }}</span>
        </div>
        <div class="workload-capability-card__actions">
          <el-tag size="small" :type="statusType(item.status)">{{ item.status }}</el-tag>
          <el-button size="small" :type="actionType(item)" @click="handleWorkloadAction(item)">
            {{ actionLabel(item) }}
          </el-button>
        </div>
      </article>
    </section>

    <section id="cluster" class="deploy-config-category">
      <h3 class="deploy-config-category__title">集群与主机</h3>
      <DeployClusterSection
        :k8s-entitled="k8sEntitled"
        :k8s-cap="k8sCap"
        @subscribe-k8s="subscribeK8s"
        @contact-admin="contactAdmin"
      />
    </section>

    <section v-if="nodeOpsVisible" id="services" class="deploy-config-category">
      <h3 class="deploy-config-category__title">基础服务</h3>
      <p class="deploy-config-category__desc">中间件与应用服务：每类服务独立卡片，展开后配置参数并生成部署脚本。</p>
      <ServiceDeployGrid />
    </section>

    <section v-if="nodeOpsVisible" id="init-tools" class="deploy-config-category">
      <h3 class="deploy-config-category__title">节点初始化</h3>
      <p class="deploy-config-category__desc">部署前环境准备：填写节点与参数，生成 Ansible 脚本在控制机执行。</p>
      <InitToolsSection />
    </section>

    <el-collapse v-if="subscribeRows.length" v-model="subscribeCollapse" class="deploy-subscribe-collapse">
      <el-collapse-item name="subscribe">
        <template #title>
          <span class="deploy-section__title deploy-section__title--collapse">
            可订阅
            <span class="deploy-section__count">（{{ subscribeRows.length }}）</span>
          </span>
        </template>
        <div class="deploy-list">
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
import DeployClusterSection from '../../components/deploy/DeployClusterSection.vue'
import ServiceDeployGrid from '../../components/deploy/ServiceDeployGrid.vue'
import InitToolsSection from '../../components/deploy/InitToolsSection.vue'
import '../../assets/app-workbench.css'
import '../../assets/deploy-config.css'
import { useCapabilityCatalog, type ResolvedCapability } from '../../composables/useCapabilityCatalog'
import { useDashboardStore } from '../../stores/dashboard'
import type { DashboardData } from '../../types/dashboard'
import { INLINE_ON_DEPLOY_CAP_IDS } from '../../config/capabilityCatalog'
import { parseHubCapId, shouldExpandSubscribe } from '../../utils/hubQuery'

const route = useRoute()
const router = useRouter()
const dashboardStore = useDashboardStore()
const { loading, load: loadCaps, subscribe, filterCapabilities, isEntitledStatus } = useCapabilityCatalog()

const highlightCapId = ref('')
const subscribeCollapse = ref<string[]>([])

const dashLoading = computed(() => dashboardStore.loading)
const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const workloadCaps = computed(() =>
  filterCapabilities({ category: 'delivery', status: 'all' }).filter((c) => INLINE_ON_DEPLOY_CAP_IDS.has(c.id))
)

const deliveryItems = computed(() =>
  filterCapabilities({ category: 'delivery', status: 'all' }).filter((c) => !INLINE_ON_DEPLOY_CAP_IDS.has(c.id))
)

const subscribeRows = computed(() => deliveryItems.value.filter((c) => !isEntitledStatus(c.status)))

const k8sCap = computed(() =>
  filterCapabilities({ category: 'delivery', status: 'all' }).find((c) => c.id === 'k8s_delivery') || null
)
const k8sEntitled = computed(() => (k8sCap.value ? isEntitledStatus(k8sCap.value.status) : false))

const nodeOpsCap = computed(() =>
  filterCapabilities({ category: 'delivery', status: 'all' }).find((c) => c.id === 'service_deploy') || null
)
const nodeOpsVisible = computed(() => {
  const cap = nodeOpsCap.value
  if (!cap) return false
  return isEntitledStatus(cap.status)
})

const DELIVERY_SOURCES = new Set(['k8s', 'cli', 'job'])

const recentDelivery = computed(() => {
  const rows = dash.value?.recentExecutions ?? []
  return rows.filter((r) => (r.source ? DELIVERY_SOURCES.has(r.source) : true)).slice(0, 3)
})

const scrollToHash = async () => {
  const hash = route.hash?.replace('#', '')
  if (!hash) return
  await nextTick()
  document.getElementById(hash)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

const syncFromRoute = async () => {
  highlightCapId.value = parseHubCapId(route)
  if (shouldExpandSubscribe(route)) {
    subscribeCollapse.value = ['subscribe']
  }
  await scrollToHash()
}

watch(
  () => [route.query.cap, route.query.zone, route.query.expand, route.hash],
  () => {
    void syncFromRoute()
  }
)

const refresh = async () => {
  await Promise.all([dashboardStore.fetchDashboardData(), loadCaps(true)])
}

const statusType = (status: string) => {
  if (status === '已订阅' || status === '免费可用' || status === '管理员已开通') return 'success'
  if (status === '未订阅' || status === '联系管理员开通') return 'warning'
  return 'info'
}

const actionLabel = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) return item.open_path ? '打开' : '已可用'
  if (item.can_subscribe) return '订阅'
  if (item.status === '联系管理员开通') return '联系管理员'
  return '查看说明'
}

const actionType = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) return 'primary'
  if (item.can_subscribe) return 'warning'
  return 'info'
}

const handleWorkloadAction = (item: ResolvedCapability) => {
  if (isEntitledStatus(item.status)) {
    if (item.open_path) void router.push(item.open_path)
    return
  }
  if (item.can_subscribe) {
    void subscribe(item)
    return
  }
  contactAdmin()
}

const subscribeItem = (item: ResolvedCapability) => {
  void subscribe(item)
}

const subscribeK8s = () => {
  if (k8sCap.value) void subscribe(k8sCap.value)
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
.workload-capability-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 12px;
}
.workload-capability-card {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  min-height: 118px;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 14px;
  background: var(--el-bg-color);
}
.workload-capability-card h3 {
  margin: 0 0 6px;
  font-size: 15px;
}
.workload-capability-card p {
  margin: 0 0 8px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.workload-capability-card span {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.workload-capability-card__actions {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: space-between;
  flex-shrink: 0;
}
@media (max-width: 640px) {
  .workload-capability-card {
    flex-direction: column;
  }
  .workload-capability-card__actions {
    flex-direction: row;
    align-items: center;
  }
}
.deploy-recent {
  list-style: none;
  margin: 0 0 8px;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.deploy-recent li {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--el-fill-color-light);
  cursor: pointer;
  font-size: 13px;
}
.deploy-recent li:hover {
  background: var(--el-fill-color);
}
.deploy-recent__status {
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
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
.deploy-list {
  padding-top: 4px;
}
</style>
