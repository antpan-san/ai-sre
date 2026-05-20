<template>
  <div class="dashboard page-shell page-shell--fill">
    <header class="dash-header">
      <h2 class="dash-header__title">概览</h2>
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
    </header>

    <section class="dash-summary" v-loading="dashboardStore.loading">
      <div class="kpi-row" :class="kpiRowClass">
        <article
          v-if="isConsoleAdmin"
          class="kpi-tile kpi-tile--link"
          role="button"
          tabindex="0"
          @click="goK8sClusters"
          @keydown.enter="goK8sClusters"
        >
          <span class="kpi-tile__label">K8s 集群</span>
          <div class="kpi-tile__value">
            {{ dash?.platformSummary?.k8sClusters?.running ?? 0 }}
            <span class="kpi-tile__muted">/ {{ dash?.platformSummary?.k8sClusters?.total ?? 0 }}</span>
          </div>
          <div class="kpi-tile__meta">
            <span>待定 {{ dash?.platformSummary?.k8sClusters?.pending ?? 0 }}</span>
            <span>失败 {{ dash?.platformSummary?.k8sClusters?.failed ?? 0 }}</span>
          </div>
        </article>

        <article v-else class="kpi-tile">
          <span class="kpi-tile__label">业务服务</span>
          <div class="kpi-tile__value">{{ dash?.serviceStatusStats?.total ?? 0 }}</div>
          <div class="kpi-tile__meta">
            <span>运行 {{ dash?.serviceStatusStats?.running ?? 0 }}</span>
            <span>部署 {{ dash?.serviceStatusStats?.deploying ?? 0 }}</span>
            <span>停止 {{ dash?.serviceStatusStats?.stopped ?? 0 }}</span>
            <span v-if="(dash?.serviceStatusStats?.error ?? 0) > 0" class="kpi-tile__warn">
              异常 {{ dash?.serviceStatusStats?.error ?? 0 }}
            </span>
          </div>
        </article>

        <article
          class="kpi-tile kpi-tile--link"
          role="button"
          tabindex="0"
          @click="goJobCenter"
          @keydown.enter="goJobCenter"
        >
          <span class="kpi-tile__label">进行中作业</span>
          <div class="kpi-tile__value">{{ dash?.platformSummary?.tasksActive ?? 0 }}</div>
          <span class="kpi-tile__hint">作业中心</span>
        </article>

        <article
          class="kpi-tile kpi-tile--link kpi-tile--exec"
          role="button"
          tabindex="0"
          @click="goExecRecords"
          @keydown.enter="goExecRecords"
        >
          <span class="kpi-tile__label">近 24h ai-sre 执行</span>
          <div class="kpi-tile__value">{{ dash?.platformSummary?.executionsLast24h ?? 0 }}</div>
          <div class="kpi-tile__meta kpi-tile__meta--exec">
            <span>成功 {{ execSuccess24h }}</span>
            <span>失败 {{ execFailed24h }}</span>
            <span>取消 {{ execCancelled24h }}</span>
            <span v-if="execTerminal24h > 0">失败率 {{ execFailRateTerminalPct }}%</span>
          </div>
          <div class="kpi-tile__meta kpi-tile__meta--sub">
            CLI {{ execSrc.cli }} · K8s {{ execSrc.k8s }} · Job {{ execSrc.job }}
          </div>
        </article>

        <article v-if="isSuperAdmin" class="kpi-tile">
          <span class="kpi-tile__label">用户 / 审计</span>
          <div class="kpi-tile__value">
            {{ dash?.platformSummary?.usersTotal ?? 0 }}
            <span class="kpi-tile__muted">/ {{ dash?.platformSummary?.operationLogsTotal ?? 0 }}</span>
          </div>
          <span class="kpi-tile__hint">用户数 · 操作日志条数</span>
        </article>
      </div>

    </section>

    <section class="dash-panels" :class="panelsClass" v-loading="dashboardStore.loading">
      <el-card v-if="isConsoleAdmin" shadow="never" class="panel-card panel-card--k8s">
        <template #header>
          <div class="panel-head">
            <span>最近 K8s 集群</span>
            <el-link type="primary" :underline="false" @click="goK8sClusters">列表</el-link>
          </div>
        </template>
        <div class="panel-body">
          <el-table :data="dash?.recentK8sClusters ?? []" stripe size="small" empty-text="暂无记录">
            <el-table-column prop="clusterName" label="名称" min-width="120" show-overflow-tooltip />
            <el-table-column prop="version" label="版本" width="88" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="88" align="center">
              <template #default="scope">
                <el-tag :type="clusterStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="masterNode" label="主节点" min-width="110" show-overflow-tooltip />
            <el-table-column prop="updatedAt" label="更新" width="140">
              <template #default="scope">{{ formatTs(scope.row.updatedAt) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>

      <el-card v-if="isConsoleAdmin" shadow="never" class="panel-card panel-card--install">
        <template #header>
          <div class="panel-head">
            <span>最近 ai-sre 安装</span>
            <el-link type="primary" :underline="false" @click="goServiceDeploy">服务部署</el-link>
          </div>
        </template>
        <div class="panel-body">
          <el-table :data="dash?.recentServiceInstalls ?? []" stripe size="small" empty-text="暂无记录">
            <el-table-column prop="service" label="组件" width="100" show-overflow-tooltip />
            <el-table-column prop="profile" label="配置" width="92" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="84" align="center">
              <template #default="scope">
                <el-tag :type="genericStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="currentStep" label="步骤" min-width="120" show-overflow-tooltip />
            <el-table-column prop="updatedAt" label="更新" width="140">
              <template #default="scope">{{ formatTs(scope.row.updatedAt) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>

      <el-card shadow="never" class="panel-card panel-card--services">
        <template #header>
          <div class="panel-head">
            <span>最近 Linux / 业务服务</span>
            <el-link type="primary" :underline="false" @click="navigateToServiceList">台账</el-link>
          </div>
        </template>
        <div class="panel-body">
          <el-table
            :data="dash?.recentDeployments ?? []"
            stripe
            size="small"
            table-layout="fixed"
            empty-text="暂无服务"
          >
            <el-table-column prop="name" label="服务" min-width="96" show-overflow-tooltip />
            <el-table-column prop="productName" label="功能" min-width="100" show-overflow-tooltip />
            <el-table-column prop="resource" label="资源" min-width="120" show-overflow-tooltip />
            <el-table-column prop="replicas" label="副本" width="56" align="center" />
            <el-table-column prop="status" label="状态" width="76" align="center">
              <template #default="scope">
                <el-tag :type="getStatusType(scope.row.status)" size="small">
                  {{ getStatusText(scope.row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="updateTime" label="更新" width="132" show-overflow-tooltip>
              <template #default="scope">{{ formatTs(scope.row.updateTime) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>

      <el-card shadow="never" class="panel-card panel-card--exec">
        <template #header>
          <div class="panel-head">
            <span>最近 ai-sre 执行</span>
            <el-link type="primary" :underline="false" @click="goExecRecords">全部</el-link>
          </div>
        </template>
        <div class="panel-body">
          <el-table
            :data="dash?.recentExecutions ?? []"
            stripe
            size="small"
            table-layout="fixed"
            empty-text="暂无记录"
          >
            <el-table-column prop="name" label="名称" min-width="100" show-overflow-tooltip />
            <el-table-column prop="source" label="来源" width="68" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="76" align="center">
              <template #default="scope">
                <el-tag :type="genericStatusType(scope.row.status)" size="small">
                  {{ scope.row.status || '—' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="目标" min-width="120" show-overflow-tooltip>
              <template #default="scope">{{ displayExecutionTarget(scope.row) }}</template>
            </el-table-column>
            <el-table-column prop="finishedAt" label="结束" width="132" show-overflow-tooltip>
              <template #default="scope">{{ formatTs(scope.row.finishedAt) }}</template>
            </el-table-column>
            <el-table-column prop="durationMs" label="耗时" width="68" align="right">
              <template #default="scope">{{ formatDuration(scope.row.durationMs) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshRight } from '@element-plus/icons-vue'
import { useDashboardStore } from '../stores/dashboard'
import type { DashboardData } from '../types/dashboard'
import { displayExecutionTarget } from '../utils/executionRecordDisplay'

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

const kpiRowClass = computed(() => {
  if (isSuperAdmin.value) return 'kpi-row--super'
  if (isConsoleAdmin.value) return 'kpi-row--admin'
  return 'kpi-row--app'
})

const panelsClass = computed(() =>
  isConsoleAdmin.value ? 'dash-panels--console' : 'dash-panels--app'
)

const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

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
  return ((execFailed24h.value / t) * 100).toFixed(1)
})

const execSrc = computed(() => {
  const b = platformS.value?.executionsBySourceLast24h
  return {
    cli: b?.cli ?? 0,
    k8s: b?.k8s ?? 0,
    job: b?.job ?? 0
  }
})

const goJobCenter = () => {
  router.push(`${shellPrefix.value}/job/center`)
}
const goK8sClusters = () => {
  if (!isConsoleAdmin.value) return
  router.push({ path: '/admin/execution-records', query: { tab: 'k8s' } })
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
  if (!data) ElMessage.error('获取仪表盘数据失败')
}

const handleRefresh = () => {
  void fetchDashboardData()
}

const formatTs = (iso?: string) => {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const formatDuration = (ms?: number) => {
  if (ms == null || ms <= 0) return '—'
  if (ms < 1000) return `${ms}ms`
  const s = ms / 1000
  if (s < 60) return `${s.toFixed(1)}s`
  const m = Math.floor(s / 60)
  return `${m}m${(s - m * 60).toFixed(0)}s`
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
  void fetchDashboardData()
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: 10px;
}

.dash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.dash-header__title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  line-height: 1.25;
}

.dash-refresh {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: 8px;
  border: 1px solid var(--el-border-color);
  background: var(--el-fill-color-blank);
  color: var(--el-text-color-regular);
  cursor: pointer;
}

.dash-refresh:hover:not(:disabled) {
  border-color: var(--el-color-primary-light-5);
  color: var(--el-color-primary);
}

.dash-refresh:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.dash-refresh__icon--spin {
  animation: dash-spin 0.85s linear infinite;
}

@keyframes dash-spin {
  to {
    transform: rotate(360deg);
  }
}

.dash-summary {
  flex-shrink: 0;
}

.kpi-row {
  display: grid;
  gap: 8px;
  align-items: stretch;
}

.kpi-row--super {
  grid-template-columns: 1.05fr 0.72fr 1.4fr 0.95fr;
}

.kpi-row--admin {
  grid-template-columns: 1.1fr 0.75fr 1.35fr;
}

.kpi-row--app {
  grid-template-columns: 1fr 0.75fr 1.35fr;
}

.kpi-tile {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 88px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
  min-width: 0;
  box-sizing: border-box;
}

.kpi-tile--link {
  cursor: pointer;
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
}

.kpi-tile--link:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.kpi-tile--exec {
  min-width: 0;
}

.kpi-tile > .kpi-tile__meta:first-of-type,
.kpi-tile > .kpi-tile__hint {
  margin-top: auto;
}

.kpi-tile__label {
  display: block;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}

.kpi-tile__value {
  margin-top: 2px;
  font-size: 22px;
  font-weight: 700;
  line-height: 1.15;
  color: var(--el-text-color-primary);
}

.kpi-tile__muted {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}

.kpi-tile__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 8px;
  margin-top: 6px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}

.kpi-tile__meta--exec span {
  white-space: nowrap;
}

.kpi-tile__meta--sub {
  margin-top: 2px;
  font-size: 10px;
  opacity: 0.9;
}

.kpi-tile__warn {
  color: var(--el-color-danger);
}

.kpi-tile__hint {
  display: block;
  margin-top: 4px;
  font-size: 10px;
  color: var(--el-text-color-placeholder);
}

.dash-panels {
  flex: 1;
  min-height: 0;
  display: grid;
  gap: 8px;
}

.dash-panels--console {
  grid-template-columns: repeat(12, minmax(0, 1fr));
  grid-template-rows: minmax(0, 1fr) minmax(0, 1fr);
  align-items: stretch;
}

.dash-panels--app {
  grid-template-columns: minmax(0, 0.82fr) minmax(0, 1.18fr);
  grid-template-rows: minmax(0, 1fr);
}

.panel-card--k8s {
  grid-column: 1 / 7;
  grid-row: 1;
  min-height: 0;
}

.panel-card--install {
  grid-column: 7 / 13;
  grid-row: 1;
  min-height: 0;
}

.panel-card--services {
  grid-column: 1 / 5;
  grid-row: 2;
}

.panel-card--exec {
  grid-column: 5 / 13;
  grid-row: 2;
}

.dash-panels--app .panel-card--services {
  grid-column: 1;
  grid-row: 1;
}

.dash-panels--app .panel-card--exec {
  grid-column: 2;
  grid-row: 1;
}

.panel-card {
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-card :deep(.el-card__header) {
  flex-shrink: 0;
  padding: 8px 12px !important;
}

.panel-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  padding: 0 12px 10px !important;
  display: flex;
  flex-direction: column;
}

.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  font-size: 13px;
  font-weight: 600;
}

.panel-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.panel-body :deep(.el-table) {
  font-size: 12px;
}

.panel-body :deep(.el-table__cell) {
  padding: 5px 0;
}

@media screen and (max-width: 1100px) {
  .kpi-row--super,
  .kpi-row--admin,
  .kpi-row--app {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .kpi-tile--exec {
    grid-column: 1 / -1;
  }

  .dash-panels--console,
  .dash-panels--app {
    grid-template-columns: 1fr;
    grid-template-rows: repeat(auto-fit, minmax(200px, 1fr));
  }

  .panel-card--k8s,
  .panel-card--install,
  .panel-card--services,
  .panel-card--exec,
  .dash-panels--app .panel-card--services,
  .dash-panels--app .panel-card--exec {
    grid-column: 1;
    grid-row: auto;
  }
}

@media screen and (max-width: 560px) {
  .kpi-row--super,
  .kpi-row--admin,
  .kpi-row--app {
    grid-template-columns: 1fr;
  }
}
</style>
