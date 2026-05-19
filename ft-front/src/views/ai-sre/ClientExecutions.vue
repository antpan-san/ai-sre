<template>
  <div class="client-executions page-shell page-shell--crud-wide">
    <header class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">客户端执行</h2>
        <p class="page-desc--muted">
          每次 ai-sre CLI 执行沉淀为一条会话：命令、采集证据、诊断结论、AI 与技能包链路。
        </p>
      </div>
    </header>

    <section v-loading="statsLoading" class="stats-row">
      <article class="stat-tile">
        <span class="stat-label">近 24h 执行</span>
        <strong>{{ stats?.total_24h ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">成功</span>
        <strong>{{ stats?.success_24h ?? 0 }}</strong>
      </article>
      <article class="stat-tile stat-tile--warn">
        <span class="stat-label">失败</span>
        <strong>{{ stats?.failed_24h ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">AI 调用</span>
        <strong>{{ stats?.ai_calls_24h ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">自动迭代</span>
        <strong>{{ stats?.auto_iteration_24h ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">证据不完整</span>
        <strong>{{ stats?.incomplete_evidence_24h ?? 0 }}</strong>
      </article>
      <article v-if="isSuperAdmin" class="stat-tile">
        <span class="stat-label">样本入库</span>
        <strong>{{ stats?.skill_samples_24h ?? 0 }}</strong>
      </article>
      <article class="stat-tile stat-tile--ok">
        <span class="stat-label">本地规则</span>
        <strong>{{ stats?.rule_hit_24h ?? 0 }}</strong>
      </article>
      <article v-if="isSuperAdmin" class="stat-tile stat-tile--warn">
        <span class="stat-label">待增强</span>
        <strong>{{ stats?.enhancement_open_24h ?? 0 }}</strong>
      </article>
    </section>

    <el-tabs v-model="activeView" class="view-tabs" @tab-change="onViewChange">
      <el-tab-pane label="全部" name="all" />
      <el-tab-pane label="诊断 check" name="check" />
      <el-tab-pane label="采集 probe" name="probe" />
      <el-tab-pane label="部署/初始化" name="deploy" />
      <el-tab-pane label="失败/待处理" name="failed" />
      <el-tab-pane label="触发自动迭代" name="auto_iteration" />
      <el-tab-pane label="已入样本池" name="skill_sample" />
      <el-tab-pane label="待增强" name="enhancement" />
    </el-tabs>

    <div class="toolbar">
      <el-input v-model="filters.target" clearable placeholder="目标" class="filter-input" @keyup.enter="reload" />
      <el-input v-model="filters.topic" clearable placeholder="Topic" class="filter-input" @keyup.enter="reload" />
      <el-select v-model="filters.status" clearable placeholder="状态" class="filter-select" @change="reload">
        <el-option label="成功" value="success" />
        <el-option label="失败" value="failed" />
        <el-option label="执行中" value="running" />
        <el-option label="已取消" value="cancelled" />
      </el-select>
      <el-button type="primary" @click="reload">搜索</el-button>
      <el-button @click="resetFilters">重置</el-button>
    </div>

    <el-card shadow="never" v-loading="loading">
      <el-table :data="rows" stripe border size="small" empty-text="暂无客户端执行记录" @row-click="openDetail">
        <el-table-column label="时间" width="168">
          <template #default="{ row }">{{ formatTime(row.time) }}</template>
        </el-table-column>
        <el-table-column label="命令" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">
            <div class="cmd-line">{{ row.command || row.normalized_command }}</div>
            <el-tag v-if="row.legacy_kind" size="small" type="info" class="legacy-tag">{{ legacyLabel(row.legacy_kind) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="target" label="目标" min-width="120" show-overflow-tooltip />
        <el-table-column label="Topic / 技能包" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">{{ row.topic || '—' }} / {{ row.skill_pack || '—' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="88" align="center">
          <template #default="{ row }">
            <el-tag :type="statusTag(row.status)" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="规则/AI" width="108" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.rule_hit" size="small" type="success">本地规则</el-tag>
            <span v-else-if="row.used_ai">{{ aiSourceLabel(row.ai_source) }}</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="沉淀/增强" width="108" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.skill_sample_recorded" size="small" type="info">样本</el-tag>
            <el-tag v-if="row.enhancement_needs || row.enhancement_review_triggered" :type="enhancementTag(row.enhancement_priority)" size="small">
              {{ enhancementLabel(row.enhancement_priority) }}
            </el-tag>
            <el-tag v-if="row.has_auto_iteration" size="small" type="warning">迭代</el-tag>
            <span v-if="!row.skill_sample_recorded && !row.enhancement_needs && !row.has_auto_iteration" class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column prop="summary" label="结论摘要" min-width="160" show-overflow-tooltip />
        <el-table-column label="证据" width="88" align="center">
          <template #default="{ row }">
            <el-tag :type="evidenceTag(row.evidence_completeness)" size="small">{{ row.evidence_completeness || '—' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="用户 / 机器" min-width="130" show-overflow-tooltip>
          <template #default="{ row }">{{ row.user || '—' }} · {{ row.machine || '—' }}</template>
        </el-table-column>
        <el-table-column label="耗时" width="72" align="right">
          <template #default="{ row }">{{ formatDuration(row.duration_ms) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="88" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="openDetail(row)">复盘</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          small
          @current-change="reload"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getAISreExecutionStats,
  listAISreExecutions,
  type ClientExecutionListItem,
  type ClientExecutionStats
} from '../../api/aisreExecutions'

const router = useRouter()
const route = useRoute()
const shellPrefix = route.path.startsWith('/app') ? '/app' : '/admin'

const isSuperAdmin = computed(() => {
  try {
    const u = JSON.parse(localStorage.getItem('userInfo') || '{}')
    return u?.role === 'super_admin'
  } catch {
    return false
  }
})

const loading = ref(false)
const statsLoading = ref(false)
const rows = ref<ClientExecutionListItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const stats = ref<ClientExecutionStats | null>(null)
const activeView = ref('all')
const filters = reactive({ target: '', topic: '', status: '' })

const loadStats = async () => {
  statsLoading.value = true
  try {
    const data = await getAISreExecutionStats(24)
    stats.value = data.stats
  } catch {
    stats.value = null
  } finally {
    statsLoading.value = false
  }
}

const reload = async () => {
  loading.value = true
  try {
    const view = activeView.value === 'all' ? undefined : activeView.value
    const data = await listAISreExecutions({
      page: page.value,
      pageSize: pageSize.value,
      view,
      target: filters.target || undefined,
      topic: filters.topic || undefined,
      status: filters.status || undefined
    })
    rows.value = data.list || []
    total.value = data.total || 0
  } catch {
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.target = ''
  filters.topic = ''
  filters.status = ''
  page.value = 1
  void reload()
}

const onViewChange = () => {
  page.value = 1
  void reload()
}

const openDetail = (row: ClientExecutionListItem) => {
  router.push(`${shellPrefix}/ai-sre/executions/${row.id}`)
}

const formatTime = (t?: string) => (t ? String(t).replace('T', ' ').slice(0, 19) : '—')
const formatDuration = (ms?: number) => {
  if (ms == null || ms <= 0) return '—'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const statusTag = (s: string) => {
  switch (s) {
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'running':
      return 'warning'
    default:
      return 'info'
  }
}

const evidenceTag = (v?: string) => {
  switch (v) {
    case 'complete':
      return 'success'
    case 'partial':
      return 'warning'
    case 'missing':
    case 'incomplete':
      return 'danger'
    default:
      return 'info'
  }
}

const aiSourceLabel = (s?: string) => {
  switch (s) {
    case 'platform_ai':
      return '平台 AI'
    case 'local_rule':
      return '本地规则'
    case 'mixed':
      return '混合'
    default:
      return s || 'AI'
  }
}

const enhancementTag = (p?: string) => {
  switch (p) {
    case 'high':
      return 'danger'
    case 'medium':
      return 'warning'
    default:
      return 'info'
  }
}

const enhancementLabel = (p?: string) => {
  switch (p) {
    case 'high':
      return '待增强·高'
    case 'medium':
      return '待增强·中'
    case 'low':
      return '待增强·低'
    default:
      return '待增强'
  }
}

const legacyLabel = (k: string) => {
  switch (k) {
    case 'legacy_ai_diagnose':
      return '历史 AI 诊断'
    case 'legacy_go_runtime':
      return '历史运行时'
    case 'legacy_cli':
      return '历史 CLI'
    default:
      return k
  }
}

onMounted(() => {
  void loadStats()
  void reload()
})
</script>

<style scoped>
.page-head {
  margin-bottom: 14px;
}
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 10px;
  margin-bottom: 14px;
}
.stat-tile {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 10px 12px;
  background: var(--el-bg-color);
}
.stat-tile strong {
  display: block;
  font-size: 22px;
  margin-top: 4px;
}
.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 12px;
}
.filter-input {
  width: min(200px, 100%);
}
.filter-select {
  width: 140px;
}
.view-tabs {
  margin-bottom: 8px;
}
.cmd-line {
  font-size: 13px;
}
.legacy-tag {
  margin-top: 4px;
}
.pager {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
.muted {
  color: var(--el-text-color-secondary);
}
</style>
