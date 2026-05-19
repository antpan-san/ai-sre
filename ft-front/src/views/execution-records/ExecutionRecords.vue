<template>
  <div class="execution-records page-shell">
    <header class="page-header page-header--row">
      <div class="page-header-copy">
        <h2 class="page-title">通用执行审计</h2>
        <el-popover placement="bottom-start" :width="280" trigger="click">
          <template #reference>
            <el-button text type="primary" size="small">说明</el-button>
          </template>
          <p class="page-desc--muted" style="margin: 0">
            脚本副本、作业、K8s 安装等非 ai-sre CLI 主路径的审计流水。CLI 复盘请使用同组「客户端执行」。
          </p>
        </el-popover>
      </div>
    </header>

    <el-tabs v-model="activePanel" class="exec-tabs" type="border-card" @tab-change="onTabChange">
      <el-tab-pane label="历史列表" name="records">
        <section class="search-filters">
          <el-input v-model="filters.keyword" placeholder="搜索命令 / 输出 / 名称" clearable :prefix-icon="Search" @keyup.enter="handleSearch" />
          <el-input v-model="filters.target" placeholder="目标主机 / 资源" clearable :prefix-icon="Search" @keyup.enter="handleSearch" />
          <el-select v-model="filters.source" placeholder="来源" clearable @change="handleSearch">
            <el-option label="AI 调用" value="ai" />
            <el-option label="安装 ai-sre" value="install" />
            <el-option label="ai-sre CLI" value="cli" />
            <el-option label="复制脚本" value="script" />
            <el-option label="初始化工具" value="init-tools" />
            <el-option label="作业中心" value="job" />
            <el-option label="K8s" value="k8s" />
            <el-option label="回滚" value="rollback" />
          </el-select>
          <el-select v-model="filters.status" placeholder="状态" clearable @change="handleSearch">
            <el-option label="等待中" value="pending" />
            <el-option label="执行中" value="running" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="failed" />
            <el-option label="已取消" value="cancelled" />
          </el-select>
          <el-select v-model="filters.category" placeholder="类型" clearable @change="handleSearch">
            <el-option label="Go Runtime" value="go_runtime" />
            <el-option label="AI 诊断" value="analyze" />
            <el-option label="check" value="check" />
            <el-option label="诊断任务单" value="diagnostic_plan" />
            <el-option label="AI 问答" value="ask" />
            <el-option label="Runbook" value="runbook" />
          </el-select>
          <el-select v-model="filters.rollbackCapability" placeholder="回滚能力" clearable @change="handleSearch">
            <el-option label="自动 / 半自动" value="auto" />
            <el-option label="人工建议" value="manual" />
            <el-option label="不可回滚" value="none" />
          </el-select>
          <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
          <el-button :icon="RefreshRight" @click="handleReset">重置</el-button>
        </section>

        <el-table v-loading="loading" :data="records" border stripe class="records-table">
          <el-table-column prop="created_at" label="时间" min-width="165">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="name" label="执行项" min-width="220">
            <template #default="{ row }">
              <div class="record-name">{{ row.name }}</div>
              <div class="record-command">{{ row.command || row.category }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="source" label="来源" width="110">
            <template #default="{ row }"><el-tag size="small">{{ sourceLabel(row.source) }}</el-tag></template>
          </el-table-column>
          <el-table-column label="账号 / 功能包" min-width="170">
            <template #default="{ row }">
              <div class="record-user">{{ row.trigger_user || row.created_by || '-' }}</div>
              <div class="record-pack">{{ packLabel(recordMeta(row).pack_key || recordMeta(row).skill_pack) }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="target_host" label="目标" min-width="150" show-overflow-tooltip>
            <template #default="{ row }">{{ displayExecutionTarget(row) }}</template>
          </el-table-column>
          <el-table-column prop="status" label="状态" width="100">
            <template #default="{ row }"><el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="rollback_capability" label="回滚" width="130">
            <template #default="{ row }">
              <el-tag :type="rollbackType(row.rollback_capability)" size="small">{{ rollbackLabel(row.rollback_capability) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="rollback_status" label="回滚状态" width="120">
            <template #default="{ row }">{{ rollbackStatusLabel(row.rollback_status) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="170" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="openDetail(row)">详情</el-button>
              <el-button type="warning" link :disabled="row.rollback_capability === 'none'" @click="previewRollback(row)">回滚</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pagination">
          <el-pagination
            v-model:current-page="filters.page"
            v-model:page-size="filters.pageSize"
            :page-sizes="[10, 20, 50, 100]"
            layout="total, sizes, prev, pager, next, jumper"
            :total="total"
            @size-change="fetchRecords"
            @current-change="fetchRecords"
          />
        </div>
      </el-tab-pane>

      <el-tab-pane v-if="showClusterTab" label="K8s 集群" name="k8s" lazy>
        <K8sClusterPanel ref="clusterPanelRef" />
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="detailVisible" title="执行详情" size="58%">
      <template v-if="detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ detail.record.name }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ statusLabel(detail.record.status) }}</el-descriptions-item>
          <el-descriptions-item label="来源">{{ sourceLabel(detail.record.source) }}</el-descriptions-item>
          <el-descriptions-item label="目标">{{ displayExecutionTarget(detail.record) }}</el-descriptions-item>
          <el-descriptions-item label="账号">{{ detail.record.trigger_user || detail.record.created_by || '-' }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ recordKindLabel(recordMeta(detail.record).record_kind) }}</el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ formatTime(detail.record.started_at) }}</el-descriptions-item>
          <el-descriptions-item label="结束时间">{{ formatTime(detail.record.finished_at) }}</el-descriptions-item>
          <el-descriptions-item label="回滚能力">{{ rollbackLabel(detail.record.rollback_capability) }}</el-descriptions-item>
          <el-descriptions-item label="退出码">{{ detail.record.exit_code ?? '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="detail.impacts?.length"
          type="warning"
          show-icon
          :closable="false"
          class="detail-block"
          title="该记录之后存在同目标/同资源的成功执行，回滚可能影响后续状态。"
        />

        <section class="detail-block">
          <h3>命令 / 脚本摘要</h3>
          <pre>{{ detail.record.command || '-' }}</pre>
        </section>
        <section v-if="recordMeta(detail.record).record_kind === 'ai_call'" class="detail-block detail-card">
          <h3>AI 调用</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="技能包">{{ packLabel(recordMeta(detail.record).skill_pack) }}</el-descriptions-item>
            <el-descriptions-item label="权益来源">{{ entitlementLabel(recordMeta(detail.record).entitlement_source) }}</el-descriptions-item>
            <el-descriptions-item label="消耗次数">{{ recordMeta(detail.record).quota_used ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="剩余额度">{{ quotaRemainingText(recordMeta(detail.record).quota_remaining) }}</el-descriptions-item>
            <el-descriptions-item label="认证">{{ authKindLabel(recordMeta(detail.record).auth_kind) }}</el-descriptions-item>
            <el-descriptions-item label="客户端">{{ recordMeta(detail.record).client?.version || '-' }}</el-descriptions-item>
            <el-descriptions-item label="诊断地址">{{ displayExecutionTarget(detail.record) }}</el-descriptions-item>
          </el-descriptions>
          <pre class="detail-pre--light">{{ pretty(recordMeta(detail.record).context) }}</pre>
        </section>
        <section v-if="recordMeta(detail.record).record_kind === 'cli_install'" class="detail-block detail-card">
          <h3>安装信息</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="绑定 ID">{{ recordMeta(detail.record).cli_binding_id || '-' }}</el-descriptions-item>
            <el-descriptions-item label="主机">{{ detail.record.target_host || '-' }}</el-descriptions-item>
            <el-descriptions-item label="系统">{{ recordMeta(detail.record).os || '-' }}</el-descriptions-item>
            <el-descriptions-item label="架构">{{ recordMeta(detail.record).arch || '-' }}</el-descriptions-item>
            <el-descriptions-item label="安装用户">{{ recordMeta(detail.record).install_user || '-' }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ recordMeta(detail.record).version || '-' }}</el-descriptions-item>
          </el-descriptions>
        </section>
        <section v-if="recordMeta(detail.record).record_kind === 'go_runtime'" class="detail-block detail-card">
          <h3>Go Runtime</h3>
          <el-descriptions :column="2" border>
            <el-descriptions-item label="结论">{{ runtimeSummary(detail.record).level || '-' }}</el-descriptions-item>
            <el-descriptions-item label="标题">{{ runtimeSummary(detail.record).title || '-' }}</el-descriptions-item>
            <el-descriptions-item label="目标">{{ runtimeTarget(detail.record).target || detail.record.resource_name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="Host PID">{{ runtimeTarget(detail.record).pid || '-' }}</el-descriptions-item>
            <el-descriptions-item label="节点">{{ runtimeTarget(detail.record).node || detail.record.target_host || '-' }}</el-descriptions-item>
            <el-descriptions-item label="容器">{{ runtimeTarget(detail.record).container || '-' }}</el-descriptions-item>
            <el-descriptions-item label="RSS">{{ bytesText(runtimeSummary(detail.record).rss_bytes) }}</el-descriptions-item>
            <el-descriptions-item label="FD">{{ runtimeSummary(detail.record).fd_open ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="线程">{{ runtimeSummary(detail.record).threads ?? '-' }}</el-descriptions-item>
            <el-descriptions-item label="观测会话">{{ recordMeta(detail.record).runtime_watch_session_id || '-' }}</el-descriptions-item>
          </el-descriptions>
          <pre class="detail-pre--light">{{ pretty(recordMeta(detail.record).summary) }}</pre>
        </section>
        <section class="detail-block">
          <h3>执行效果</h3>
          <pre>{{ pretty(detail.record.effects) }}</pre>
        </section>
        <section class="detail-block">
          <h3>回滚计划</h3>
          <pre>{{ pretty(detail.record.rollback_plan) || detail.record.rollback_advice || '-' }}</pre>
        </section>
        <section class="detail-block">
          <h3>事件</h3>
          <el-timeline>
            <el-timeline-item v-for="event in detail.events" :key="event.id" :timestamp="formatTime(event.created_at)">
              <strong>{{ event.phase }}</strong> · {{ event.message }}
              <pre v-if="event.output">{{ event.output }}</pre>
            </el-timeline-item>
          </el-timeline>
        </section>
      </template>
    </el-drawer>

    <el-dialog v-model="rollbackVisible" title="回滚确认" width="680px">
      <template v-if="rollbackPreview">
        <el-alert
          v-if="rollbackPreview.impacts?.length"
          type="warning"
          show-icon
          :closable="false"
          title="检测到关联影响"
          description="下面这些后续执行可能依赖当前状态。确认后系统会创建回滚记录，实际执行前请再次核对。"
        />
        <el-empty v-else description="未检测到同目标/同资源的后续成功执行" />
        <el-table v-if="rollbackPreview.impacts?.length" :data="rollbackPreview.impacts" border size="small" class="impact-table">
          <el-table-column prop="created_at" label="时间" width="165">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="name" label="后续执行" />
          <el-table-column prop="status" label="状态" width="90" />
        </el-table>
        <h3>回滚计划</h3>
        <pre>{{ pretty(rollbackPreview.rollbackPlan) || rollbackPreview.rollbackAdvice || '-' }}</pre>
      </template>
      <template #footer>
        <el-button @click="rollbackVisible = false">取消</el-button>
        <el-button type="warning" :loading="rollbackLoading" @click="confirmRollback">确认创建回滚记录</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshRight, Search } from '@element-plus/icons-vue'
import {
  getExecutionRecordDetail,
  getExecutionRecords,
  previewExecutionRollback,
  rollbackExecutionRecord,
} from '../../api/execution-records'
import { getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'
import K8sClusterPanel from '../../components/k8s/K8sClusterPanel.vue'
import { displayExecutionTarget } from '../../utils/executionRecordDisplay'

const route = useRoute()
const router = useRouter()

const isAdminShell = computed(() => route.path.startsWith('/admin'))
const capabilityByFeature = ref<Record<string, BillingCapabilityFeature>>({})

const showClusterTab = computed(() => {
  if (!isAdminShell.value) return false
  const row = capabilityByFeature.value['feature.k8s_delivery']
  return row ? row.visible_enabled && row.can_view !== false : true
})

const activePanel = ref<'records' | 'k8s'>('records')
const clusterPanelRef = ref<InstanceType<typeof K8sClusterPanel> | null>(null)

function syncTabFromRoute() {
  if (route.query.tab === 'k8s' && showClusterTab.value) {
    activePanel.value = 'k8s'
  } else {
    activePanel.value = 'records'
  }
}

async function loadBillingCapabilities() {
  try {
    const data = await getBillingCapabilities()
    const next: Record<string, BillingCapabilityFeature> = {}
    ;(data.features || []).forEach((item) => {
      next[item.feature_key] = item
    })
    capabilityByFeature.value = next
  } catch {
    capabilityByFeature.value = {}
  }
}

function onTabChange(name: string | number) {
  const n = String(name)
  if (n === 'k8s') {
    router.replace({ path: route.path, query: { ...route.query, tab: 'k8s' } })
    void nextTick(() => clusterPanelRef.value?.loadClusters())
    return
  }
  const { tab: _t, ...rest } = route.query
  router.replace({ path: route.path, query: Object.keys(rest).length ? rest : undefined })
}

watch(
  () => [route.query.tab, showClusterTab.value] as const,
  () => {
    syncTabFromRoute()
  }
)

const loading = ref(false)
const records = ref<any[]>([])
const total = ref(0)
const detailVisible = ref(false)
const detail = ref<any>(null)
const rollbackVisible = ref(false)
const rollbackPreview = ref<any>(null)
const rollbackTarget = ref<any>(null)
const rollbackLoading = ref(false)

const filters = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  target: '',
  source: '',
  status: '',
  category: '',
  rollbackCapability: '',
})

onMounted(async () => {
  await loadBillingCapabilities()
  syncTabFromRoute()
  void fetchRecords()
})

async function fetchRecords() {
  loading.value = true
  try {
    const res = await getExecutionRecords(filters)
    records.value = res.list || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  filters.page = 1
  fetchRecords()
}

function handleReset() {
  Object.assign(filters, { page: 1, pageSize: 20, keyword: '', target: '', source: '', status: '', category: '', rollbackCapability: '' })
  fetchRecords()
}

async function openDetail(row: any) {
  detail.value = await getExecutionRecordDetail(row.id)
  detailVisible.value = true
}

async function previewRollback(row: any) {
  rollbackTarget.value = row
  rollbackPreview.value = await previewExecutionRollback(row.id)
  rollbackVisible.value = true
}

async function confirmRollback() {
  if (!rollbackTarget.value) return
  rollbackLoading.value = true
  try {
    await rollbackExecutionRecord(rollbackTarget.value.id, true)
    ElMessage.success('已创建回滚记录')
    rollbackVisible.value = false
    fetchRecords()
  } finally {
    rollbackLoading.value = false
  }
}

function formatTime(value?: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function pretty(value: any) {
  if (!value) return ''
  if (typeof value === 'string') return value
  return JSON.stringify(value, null, 2)
}

function recordMeta(row: any) {
  return row?.metadata && typeof row.metadata === 'object' ? row.metadata : {}
}

function sourceLabel(value: string) {
  return ({ ai: 'AI', install: '安装', cli: 'CLI', script: '脚本', 'init-tools': '初始化', job: '作业', k8s: 'K8s', rollback: '回滚' } as Record<string, string>)[value] || value || '-'
}

function recordKindLabel(value: string) {
  return ({ ai_call: 'AI 调用', cli_install: '安装 ai-sre', go_runtime: 'Go Runtime', script: '脚本', task: '任务' } as Record<string, string>)[value] || value || '-'
}

function packLabel(value: string) {
  return ({
    'pack.k8s_delivery': 'K8s 交付',
    'pack.node_ops': '节点运维',
    'pack.monitoring': '可观测性',
    'pack.backup_performance': '备份与性能',
    'pack.runtime_observe': '运行时诊断',
    'skillpack.k8s': 'K8s 技能包',
    'skillpack.kafka': 'Kafka 技能包',
    'skillpack.redis': 'Redis 技能包',
    'skillpack.nginx': 'Nginx 技能包',
    'skillpack.mysql': 'MySQL 技能包',
    'skillpack.elasticsearch': 'Elasticsearch 技能包',
  } as Record<string, string>)[value] || value || '-'
}

function runtimeSummary(row: any) {
  const meta = recordMeta(row)
  return meta.summary && typeof meta.summary === 'object' ? meta.summary : {}
}

function runtimeTarget(row: any) {
  const meta = recordMeta(row)
  return meta.target && typeof meta.target === 'object' ? meta.target : {}
}

function bytesText(value: any) {
  const n = Number(value)
  if (!Number.isFinite(n) || n <= 0) return '-'
  if (n >= 1024 * 1024 * 1024) return `${(n / 1024 / 1024 / 1024).toFixed(1)} GiB`
  if (n >= 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MiB`
  if (n >= 1024) return `${(n / 1024).toFixed(1)} KiB`
  return `${n} B`
}

function entitlementLabel(value: string) {
  return ({ free: '免费额度', entitlement: '权益', manual: '人工授权', stripe: '订阅', super_admin: '超级管理员' } as Record<string, string>)[value] || value || '-'
}

function authKindLabel(value: string) {
  return ({ jwt: '控制台登录', cli: 'CLI 绑定', anonymous: '匿名' } as Record<string, string>)[value] || value || '-'
}

function quotaRemainingText(value: any) {
  if (value === -1) return '不限'
  return value ?? '-'
}

function statusLabel(value: string) {
  return ({ pending: '等待中', running: '执行中', success: '成功', failed: '失败', cancelled: '已取消' } as Record<string, string>)[value] || value || '-'
}

function statusType(value: string) {
  return ({ success: 'success', failed: 'danger', running: 'warning', cancelled: 'info' } as Record<string, any>)[value] || 'info'
}

function rollbackLabel(value: string) {
  return ({ auto: '可回滚', manual: '人工建议', none: '不可回滚' } as Record<string, string>)[value] || value || '-'
}

function rollbackType(value: string) {
  return ({ auto: 'success', manual: 'warning', none: 'info' } as Record<string, any>)[value] || 'info'
}

function rollbackStatusLabel(value: string) {
  return ({ not_started: '未回滚', pending: '待执行', success: '成功', failed: '失败', blocked: '有依赖' } as Record<string, string>)[value] || value || '-'
}
</script>

<style scoped>
.execution-records {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  min-height: 0;
}

.exec-tabs {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.exec-tabs :deep(.el-tabs__header) {
  margin: 0;
}

.exec-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 12px;
}

.exec-tabs :deep(.el-tab-pane) {
  min-height: 0;
}

.page-header {
  flex-shrink: 0;
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}

.page-header-copy {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.page-title {
  margin: 0;
  font-size: var(--page-header-title-max, 17px);
  font-weight: 600;
  color: var(--apple-ink, #111827);
}

.page-header-inner {
  display: none;
}

.page-kicker {
  display: none;
}

.page-desc {
  display: none;
}

.search-filters {
  display: grid;
  grid-template-columns: minmax(220px, 1.3fr) minmax(180px, 1fr) 120px 120px 130px 140px auto auto;
  gap: 12px;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgb(0 0 0 / 8%);
}

.records-table,
.pagination {
  background: #fff;
}

.pagination {
  display: flex;
  justify-content: flex-end;
  padding: 12px;
  border-radius: 8px;
}

.record-name {
  font-weight: 600;
  color: #1f2937;
}

.record-command {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.record-user {
  font-weight: 500;
  color: #1f2937;
}

.record-pack {
  margin-top: 4px;
  color: #64748b;
  font-size: 12px;
}

.detail-block {
  margin-top: 18px;
}

.detail-card {
  background: #f5f5f7;
  padding: 14px;
}

.detail-block h3,
.execution-records :deep(.el-dialog h3) {
  margin: 0 0 8px;
  font-size: 14px;
  color: #334155;
}

pre {
  margin: 0;
  padding: 12px;
  max-height: 260px;
  overflow: auto;
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 6px;
  white-space: pre-wrap;
  word-break: break-word;
}

.detail-pre--light {
  margin-top: 12px;
  background: #fff;
  color: #1f2937;
}

.impact-table {
  margin: 14px 0;
}

@media (max-width: 1180px) {
  .search-filters {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
