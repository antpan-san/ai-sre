<template>
  <div class="dashboard page-shell page-shell--fill dashboard--layout">
    <div class="page-header">
      <div class="page-header__titles">
        <h2>概览</h2>
      </div>
      <el-tooltip content="刷新数据" placement="bottom-end">
        <button
          type="button"
          class="dash-refresh"
          :disabled="dashboardStore.loading"
          :aria-busy="dashboardStore.loading"
          aria-label="刷新概览"
          @click="handleRefresh"
        >
          <el-icon class="dash-refresh__icon" :class="{ 'dash-refresh__icon--spin': dashboardStore.loading }">
            <RefreshRight />
          </el-icon>
        </button>
      </el-tooltip>
    </div>

    <div class="dash-grid dash-grid--kpi" :style="{ '--kpi-cols': String(kpiColumnCount) }">
      <el-card
        v-if="isConsoleAdmin"
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
        v-else
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card snapshot-card--static"
      >
        <div class="snapshot-label">业务服务（台账）</div>
        <div class="snapshot-value">{{ dash?.serviceStatusStats?.total ?? 0 }}</div>
        <div class="snapshot-foot">
          运行 {{ dash?.serviceStatusStats?.running ?? 0 }} · 部署中 {{ dash?.serviceStatusStats?.deploying ?? 0 }} · 停止
          {{ dash?.serviceStatusStats?.stopped ?? 0 }} · 异常 {{ dash?.serviceStatusStats?.error ?? 0 }}
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
        <div class="snapshot-foot">{{ exec24hFoot }}</div>
      </el-card>

      <el-card
        v-if="isSuperAdmin"
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card snapshot-card--static"
      >
        <div class="snapshot-label">用户 / 审计日志</div>
        <div class="snapshot-value">
          {{ dash?.platformSummary?.usersTotal ?? 0 }}
          <span class="snapshot-muted"> / {{ dash?.platformSummary?.operationLogsTotal ?? 0 }}</span>
        </div>
        <div class="snapshot-foot">用户数 · 操作日志条数</div>
      </el-card>
    </div>

    <el-card v-loading="dashboardStore.loading" shadow="hover" class="dash-exec-health">
      <div class="dash-exec-health__inner">
        <span class="dash-exec-health__title">近 24h 执行健康</span>
        <div class="dash-exec-health__stats">
          <span>
            成功 <strong>{{ execSuccess24h }}</strong> · 已取消 <strong>{{ execCancelled24h }}</strong> · 失败
            <strong>{{ execFailed24h }}</strong>
            <template v-if="execTerminal24h > 0">
              · 终态失败率 <strong>{{ execFailRateTerminalPct }}%</strong>
            </template>
          </span>
          <span class="dash-exec-health__sep" aria-hidden="true">|</span>
          <span class="dash-exec-health__by-src">
            按来源 CLI <strong>{{ execSrc.cli }}</strong> · K8s <strong>{{ execSrc.k8s }}</strong> · Job
            <strong>{{ execSrc.job }}</strong>
          </span>
          <el-link type="primary" :underline="false" class="dash-exec-health__link" @click="goExecRecords">执行记录</el-link>
        </div>
      </div>
    </el-card>

    <div v-if="isSuperAdmin" class="dash-grid dash-grid--main dash-grid--main--meters">
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="meter-card meter-card--cpu">
        <div class="meter-head">
          <span>服务端 CPU</span>
          <el-tag :type="getUsageType(dash?.resourceUsage?.cpu ?? 0)" size="small">
            {{ Number(dash?.resourceUsage?.cpu ?? 0).toFixed(1) }}%
          </el-tag>
        </div>
        <p class="meter-sub">{{ hostRuntimeLine }}</p>
        <p v-if="dash?.hostRuntime?.error" class="meter-sub meter-sub--err">{{ dash.hostRuntime.error }}</p>
        <el-progress
          :percentage="Math.round(clampPct(dash?.resourceUsage?.cpu ?? 0))"
          :color="getUsageColor(dash?.resourceUsage?.cpu ?? 0)"
          :show-text="false"
          class="meter-progress"
        />
      </el-card>
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="meter-card meter-card--mem">
        <div class="meter-head">
          <span>服务端内存</span>
          <el-tag :type="getUsageType(dash?.resourceUsage?.memory ?? 0)" size="small">
            {{ Number(dash?.resourceUsage?.memory ?? 0).toFixed(1) }}%
          </el-tag>
        </div>
        <p class="meter-sub">{{ hostRuntimeLine }}</p>
        <el-progress
          :percentage="Math.round(clampPct(dash?.resourceUsage?.memory ?? 0))"
          :color="getUsageColor(dash?.resourceUsage?.memory ?? 0)"
          :show-text="false"
          class="meter-progress"
        />
      </el-card>
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="meter-card meter-card--disk">
        <div class="meter-head">
          <span>服务端磁盘</span>
          <el-tag :type="getUsageType(dash?.resourceUsage?.disk ?? 0)" size="small">
            {{ Number(dash?.resourceUsage?.disk ?? 0).toFixed(1) }}%
          </el-tag>
        </div>
        <p class="meter-sub">{{ diskRootHint }}</p>
        <el-progress
          :percentage="Math.round(clampPct(dash?.resourceUsage?.disk ?? 0))"
          :color="getUsageColor(dash?.resourceUsage?.disk ?? 0)"
          :show-text="false"
          class="meter-progress"
        />
      </el-card>
    </div>

    <div
      v-loading="dashboardStore.loading"
      class="dash-tables-shell"
      element-loading-background="rgba(255,255,255,0.6)"
    >
      <div class="dash-tables-grid">
        <el-card v-if="isConsoleAdmin" shadow="always" class="dash-table-card">
          <template #header>
            <div class="table-card-head">
              <span class="table-card-head__title">最近 K8s 集群</span>
              <el-link type="primary" :underline="false" @click="goK8sClusters">列表</el-link>
            </div>
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
        </el-card>

        <el-card v-if="isConsoleAdmin" shadow="always" class="dash-table-card">
          <template #header>
            <div class="table-card-head">
              <span class="table-card-head__title">最近 ai-sre 安装</span>
              <el-link type="primary" :underline="false" @click="goServiceDeploy">服务部署</el-link>
            </div>
          </template>
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
        </el-card>

        <el-card shadow="always" class="dash-table-card">
          <template #header>
            <div class="table-card-head">
              <span class="table-card-head__title">最近 Linux / 业务服务</span>
              <el-link type="primary" :underline="false" @click="navigateToServiceList">台账</el-link>
            </div>
          </template>
          <el-table
            :data="dash?.recentDeployments ?? []"
            stripe
            border
            size="small"
            table-layout="fixed"
            class="dash-table--dense"
            :max-height="tableMaxPx"
            empty-text="暂无服务"
          >
            <el-table-column prop="name" label="服务名称" min-width="96" show-overflow-tooltip />
            <el-table-column prop="productName" label="功能名称" min-width="108" show-overflow-tooltip />
            <el-table-column prop="resource" label="资源" min-width="140" show-overflow-tooltip />
            <el-table-column prop="replicas" label="副本" width="64" align="center" />
            <el-table-column prop="status" label="状态" width="84" align="center">
              <template #default="scope">
                <el-tag :type="getStatusType(scope.row.status)" size="small">
                  {{ getStatusText(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="updateTime" label="更新时间" min-width="128" show-overflow-tooltip>
              <template #default="scope">
                {{ formatTs(scope.row.updateTime) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>

        <el-card shadow="always" class="dash-table-card">
          <template #header>
            <div class="table-card-head">
              <span class="table-card-head__title">最近执行记录</span>
              <el-link type="primary" :underline="false" @click="goExecRecords">全部</el-link>
            </div>
          </template>
          <el-table
            :data="dash?.recentExecutions ?? []"
            stripe
            border
            size="small"
            table-layout="fixed"
            class="dash-table--dense"
            :max-height="tableMaxPx"
            empty-text="暂无记录"
          >
            <el-table-column prop="name" label="名称" min-width="110" show-overflow-tooltip />
            <el-table-column prop="source" label="来源" width="72" show-overflow-tooltip />
            <el-table-column prop="category" label="类别" width="88" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="80" align="center">
              <template #default="scope">
                <el-tag :type="genericStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="targetHost" label="目标" min-width="96" show-overflow-tooltip />
            <el-table-column prop="finishedAt" label="结束时间" min-width="128" show-overflow-tooltip>
              <template #default="scope">
                {{ formatTs(scope.row.finishedAt) }}
              </template>
            </el-table-column>
            <el-table-column prop="durationMs" label="耗时" width="72" align="right">
              <template #default="scope">
                {{ formatDuration(scope.row.durationMs) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshRight } from '@element-plus/icons-vue'
import { useDashboardStore } from '../stores/dashboard'
import type { DashboardData } from '../types/dashboard'

const dashboardStore = useDashboardStore()
const router = useRouter()
const route = useRoute()

const shellPrefix = computed(() => (route.path.startsWith('/admin') ? '/admin' : '/app'))

const userRole = computed(() => {
  try {
    return String((JSON.parse(localStorage.getItem('userInfo') || '{}') as { role?: string }).role ?? '')
  } catch {
    return ''
  }
})

const isSuperAdmin = computed(() => userRole.value === 'super_admin')
const isConsoleAdmin = computed(() => userRole.value === 'admin' || userRole.value === 'super_admin')

const kpiColumnCount = computed(() => (isSuperAdmin.value ? 4 : 3))

const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const hostRuntimeLine = computed(() => {
  const h = dash.value?.hostRuntime
  if (!h?.hostname) return '本机（运行 opsfleet-backend）'
  const bits = [h.hostname]
  if (h.sampledAt) bits.push(formatTs(h.sampledAt))
  if (h.os) bits.push(h.os)
  return bits.join(' · ')
})

const diskRootHint = computed(() => '根分区使用率（Linux 为 /）')

const exec24hFoot = computed(() => {
  const total = dash.value?.platformSummary?.executionsLast24h ?? 0
  if (total <= 0) return '执行记录'
  return `新建 ${total} 条（含进行中/排队）`
})

const platformS = computed(() => dash.value?.platformSummary)

const execSuccess24h = computed(() => platformS.value?.executionsSuccessLast24h ?? 0)
const execCancelled24h = computed(() => platformS.value?.executionsCancelledLast24h ?? 0)
const execFailed24h = computed(() => platformS.value?.executionsFailedLast24h ?? 0)

const execTerminal24h = computed(
  () => execSuccess24h.value + execCancelled24h.value + execFailed24h.value
)

const execFailRateTerminalPct = computed(() => {
  const t = execTerminal24h.value
  if (t <= 0) return '0.0'
  const f = execFailed24h.value
  return ((f / t) * 100).toFixed(1)
})

const execSrc = computed(() => {
  const b = platformS.value?.executionsBySourceLast24h
  return {
    cli: b?.cli ?? 0,
    k8s: b?.k8s ?? 0,
    job: b?.job ?? 0
  }
})

const tableMaxPx = ref(240)
const recalcTableMax = () => {
  const h = typeof window !== 'undefined' ? window.innerHeight : 900
  tableMaxPx.value = Math.min(320, Math.max(180, Math.floor(h * 0.26)))
}

const goJobCenter = () => {
  router.push(`${shellPrefix.value}/job/center`)
}
const goK8sClusters = () => {
  if (!isConsoleAdmin.value) return
  router.push('/admin/service/k8s/clusters')
}
const goExecRecords = () => {
  router.push(`${shellPrefix.value}/execution-records`)
}
const goServiceDeploy = () => {
  if (!isConsoleAdmin.value) {
    ElMessage.info('服务部署仅限管理员')
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

.dash-refresh {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border-radius: 10px;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    color 0.15s ease,
    box-shadow 0.15s ease;
}

.dash-refresh:hover:not(:disabled) {
  border-color: var(--el-color-primary-light-5);
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.dash-refresh:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.dash-refresh__icon {
  font-size: 18px;
}

.dash-refresh__icon--spin {
  animation: dash-spin 0.85s linear infinite;
}

@keyframes dash-spin {
  to {
    transform: rotate(360deg);
  }
}

.meter-sub {
  margin: 0 0 6px;
  font-size: 11px;
  line-height: 1.35;
  color: var(--el-text-color-secondary);
}

.meter-sub--err {
  color: var(--el-color-danger);
  margin-top: -2px;
}

.dash-grid--kpi {
  grid-template-columns: repeat(var(--kpi-cols, 4), minmax(0, 1fr));
  align-items: stretch;
}

.dash-grid--kpi > .el-card {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.dash-grid--kpi > .el-card :deep(.el-card__body) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.dash-grid--kpi > .snapshot-card :deep(.el-card__body) > .snapshot-foot {
  margin-top: auto;
}

.dash-grid--main--meters {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.dash-grid {
  display: grid;
  gap: 10px;
  flex-shrink: 0;
}

.dash-grid--main {
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

.dash-tables-shell {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: auto;
}

.dash-tables-grid {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
  align-content: start;
}

.dash-table-card.el-card {
  min-width: 0;
  overflow: hidden;
  background-color: var(--of-surface, #ffffff) !important;
  box-shadow:
    0 0 0 1px rgba(29, 29, 31, 0.06),
    0 4px 24px rgba(0, 0, 0, 0.06) !important;
}

.dash-table-card :deep(.el-card__header) {
  padding: 12px 14px !important;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background-color: var(--of-surface, #ffffff) !important;
}

.dash-table-card :deep(.el-card__body) {
  padding: 12px 14px !important;
  background-color: var(--of-surface, #ffffff) !important;
}

.dash-exec-health {
  flex-shrink: 0;
}

.dash-exec-health :deep(.el-card__body) {
  padding: 10px 14px !important;
}

.dash-exec-health__inner {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px 12px;
  font-size: 13px;
  line-height: 1.45;
}

.dash-exec-health__title {
  font-weight: 600;
  color: var(--apple-ink, #303133);
  flex-shrink: 0;
}

.dash-exec-health__stats {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px 10px;
  color: var(--el-text-color-regular);
  font-size: 12px;
  min-width: 0;
  flex: 1;
  width: 100%;
}

.dash-exec-health__stats strong {
  font-weight: 600;
  color: var(--apple-ink, #303133);
}

.dash-exec-health__sep {
  color: var(--el-border-color);
  user-select: none;
}

.dash-exec-health__by-src {
  color: var(--el-text-color-secondary);
}

.dash-exec-health__link {
  margin-left: auto;
  flex-shrink: 0;
}

.dash-table--dense :deep(.el-table__cell) {
  padding: 6px 0;
}

.table-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.table-card-head__title {
  font-weight: 600;
  font-size: 14px;
  line-height: 1.35;
  color: var(--apple-ink, #303133);
  min-width: 0;
}

@media screen and (max-width: 1280px) {
  .dash-grid--main--meters {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media screen and (max-width: 720px) {
  .dash-grid--kpi {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dash-grid--main--meters {
    grid-template-columns: 1fr;
  }
}
</style>
