<template>
  <div class="dashboard page-shell page-shell--dashboard">
    <div class="page-header">
      <div>
        <h2>仪表盘</h2>
        <p class="page-sub">
          数据来自当前租户下已入库的资产与任务；资源率为<strong>在线机器</strong>上报心跳的平均值；未对接集群内 Pod/带宽采集。
        </p>
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

    <!-- 平台快照 -->
    <div class="snapshot-row">
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="router.push('/init-tools')"
      >
        <div class="snapshot-label">托管机器</div>
        <div class="snapshot-value">
          {{ dash?.platformSummary?.machines?.online ?? 0 }}
          <span class="snapshot-muted">/ {{ dash?.platformSummary?.machines?.total ?? 0 }}</span>
        </div>
        <div class="snapshot-foot">离线 {{ dash?.platformSummary?.machines?.offline ?? 0 }}</div>
      </el-card>
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="router.push('/service/k8s/clusters')"
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
        @click="router.push('/job/center')"
      >
        <div class="snapshot-label">进行中任务</div>
        <div class="snapshot-value">{{ dash?.platformSummary?.tasksActive ?? 0 }}</div>
        <div class="snapshot-foot">作业中心</div>
      </el-card>
      <el-card
        v-loading="dashboardStore.loading"
        shadow="hover"
        class="snapshot-card"
        role="button"
        tabindex="0"
        @click="router.push('/execution-records')"
      >
        <div class="snapshot-label">近 24h 执行记录</div>
        <div class="snapshot-value">{{ dash?.platformSummary?.executionsLast24h ?? 0 }}</div>
        <div class="snapshot-foot">控制台 / CLI 汇总</div>
      </el-card>
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="snapshot-card snapshot-card--static">
        <div class="snapshot-label">用户 / 审计</div>
        <div class="snapshot-value">
          {{ dash?.platformSummary?.usersTotal ?? 0 }}
          <span class="snapshot-muted"> / {{ dash?.platformSummary?.operationLogsTotal ?? 0 }}</span>
        </div>
        <div class="snapshot-foot">用户·操作日志条数</div>
      </el-card>
    </div>

    <!-- 在线机器平均资源 -->
    <div class="resource-cards">
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>在线机器平均 CPU</span>
            <el-tag :type="getUsageType(dash?.resourceUsage?.cpu ?? 0)" size="small">
              {{ Number(dash?.resourceUsage?.cpu ?? 0).toFixed(1) }}%
            </el-tag>
          </div>
        </template>
        <div class="card-content">
          <el-progress
            :percentage="Math.round(clampPct(dash?.resourceUsage?.cpu ?? 0))"
            :color="getUsageColor(dash?.resourceUsage?.cpu ?? 0)"
            :show-text="false"
            class="progress-bar"
          />
        </div>
      </el-card>

      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>在线机器平均内存</span>
            <el-tag :type="getUsageType(dash?.resourceUsage?.memory ?? 0)" size="small">
              {{ Number(dash?.resourceUsage?.memory ?? 0).toFixed(1) }}%
            </el-tag>
          </div>
        </template>
        <div class="card-content">
          <el-progress
            :percentage="Math.round(clampPct(dash?.resourceUsage?.memory ?? 0))"
            :color="getUsageColor(dash?.resourceUsage?.memory ?? 0)"
            :show-text="false"
            class="progress-bar"
          />
        </div>
      </el-card>

      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>在线机器平均磁盘</span>
            <el-tag :type="getUsageType(dash?.resourceUsage?.disk ?? 0)" size="small">
              {{ Number(dash?.resourceUsage?.disk ?? 0).toFixed(1) }}%
            </el-tag>
          </div>
        </template>
        <div class="card-content">
          <el-progress
            :percentage="Math.round(clampPct(dash?.resourceUsage?.disk ?? 0))"
            :color="getUsageColor(dash?.resourceUsage?.disk ?? 0)"
            :show-text="false"
            class="progress-bar"
          />
        </div>
      </el-card>

      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>网络</span>
            <el-tag type="info" size="small">未采集</el-tag>
          </div>
        </template>
        <div class="card-content network-content">
          <p class="network-hint">
            入/出流量需 Agent 或监控侧上报；当前仅占位
            {{ formatNetwork(dash?.resourceUsage?.network?.in ?? 0) }} /
            {{ formatNetwork(dash?.resourceUsage?.network?.out ?? 0) }}。
          </p>
        </div>
      </el-card>
    </div>

    <div class="middle-section">
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="overview-card">
        <template #header>
          <div class="card-header card-header--wide">
            <span>平台概览（控制台台账）</span>
            <span class="card-hint">非 kube-apiserver 实时口径</span>
          </div>
        </template>
        <div class="overview-content">
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Grid /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dash?.kubernetesOverview?.nodes ?? 0 }}</div>
              <div class="overview-label">机器总数</div>
            </div>
          </div>
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Histogram /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dash?.kubernetesOverview?.pods ?? 0 }}</div>
              <div class="overview-label">K8s 集群条目</div>
            </div>
          </div>
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Check /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dash?.kubernetesOverview?.runningPods ?? 0 }}</div>
              <div class="overview-label">在线机器数</div>
            </div>
          </div>
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Link /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dash?.kubernetesOverview?.services ?? 0 }}</div>
              <div class="overview-label">Linux/服务条目</div>
            </div>
          </div>
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Upload /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dash?.kubernetesOverview?.deployments ?? 0 }}</div>
              <div class="overview-label">运行中集群数</div>
            </div>
          </div>
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Operation /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dash?.kubernetesOverview?.replicasets ?? 0 }}</div>
              <div class="overview-label">活跃作业任务</div>
            </div>
          </div>
        </div>
        <div class="topology-hint">
          拓扑角色：主控 {{ dash?.platformSummary?.machines?.masters ?? 0 }} · Worker
          {{ dash?.platformSummary?.machines?.workers ?? 0 }}
        </div>
      </el-card>

      <el-card v-loading="dashboardStore.loading" shadow="hover" class="stats-card">
        <template #header>
          <div class="card-header">
            <span>业务服务状态</span>
            <el-tag type="info" size="small"> 总计: {{ dash?.serviceStatusStats?.total ?? 0 }} </el-tag>
          </div>
        </template>
        <p class="stats-note">「运行中」含状态为 deploying 的服务。</p>
        <div class="stats-content">
          <div class="stats-item">
            <el-progress
              type="circle"
              :percentage="getStatusPercentage('running')"
              :color="getStatusColor('running')"
              :format="() => ''"
              :width="60"
            />
            <div class="stats-info">
              <div class="stats-value">{{ dash?.serviceStatusStats?.running ?? 0 }}</div>
              <div class="stats-label">运行中</div>
            </div>
          </div>
          <div class="stats-item">
            <el-progress
              type="circle"
              :percentage="getStatusPercentage('stopped')"
              :color="getStatusColor('stopped')"
              :format="() => ''"
              :width="60"
            />
            <div class="stats-info">
              <div class="stats-value">{{ dash?.serviceStatusStats?.stopped ?? 0 }}</div>
              <div class="stats-label">已停止</div>
            </div>
          </div>
          <div class="stats-item">
            <el-progress
              type="circle"
              :percentage="getStatusPercentage('error')"
              :color="getStatusColor('error')"
              :format="() => ''"
              :width="60"
            />
            <div class="stats-info">
              <div class="stats-value">{{ dash?.serviceStatusStats?.error ?? 0 }}</div>
              <div class="stats-label">异常</div>
            </div>
          </div>
        </div>
      </el-card>
    </div>

    <div class="tables-grid">
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="table-card">
        <template #header>
          <div class="card-header">
            <span>最近 K8s 集群</span>
            <el-link type="primary" :underline="false" @click="router.push('/service/k8s/clusters')">
              打开列表
            </el-link>
          </div>
        </template>
        <el-table :data="dash?.recentK8sClusters ?? []" stripe border size="small" empty-text="暂无记录">
          <el-table-column prop="clusterName" label="名称" min-width="120" show-overflow-tooltip />
          <el-table-column prop="version" label="版本" width="90" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="clusterStatusType(scope.row.status)" size="small">
                {{ scope.row.status || '—' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="masterNode" label="主节点" min-width="120" show-overflow-tooltip />
          <el-table-column prop="updatedAt" label="更新" min-width="150">
            <template #default="scope">
              {{ formatTs(scope.row.updatedAt) }}
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card v-loading="dashboardStore.loading" shadow="hover" class="table-card">
        <template #header>
          <div class="card-header">
            <span>最近 ai-sre 安装</span>
            <el-link type="primary" :underline="false" @click="router.push('/service/deploy')">
              服务部署
            </el-link>
          </div>
        </template>
        <el-table :data="dash?.recentServiceInstalls ?? []" stripe border size="small" empty-text="暂无记录">
          <el-table-column prop="service" label="组件" width="110" show-overflow-tooltip />
          <el-table-column prop="profile" label="配置" width="100" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="genericStatusType(scope.row.status)" size="small">
                {{ scope.row.status || '—' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="currentStep" label="步骤" min-width="140" show-overflow-tooltip />
          <el-table-column prop="updatedAt" label="更新" min-width="150">
            <template #default="scope">
              {{ formatTs(scope.row.updatedAt) }}
            </template>
          </el-table-column>
        </el-table>
      </el-card>
    </div>

    <el-card v-loading="dashboardStore.loading" shadow="hover" class="recent-deployments-card">
      <template #header>
        <div class="card-header">
          <span>最近 Linux / 业务服务（台账）</span>
          <el-link type="primary" :underline="false" @click="navigateToServiceList"> 查看全部 </el-link>
        </div>
      </template>
      <div class="recent-deployments-content">
        <el-table :data="dash?.recentDeployments ?? []" stripe border size="small" empty-text="暂无服务">
          <el-table-column prop="name" label="服务名称" min-width="150" show-overflow-tooltip />
          <el-table-column prop="image" label="镜像" min-width="180" show-overflow-tooltip />
          <el-table-column prop="replicas" label="副本" width="72" align="center" />
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="getStatusType(scope.row.status)" size="small">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="updateTime" label="更新时间" min-width="160">
            <template #default="scope">
              {{ formatTs(scope.row.updateTime) }}
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-card>

    <el-card v-loading="dashboardStore.loading" shadow="hover" class="recent-deployments-card">
      <template #header>
        <div class="card-header">
          <span>最近执行记录</span>
          <el-link type="primary" :underline="false" @click="router.push('/execution-records')"> 全部记录 </el-link>
        </div>
      </template>
      <el-table :data="dash?.recentExecutions ?? []" stripe border size="small" empty-text="暂无执行记录">
        <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
        <el-table-column prop="source" label="来源" width="90" show-overflow-tooltip />
        <el-table-column prop="category" label="类别" width="110" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="scope">
            <el-tag :type="genericStatusType(scope.row.status)" size="small">
              {{ scope.row.status || '—' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="targetHost" label="目标" min-width="120" show-overflow-tooltip />
        <el-table-column prop="finishedAt" label="结束时间" min-width="160">
          <template #default="scope">
            {{ formatTs(scope.row.finishedAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="durationMs" label="耗时" width="90" align="right">
          <template #default="scope">
            {{ formatDuration(scope.row.durationMs) }}
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  RefreshRight,
  Grid,
  Check,
  Link,
  Upload,
  Histogram,
  Operation
} from '@element-plus/icons-vue'
import { useDashboardStore } from '../stores/dashboard'
import type { DashboardData } from '../types/dashboard'

const dashboardStore = useDashboardStore()
const router = useRouter()

const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

onMounted(() => {
  void fetchDashboardData()
})

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

const formatNetwork = (value: number) => {
  if (value >= 1024) {
    return `${(value / 1024).toFixed(2)} GB`
  }
  return `${value.toFixed(2)} MB`
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

const getStatusPercentage = (status: 'running' | 'stopped' | 'error') => {
  const total = dash.value?.serviceStatusStats?.total ?? 0
  if (total === 0) return 0
  let count = 0
  switch (status) {
    case 'running':
      count = dash.value?.serviceStatusStats?.running ?? 0
      break
    case 'stopped':
      count = dash.value?.serviceStatusStats?.stopped ?? 0
      break
    case 'error':
      count = dash.value?.serviceStatusStats?.error ?? 0
      break
  }
  return Math.round((count / total) * 100)
}

const getStatusColor = (status: 'running' | 'stopped' | 'error') => {
  switch (status) {
    case 'running':
      return '#67c23a'
    case 'stopped':
      return '#909399'
    case 'error':
      return '#f56c6c'
    default:
      return '#ff6900'
  }
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
  router.push('/service/deploy')
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 16px;
}

.page-header h2 {
  margin: 0 0 6px;
  color: #303133;
}

.page-sub {
  margin: 0;
  max-width: 720px;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.snapshot-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}

.snapshot-card {
  cursor: pointer;
  transition: transform 0.12s ease, box-shadow 0.12s ease;
}

.snapshot-card:not(.snapshot-card--static):hover {
  transform: translateY(-2px);
}

.snapshot-card--static {
  cursor: default;
}

.snapshot-label {
  font-size: 13px;
  color: #909399;
}

.snapshot-value {
  margin-top: 6px;
  font-size: 22px;
  font-weight: 700;
  color: #303133;
}

.snapshot-muted {
  font-size: 15px;
  font-weight: 500;
  color: #909399;
}

.snapshot-foot {
  margin-top: 6px;
  font-size: 12px;
  color: #a8abb2;
}

.resource-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.resource-card {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
}

.card-header--wide {
  flex-wrap: wrap;
  gap: 8px;
}

.card-hint {
  font-size: 12px;
  font-weight: 400;
  color: #909399;
}

.card-content {
  padding: 20px 0;
}

.progress-bar {
  height: 10px;
}

.network-content {
  padding: 8px 0;
}

.network-hint {
  margin: 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.middle-section {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

.overview-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 20px;
}

.overview-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.overview-icon {
  color: var(--el-color-primary);
}

.overview-info {
  display: flex;
  flex-direction: column;
}

.overview-value {
  font-size: 20px;
  font-weight: bold;
  color: #303133;
}

.overview-label {
  color: #606266;
  font-size: 13px;
  line-height: 1.3;
}

.topology-hint {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 12px;
  color: #909399;
}

.stats-note {
  margin: 0 0 12px;
  font-size: 12px;
  color: #909399;
}

.stats-content {
  display: flex;
  justify-content: space-around;
  align-items: center;
}

.stats-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.stats-info {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stats-value {
  font-size: 18px;
  font-weight: bold;
  color: #303133;
}

.stats-label {
  color: #606266;
  font-size: 14px;
}

.tables-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.table-card {
  min-width: 0;
}

.recent-deployments-card {
  margin-bottom: 20px;
}

@media screen and (max-width: 1200px) {
  .middle-section {
    grid-template-columns: 1fr;
  }
}

@media screen and (max-width: 768px) {
  .resource-cards {
    grid-template-columns: 1fr;
  }

  .overview-content {
    grid-template-columns: repeat(2, 1fr);
  }

  .stats-content {
    flex-direction: column;
    gap: 20px;
  }
}
</style>
