<template>
  <div class="app-dashboard page-shell page-shell--fill">
    <header class="dash-header">
      <h2 class="dash-header__title">概览</h2>
      <el-button size="small" :loading="dashboardStore.loading" @click="refresh">刷新</el-button>
    </header>

    <section v-loading="dashboardStore.loading" class="dash-summary">
      <div class="kpi-row kpi-row--app">
        <article class="kpi-tile kpi-tile--link" @click="goExec">
          <span class="kpi-tile__label">近 24h 执行</span>
          <div class="kpi-tile__value">{{ dash?.platformSummary?.executionsLast24h ?? 0 }}</div>
          <div class="kpi-tile__meta">
            <span>成功 {{ execSuccess24h }}</span>
            <span class="kpi-tile__warn">失败 {{ execFailed24h }}</span>
          </div>
        </article>
        <article class="kpi-tile kpi-tile--link" @click="goJobs">
          <span class="kpi-tile__label">进行中作业</span>
          <div class="kpi-tile__value">{{ dash?.platformSummary?.tasksActive ?? 0 }}</div>
        </article>
        <article class="kpi-tile kpi-tile--link" @click="goCapabilities">
          <span class="kpi-tile__label">已订阅能力</span>
          <div class="kpi-tile__value">{{ subscribedCount }}</div>
          <el-link class="kpi-tile__link" type="primary" :underline="false" @click.stop="goSubscribeable">
            查看可订阅
          </el-link>
        </article>
        <article class="kpi-tile">
          <span class="kpi-tile__label">业务服务</span>
          <div class="kpi-tile__value">{{ dash?.serviceStatusStats?.running ?? 0 }}</div>
          <span class="kpi-tile__hint">运行中</span>
        </article>
      </div>
    </section>

    <section class="dash-panels dash-panels--app" v-loading="dashboardStore.loading">
      <el-card shadow="never" class="panel-card">
        <template #header>
          <div class="panel-head">
            <span>推荐下一步</span>
          </div>
        </template>
        <div class="next-actions">
          <el-button @click="router.push('/app/troubleshooting')">问题排查</el-button>
          <el-button @click="router.push('/app/deploy')">部署中心</el-button>
          <el-button @click="router.push('/app/deploy?expand=subscribe')">可订阅能力</el-button>
          <el-button @click="router.push('/app/settings')">安装 CLI</el-button>
        </div>
      </el-card>

      <el-card shadow="never" class="panel-card">
        <template #header>
          <div class="panel-head">
            <span>最近执行</span>
            <el-link type="primary" :underline="false" @click="goExec">全部</el-link>
          </div>
        </template>
        <el-table :data="dash?.recentExecutions ?? []" stripe size="small" empty-text="暂无记录">
          <el-table-column prop="name" label="名称" min-width="120" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="88">
            <template #default="{ row }">
              <el-tag size="small" :type="row.status === 'success' ? 'success' : row.status === 'failed' ? 'danger' : 'info'">
                {{ row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="finishedAt" label="结束" width="140">
            <template #default="{ row }">{{ formatTs(row.finishedAt) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card v-if="subscribedPreview.length" shadow="never" class="panel-card">
        <template #header>
          <div class="panel-head">
            <span>已订阅能力</span>
            <el-link type="primary" :underline="false" @click="goDeploy">部署中心</el-link>
          </div>
        </template>
        <ul class="sub-list">
          <li v-for="item in subscribedPreview" :key="item.id">{{ item.name }}</li>
        </ul>
      </el-card>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useDashboardStore } from '../../stores/dashboard'
import { useCapabilityCatalog } from '../../composables/useCapabilityCatalog'
import type { DashboardData } from '../../types/dashboard'

const router = useRouter()
const dashboardStore = useDashboardStore()
const { resolved, load: loadCaps } = useCapabilityCatalog()

const dash = computed<DashboardData | null>(() => dashboardStore.dashboardData)

const execSuccess24h = computed(() => dash.value?.platformSummary?.executionsSuccessLast24h ?? 0)
const execFailed24h = computed(() => dash.value?.platformSummary?.executionsFailedLast24h ?? 0)

const subscribedCount = computed(
  () => resolved.value.filter((c) => ['已订阅', '免费可用', '管理员已开通'].includes(c.status)).length
)
const subscribedPreview = computed(() =>
  resolved.value.filter((c) => c.status === '已订阅' || c.status === '免费可用').slice(0, 6)
)

const refresh = async () => {
  await dashboardStore.fetchDashboardData()
  await loadCaps()
}

const goExec = () => router.push('/app/execution-records')
const goJobs = () => router.push('/app/job/center')
const goDeploy = () => router.push('/app/deploy')
const goCapabilities = () => router.push('/app/deploy')
const goSubscribeable = () => router.push('/app/deploy?expand=subscribe')

const formatTs = (iso?: string) => {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

onMounted(() => {
  void refresh()
})
</script>

<style scoped>
.dash-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.dash-header__title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}
.dash-summary {
  margin-bottom: 16px;
}
.kpi-row {
  display: grid;
  gap: 12px;
}
.kpi-row--app {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}
@media (max-width: 960px) {
  .kpi-row--app {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
.kpi-tile {
  display: flex;
  flex-direction: column;
  min-height: 88px;
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-blank);
}
.kpi-tile--link {
  cursor: pointer;
}
.kpi-tile--link:hover {
  border-color: var(--el-color-primary-light-5);
}
.kpi-tile__label {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.kpi-tile__value {
  margin-top: 4px;
  font-size: 22px;
  font-weight: 700;
}
.kpi-tile__meta {
  margin-top: auto;
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.kpi-tile__warn {
  color: var(--el-color-danger);
}
.kpi-tile__hint {
  margin-top: auto;
  font-size: 10px;
  color: var(--el-text-color-placeholder);
}
.kpi-tile__link {
  margin-top: auto;
  font-size: 11px;
}
.dash-panels--app {
  display: grid;
  gap: 14px;
}
.panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.next-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.sub-list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
  line-height: 1.8;
}
</style>
