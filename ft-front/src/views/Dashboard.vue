<template>
  <div class="dashboard page-shell page-shell--fill dashboard--layout">
    <div class="page-header">
      <div class="page-header__titles">
        <h2>概览</h2>
        <el-popover placement="bottom-start" :width="320" trigger="click">
          <template #reference>
            <el-button text type="primary" class="dashboard-help">说明</el-button>
          </template>
          <ul class="dashboard-help-list page-desc--muted">
            <li>顶部 KPI 与控制台登记的机器、集群、作业、审计数据一致。</li>
            <li>机器 CPU / 内存 / 磁盘为<strong>在线</strong>机器心跳上报均值，离线机器不参与平均。</li>
            <li>
              「业务服务」指应用服务台账；「运行」含<strong> deploying</strong>。</li>
          </ul>
        </el-popover>
      </div>
      <el-button
        type="primary"
        :icon="RefreshRight"
        :loading="dashboardStore.loading"
        @click="handleRefresh"
      >
        刷新
      </el-button>
    </div>

    <!-- KPI -->
    <div class="dash-grid dash-grid--kpi">
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="goInitTools"
      >
        <div class="snapshot-label">托管机器</div>
        <div class="snapshot-value">
          {{ dash?.platformSummary?.machines?.online ?? 0 }}
          <span class="snapshot-muted">/ {{ dash?.platformSummary?.machines?.total ?? 0 }}</span>
        </div>
        <div class="snapshot-foot">离线 {{ dash?.platformSummary?.machines?.offline ?? 0 }}</div>
        <div class="snapshot-foot snapshot-foot--sub">
          主控 {{ dash?.platformSummary?.machines?.masters ?? 0 }} · Worker
          {{ dash?.platformSummary?.machines?.workers ?? 0 }}
        </div>
      </el-card>
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="goK8sClusters"
      >
        <div class="snapshot-label">K8s 集群</div>
        <div class="snapshot-value">
          {{ dash?.platformSummary?.k8sClusters?.running ?? 0 }}
          <span class="snapshot-muted">/ {{ dash?.platformSummary?.k8sClusters?.total ?? 0 }}</span>
        </div>
        <div class="snapshot-foot">
          待定 {{ dash?.platformSummary?.k8sClusters?.pending ?? 0 }} · 失败
          {{ dash?.platformSummary?.k8sClusters?.failed ?? 0 }}
        </div>
      </el-card>
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="goJobCenter"
      >
        <div class="snapshot-label">进行中作业</div>
        <div class="snapshot-value">{{ dash?.platformSummary?.tasksActive ?? 0 }}</div>
        <div class="snapshot-foot">作业中心</div>
      </el-card>
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="goExecRecords"
      >
        <div class="snapshot-label">近 24h 执行</div>
        <div class="snapshot-value">{{ dash?.platformSummary?.executionsLast24h ?? 0 }}</div>
        <div class="snapshot-foot">控制台 / CLI</div>
      </el-card>
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="snapshot-card snapshot-card--static">
        <div class="snapshot-label">用户 / 审计日志</div>
        <div class="snapshot-value">
          {{ dash?.platformSummary?.usersTotal ?? 0 }}
          <span class="snapshot-muted"> / {{ dash?.platformSummary?.operationLogsTotal ?? 0 }}</span>
        </div>
        <div class="snapshot-foot">用户数 · 操作日志条数</div>
      </el-card>
    </div>

    <!-- 资源 + 业务分布：同一网格，少滚动 -->
    <div class="dash-grid dash-grid--main">
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="meter-card meter-card--cpu">
        <div class="meter-head">
          <span>在线机器平均 CPU</span>
          <el-tag :type="getUsageType(dash?.resourceUsage?.cpu ?? 0)" size="small">
            {{ Number(dash?.resourceUsage?.cpu ?? 0).toFixed(1) }}%
          </el-tag>
        </div>
        <el-progress
          :percentage="Math.round(clampPct(dash?.resourceUsage?.cpu ?? 0))"
          :color="getUsageColor(dash?.resourceUsage?.cpu ?? 0)"
          :show-text="false"
          class="meter-progress"
        />
      </el-card>
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="meter-card meter-card--mem">
        <div class="meter-head">
          <span>在线机器平均内存</span>
          <el-tag :type="getUsageType(dash?.resourceUsage?.memory ?? 0)" size="small">
            {{ Number(dash?.resourceUsage?.memory ?? 0).toFixed(1) }}%
          </el-tag>
        </div>
        <el-progress
          :percentage="Math.round(clampPct(dash?.resourceUsage?.memory ?? 0))"
          :color="getUsageColor(dash?.resourceUsage?.memory ?? 0)"
          :show-text="false"
          class="meter-progress"
        />
      </el-card>
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="meter-card meter-card--disk">
        <div class="meter-head">
          <span>在线机器平均磁盘</span>
          <el-tag :type="getUsageType(dash?.resourceUsage?.disk ?? 0)" size="small">
            {{ Number(dash?.resourceUsage?.disk ?? 0).toFixed(1) }}%
          </el-tag>
        </div>
        <el-progress
          :percentage="Math.round(clampPct(dash?.resourceUsage?.disk ?? 0))"
          :color="getUsageColor(dash?.resourceUsage?.disk ?? 0)"
          :show-text="false"
          class="meter-progress"
        />
      </el-card>

      <el-card v-loading="dashboardStore.loading" shadow="hover" class="svc-card">
        <div class="svc-card-head">
          <span class="svc-card-title">业务服务状态</span>
          <span class="svc-card-meta">台账合计 {{ svcTotal }}</span>
        </div>
        <div class="svc-stack-wrap" aria-label="状态占比">
          <div v-if="svcTotal > 0" class="svc-stack-bar">
            <div
              v-if="svcRunWidth > 0"
              class="svc-stack-seg svc-stack-seg--run"
              :style="{ width: svcRunWidth + '%' }"
              title="运行中（含 deploying）"
            />
            <div
              v-if="svcStopWidth > 0"
              class="svc-stack-seg svc-stack-seg--stopped"
              :style="{ width: svcStopWidth + '%' }"
              title="已停止"
            />
            <div
              v-if="svcErrWidth > 0"
              class="svc-stack-seg svc-stack-seg--error"
              :style="{ width: svcErrWidth + '%' }"
              title="异常"
            />
          </div>
          <div v-else class="svc-stack-empty page-desc--muted">暂无业务服务台账</div>
        </div>
        <div class="svc-legends">
          <div class="svc-legend">
            <span class="svc-dot svc-dot--run" />
            <span>运行 {{ dash?.serviceStatusStats?.running ?? 0 }}</span>
          </div>
          <div class="svc-legend">
            <span class="svc-dot svc-dot--stopped" />
            <span>停止 {{ dash?.serviceStatusStats?.stopped ?? 0 }}</span>
          </div>
          <div class="svc-legend">
            <span class="svc-dot svc-dot--error" />
            <span>异常 {{ dash?.serviceStatusStats?.error ?? 0 }}</span>
          </div>
        </div>
      </el-card>
    </div>

    <el-card v-loading="dashboardStore.loading" shadow="never" class="dash-tabs-card">
      <el-tabs v-model="activeDetailTab" class="dash-tabs">
        <el-tab-pane v-if="isAdminUser" label="最近集群" name="k8s">
          <template #label>
            <span class="dash-tab-label">最近集群</span>
          </template>
          <el-table :data="dash?.recentK8sClusters ?? []" stripe border size="small" :max-height="tableMaxPx" empty-text="暂无记录">
            <el-table-column prop="clusterName" label="名称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="version" label="版本" width="90" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="96" align="center">
              <template #default="scope">
                <el-tag :type="clusterStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="masterNode" label="主节点" min-width="120" show-overflow-tooltip />
            <el-table-column prop="updatedAt" label="更新" min-width="146">
              <template #default="scope">
                {{ formatTs(scope.row.updatedAt) }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane v-if="isAdminUser" label="最近安装" name="install">
          <el-table :data="dash?.recentServiceInstalls ?? []" stripe border size="small" :max-height="tableMaxPx" empty-text="暂无记录">
            <el-table-column prop="service" label="组件" width="110" show-overflow-tooltip />
            <el-table-column prop="profile" label="配置" width="100" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="92" align="center">
              <template #default="scope">
                <el-tag :type="genericStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="currentStep" label="步骤" min-width="136" show-overflow-tooltip />
            <el-table-column prop="updatedAt" label="更新" min-width="146">
              <template #default="scope">
                {{ formatTs(scope.row.updatedAt) }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="业务服务" name="services">
          <el-table :data="dash?.recentDeployments ?? []" stripe border size="small" :max-height="tableMaxPx" empty-text="暂无服务">
            <el-table-column prop="name" label="服务名称" min-width="140" show-overflow-tooltip />
            <el-table-column prop="image" label="镜像" min-width="160" show-overflow-tooltip />
            <el-table-column prop="replicas" label="副本" width="72" align="center" />
            <el-table-column prop="status" label="状态" width="92" align="center">
              <template #default="scope">
                <el-tag :type="getStatusType(scope.row.status)" size="small">
                  {{ getStatusText(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="updateTime" label="更新时间" min-width="146">
              <template #default="scope">
                {{ formatTs(scope.row.updateTime) }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="执行记录" name="exec">
          <el-table :data="dash?.recentExecutions ?? []" stripe border size="small" :max-height="tableMaxPx" empty-text="暂无执行记录">
            <el-table-column prop="name" label="名称" min-width="168" show-overflow-tooltip />
            <el-table-column prop="source" label="来源" width="80" show-overflow-tooltip />
            <el-table-column prop="category" label="类别" width="100" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="92" align="center">
              <template #default="scope">
                <el-tag :type="genericStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="targetHost" label="目标" min-width="112" show-overflow-tooltip />
            <el-table-column prop="finishedAt" label="结束时间" min-width="146">
              <template #default="scope">
                {{ formatTs(scope.row.finishedAt) }}
              </template>
            </el-table-column>
            <el-table-column prop="durationMs" label="耗时" width="84" align="right">
              <template #default="scope">
                {{ formatDuration(scope.row.durationMs) }}
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
      <div class="dash-tabs-actions">
        <el-link v-if="isAdminUser && activeDetailTab === 'k8s'" type="primary" :underline="false" @click="goK8sClusters">
          集群列表
        </el-link>
        <el-link v-if="isAdminUser && activeDetailTab === 'install'" type="primary" :underline="false" @click="goServiceDeploy">
          服务部署
        </el-link>
        <el-link v-if="activeDetailTab === 'services'" type="primary" :underline="false" @click="navigateToServiceList">
          台账列表
        </el-link>
        <el-link
          v-if="activeDetailTab === 'exec'"
          type="primary"
          :underline="false"
          @click="isAdminUser ? goExecRecords() : goJobCenter()"
        >
          {{ isAdminUser ? '全部记录' : '作业中心' }}
        </el-link>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshRight } from '@element-plus/icons-vue'
import { useDashboardStore } from '../stores/dashboard'
import type { DashboardData } from '../types/dashboard'

const dashboardStore = useDashboardStore()
const router = useRouter()
const route = useRoute()

const shellPrefix = computed(() => (route.path.startsWith('/admin') ? '/admin' : '/app'))
const isAdminUser = computed(() => {
  try {
    const u = JSON.parse(localStorage.getItem('userInfo') || '{}') as { role?: string }
    return u?.role === 'admin' || u?.role === 'super_admin'
  } catch {
    return false
  }
})

const activeDetailTab = ref('services')

watch(
  isAdminUser,
  (admin) => {
    if (!admin && (activeDetailTab.value === 'k8s' || activeDetailTab.value === 'install')) {
      activeDetailTab.value = 'services'
    }
  },
  { immediate: true }
)

const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const svcTotal = computed(() => dash.value?.serviceStatusStats?.total ?? 0)

const svcRunWidth = computed(() => {
  const t = svcTotal.value
  if (!t) return 0
  const n = dash.value?.serviceStatusStats?.running ?? 0
  return Math.max(0, Math.round((n / t) * 1000) / 10)
})
const svcStopWidth = computed(() => {
  const t = svcTotal.value
  if (!t) return 0
  const n = dash.value?.serviceStatusStats?.stopped ?? 0
  return Math.max(0, Math.round((n / t) * 1000) / 10)
})
const svcErrWidth = computed(() => {
  const t = svcTotal.value
  if (!t) return 0
  const n = dash.value?.serviceStatusStats?.error ?? 0
  return Math.max(0, Math.round((n / t) * 1000) / 10)
})

const tableMaxPx = ref(280)
const recalcTableMax = () => {
  const h = typeof window !== 'undefined' ? window.innerHeight : 900
  tableMaxPx.value = Math.min(420, Math.max(200, Math.floor(h * 0.34)))
}

const goInitTools = () => {
  router.push(`${shellPrefix.value}/init-tools`)
}
const goJobCenter = () => {
  router.push(`${shellPrefix.value}/job/center`)
}
const goK8sClusters = () => {
  if (!isAdminUser.value) {
    ElMessage.info('Kubernetes 集群列表请使用管理端入口（管理员）')
    return
  }
  router.push('/admin/service/k8s/clusters')
}
const goExecRecords = () => {
  if (!isAdminUser.value) {
    ElMessage.info('执行记录请使用管理端入口（管理员）')
    return
  }
  router.push('/admin/execution-records')
}
const goServiceDeploy = () => {
  if (!isAdminUser.value) {
    ElMessage.info('服务部署请使用管理端入口（管理员）')
    return
  }
  router.push('/admin/service/deploy')
}

const fetchDashboardData = async () => {
  const data = await dashboardStore.fetchDashboardData()
  if (!data) {
    ElMessage.error('获取仪表盘数据失败')
  }
}

const handleRefresh = () => {
  void fetchDashboardData()
}

const clampPct = (n: number) => {
  if (Number.isNaN(n) || n < 0) return 0
  if (n > 100) return 100
  return n
}

const getUsageType = (percentage: number) => {
  if (percentage >= 80) return 'danger'
  if (percentage >= 60) return 'warning'
  return 'success'
}

const getUsageColor = (percentage: number) => {
  if (percentage >= 80) return '#f56c6c'
  if (percentage >= 60) return '#e6a23c'
  return '#67c23a'
}

const formatTs = (iso?: string) => {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

const formatDuration = (ms?: number) => {
  if (ms == null || ms <= 0) return '—'
  if (ms < 1000) return `${ms} ms`
  const s = ms / 1000
  if (s < 60) return `${s.toFixed(1)} s`
  const m = Math.floor(s / 60)
  const rs = s - m * 60
  return `${m}m ${rs.toFixed(0)}s`
}

const getStatusType = (status: string) => {
  switch (status) {
    case 'running':
      return 'success'
    case 'deploying':
      return 'warning'
    case 'stopped':
      return 'info'
    case 'error':
    case 'failed':
      return 'danger'
    default:
      return 'warning'
  }
}

const getStatusText = (status: string) => {
  switch (status) {
    case 'running':
      return '运行中'
    case 'deploying':
      return '部署中'
    case 'stopped':
      return '已停止'
    case 'error':
    case 'failed':
      return '异常'
    default:
      return status || '—'
  }
}

const clusterStatusType = (s: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'running' || v === 'success') return 'success'
  if (v === 'failed' || v === 'error') return 'danger'
  if (v === 'pending') return 'info'
  return 'warning'
}

const genericStatusType = (s: string) => {
  const v = (s || '').toLowerCase()
  if (v === 'success' || v === 'running' || v === 'completed') return 'success'
  if (v === 'failed' || v === 'error') return 'danger'
  if (v === 'pending' || v === 'cancelled' || v === 'canceled') return 'info'
  return 'warning'
}

const navigateToServiceList = () => {
  goServiceDeploy()
}

onMounted(() => {
  const b = route.query.billing
  if (b === 'success') ElMessage.success('支付已完成，订阅状态将在 Stripe Webhook 同步后更新')
  if (b === 'cancel') ElMessage.info('已取消支付')
  recalcTableMax()
  window.addEventListener('resize', recalcTableMax)
  void fetchDashboardData()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', recalcTableMax)
})
</script>

<style scoped>
.dashboard--layout {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: 10px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-shrink: 0;
}

.page-header__titles {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
  min-width: 0;
}

.page-header h2 {
  margin: 0;
  font-size: var(--page-header-title-max);
  color: var(--apple-ink);
}

.dashboard-help {
  font-size: 13px !important;
  padding: 0 4px !important;
  min-height: auto !important;
}

.dashboard-help-list {
  margin: 0;
  padding-left: 1.1em;
}

.dash-grid {
  display: grid;
  gap: 10px;
  flex-shrink: 0;
}

.dash-grid--kpi {
  grid-template-columns: repeat(5, minmax(0, 1fr));
}

.dash-grid--main {
  grid-template-columns: repeat(4, minmax(0, 1fr));
  align-items: stretch;
}

.snapshot-card {
  cursor: pointer;
  transition:
    transform 0.12s ease,
    box-shadow 0.12s ease;
}

.snapshot-card:not(.snapshot-card--static):hover {
  transform: translateY(-2px);
}

.snapshot-card--static {
  cursor: default;
}

.snapshot-card :deep(.el-card__body) {
  padding: 12px 14px;
}

.snapshot-label {
  font-size: 12px;
  color: var(--apple-muted, #909399);
}

.snapshot-value {
  margin-top: 4px;
  font-size: 20px;
  font-weight: 700;
  color: #303133;
  line-height: 1.2;
}

.snapshot-muted {
  font-size: 14px;
  font-weight: 500;
  color: #909399;
}

.snapshot-foot {
  margin-top: 6px;
  font-size: 11px;
  color: #a8abb2;
  line-height: 1.35;
}

.snapshot-foot--sub {
  margin-top: 2px;
  font-size: 11px;
}

.meter-card :deep(.el-card__body) {
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 8px;
  min-height: 72px;
}

.meter-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.meter-progress {
  margin: 0;
}

.meter-progress :deep(.el-progress-bar__outer) {
  height: 8px !important;
  border-radius: 999px;
}

.svc-card {
  grid-column: span 1;
}

.svc-card :deep(.el-card__body) {
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  justify-content: center;
  min-height: 92px;
}

.svc-card-head {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}

.svc-card-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}

.svc-card-meta {
  font-size: 11px;
  color: #a8abb2;
}

.svc-stack-wrap {
  min-height: 14px;
}

.svc-stack-bar {
  display: flex;
  width: 100%;
  height: 12px;
  border-radius: 6px;
  overflow: hidden;
  background: var(--apple-hairline, #ebeef5);
}

.svc-stack-seg {
  height: 100%;
  min-width: 0;
}

.svc-stack-seg--run {
  background: #67c23a;
}

.svc-stack-seg--stopped {
  background: #c8c9cc;
}

.svc-stack-seg--error {
  background: #f56c6c;
}

.svc-stack-empty {
  font-size: 12px;
}

.svc-legends {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 16px;
}

.svc-legend {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #606266;
}

.svc-dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  flex-shrink: 0;
}

.svc-dot--run {
  background: #67c23a;
}
.svc-dot--stopped {
  background: #c8c9cc;
}
.svc-dot--error {
  background: #f56c6c;
}

.dash-tabs-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-lighter);
}

.dash-tabs-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 10px 12px 10px;
  position: relative;
}

.dash-tabs {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.dash-tabs :deep(.el-tabs__header) {
  margin: 0 0 8px;
  flex-shrink: 0;
}

.dash-tabs :deep(.el-tabs__nav-wrap)::after {
  height: 1px;
}

.dash-tabs :deep(.el-tabs__content),
.dash-tabs :deep(.el-tab-pane) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.dash-tab-label {
  letter-spacing: 0.02em;
}

.dash-tabs-actions {
  position: absolute;
  top: 10px;
  right: 12px;
  z-index: 2;
}

@media screen and (max-width: 1280px) {
  .dash-grid--kpi {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .dash-grid--main {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .svc-card {
    grid-column: span 2;
  }
}

@media screen and (max-width: 720px) {
  .dash-grid--kpi {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dash-grid--main {
    grid-template-columns: 1fr;
  }

  .svc-card {
    grid-column: span 1;
  }

  .dash-tabs-actions {
    position: static;
    margin: 0 0 6px;
    text-align: right;
  }
}
</style>
