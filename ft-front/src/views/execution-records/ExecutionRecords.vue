<template>
  <div class="execution-records page-shell">
    <header class="page-header">
      <div class="page-header-inner">
        <span class="page-kicker">Execution History</span>
        <h2 class="page-title">执行记录</h2>
        <p class="page-desc">统一查看 ai-sre、复制脚本和作业任务的执行结果、影响摘要与回滚状态。</p>
      </div>
    </header>

    <section class="search-filters">
      <el-input v-model="filters.keyword" placeholder="搜索命令 / 输出 / 名称" clearable :prefix-icon="Search" @keyup.enter="handleSearch" />
      <el-input v-model="filters.target" placeholder="目标主机 / 资源" clearable :prefix-icon="Search" @keyup.enter="handleSearch" />
      <el-select v-model="filters.source" placeholder="来源" clearable @change="handleSearch">
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
      <el-table-column prop="target_host" label="目标" min-width="150">
        <template #default="{ row }">{{ row.target_host || row.resource_name || row.resource_id || '-' }}</template>
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

    <el-drawer v-model="detailVisible" title="执行详情" size="58%">
      <template v-if="detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="名称">{{ detail.record.name }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ statusLabel(detail.record.status) }}</el-descriptions-item>
          <el-descriptions-item label="来源">{{ sourceLabel(detail.record.source) }}</el-descriptions-item>
          <el-descriptions-item label="目标">{{ detail.record.target_host || detail.record.resource_name || '-' }}</el-descriptions-item>
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
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { RefreshRight, Search } from '@element-plus/icons-vue'
import {
  getExecutionRecordDetail,
  getExecutionRecords,
  previewExecutionRollback,
  rollbackExecutionRecord,
} from '../../api/execution-records'

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
  rollbackCapability: '',
})

onMounted(fetchRecords)

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
  Object.assign(filters, { page: 1, pageSize: 20, keyword: '', target: '', source: '', status: '', rollbackCapability: '' })
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

function sourceLabel(value: string) {
  return ({ cli: 'CLI', script: '脚本', 'init-tools': '初始化', job: '作业', k8s: 'K8s', rollback: '回滚' } as Record<string, string>)[value] || value || '-'
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
  gap: 16px;
}

.search-filters {
  display: grid;
  grid-template-columns: minmax(220px, 1.3fr) minmax(180px, 1fr) 130px 130px 150px auto auto;
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

.detail-block {
  margin-top: 18px;
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

.impact-table {
  margin: 14px 0;
}

@media (max-width: 1180px) {
  .search-filters {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
