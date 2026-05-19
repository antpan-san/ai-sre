<template>
  <div class="runtime-diagnose page-shell">
    <div class="page-header">
      <h2>运行时诊断</h2>
      <p class="page-desc--muted">
        每次执行 <code>ai-sre check go</code> 产生<strong>一条独立报告</strong>（根因 + 证据），按时间倒序排列；不会对同一目标做历史对比或合并。
      </p>
    </div>

    <p v-if="billingAlertTitle" class="billing-strip">{{ billingAlertTitle }}</p>

    <div class="toolbar">
      <el-input
        v-model="keyword"
        clearable
        placeholder="搜索目标、根因、证据…"
        class="toolbar-search"
        :disabled="!canUse"
      />
      <el-select v-model="levelFilter" clearable placeholder="级别" style="width: 120px" :disabled="!canUse">
        <el-option label="严重" value="CRITICAL" />
        <el-option label="警告" value="WARN" />
        <el-option label="正常" value="OK" />
        <el-option label="未知" value="UNKNOWN" />
      </el-select>
      <el-button :disabled="!canUse" @click="loadReports">刷新</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="filteredReports"
      stripe
      class="report-table"
      empty-text="暂无诊断报告。在本机执行一次 diagnose 后刷新本页。"
      @row-click="openDetail"
    >
      <el-table-column label="诊断时间" width="172">
        <template #default="{ row }">
          {{ formatTime(row.last_diagnosed_at || row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="级别" width="88" align="center">
        <template #default="{ row }">
          <el-tag :type="levelTagType(row.diagnosis_level)" size="small" effect="plain">
            {{ levelLabel(row.diagnosis_level) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="诊断目标" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="target-text">{{ displayTarget(row) }}</span>
          <span v-if="row.work_pod && row.resource_kind" class="work-pod-hint"> → {{ row.work_pod }}</span>
        </template>
      </el-table-column>
      <el-table-column label="根因" min-width="280">
        <template #default="{ row }">
          <div class="root-cause-cell">
            <SafeMarkdown :content="row.root_cause" :clamp-lines="3" />
          </div>
        </template>
      </el-table-column>
      <el-table-column label="来源" width="72" align="center">
        <template #default="{ row }">
          <span v-if="row.diagnosis_source === 'ai'" class="source-ai">AI</span>
          <span v-else-if="row.diagnosis_source === 'local'" class="source-local">本地</span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="" width="148" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click.stop="openDetail(row)">查看</el-button>
          <el-button link type="danger" :disabled="deletingId === row.id" @click.stop="confirmDelete(row)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="drawerVisible" title="单次诊断报告" size="560px" destroy-on-close>
      <template v-if="active">
        <el-card shadow="never" class="detail-card">
          <template #header>
            <div class="detail-header">
              <el-tag :type="levelTagType(active.diagnosis_level)" effect="dark" size="small">
                {{ levelLabel(active.diagnosis_level) }}
              </el-tag>
              <span class="detail-target">{{ displayTarget(active) }}</span>
            </div>
          </template>
          <div class="detail-section">
            <div class="detail-label">根因</div>
            <div class="detail-body detail-md">
              <SafeMarkdown :content="active.root_cause" />
            </div>
          </div>
          <div v-if="active.evidence" class="detail-section">
            <div class="detail-label">证据</div>
            <div class="detail-evidence-md">
              <SafeMarkdown :content="active.evidence" />
            </div>
          </div>
          <el-descriptions :column="1" border size="small" class="meta-desc">
            <el-descriptions-item label="诊断时间">
              {{ formatTime(active.last_diagnosed_at || active.created_at) }}
            </el-descriptions-item>
            <el-descriptions-item v-if="active.diagnosis_source" label="结论来源">
              {{ active.diagnosis_source === 'ai' ? '平台 AI 分析' : '本地规则/指标' }}
            </el-descriptions-item>
            <el-descriptions-item v-if="active.sample_count != null && active.sample_count > 0" label="本次 proc 采样">
              {{ active.sample_count }} 次
            </el-descriptions-item>
            <el-descriptions-item v-if="active.work_pod" label="分析 Pod">
              {{ active.namespace }}/{{ active.work_pod }}
            </el-descriptions-item>
            <el-descriptions-item label="报告 ID">
              <span class="report-id">{{ active.id }}</span>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <div v-if="findings.length" class="detail-section" style="margin-top: 16px">
          <div class="detail-label">辅助发现（当次采集）</div>
          <ul class="finding-list">
            <li v-for="(f, i) in findings" :key="i">
              <el-tag :type="findingTag(f.severity)" size="small" effect="plain">{{ f.severity }}</el-tag>
              {{ f.title }}
            </li>
          </ul>
        </div>

        <el-collapse v-if="detailPayload" style="margin-top: 16px">
          <el-collapse-item title="技术数据（可选，供研发复核）" name="raw">
            <pre class="raw-json">{{ prettyJSON(detailPayload) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listRuntimeDiagnoses, getRuntimeWatchSamples, deleteRuntimeWatchSession, type RuntimeDiagnosisRow } from '../../api/runtimeWatch'
import { getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'
import SafeMarkdown from '../../components/markdown/SafeMarkdown.vue'

const loading = ref(false)
const reports = ref<RuntimeDiagnosisRow[]>([])
const keyword = ref('')
const levelFilter = ref('')
const capability = ref<BillingCapabilityFeature | null>(null)
const observePackKey = computed(() => capability.value?.pack_key || 'pack.runtime_observe')
const canUse = computed(() => capability.value?.can_execute ?? false)
const billingAlertTitle = computed(() => {
  if (!capability.value) return '能力信息载入中…'
  if (canUse.value) return '已开通 · 每次 diagnose 生成一条报告，仅本账号可见'
  const st = capability.value.execute_state as Record<string, unknown> | undefined
  return String(st?.msg || `需订阅 ${observePackKey.value} 后查看运行时诊断`)
})

const drawerVisible = ref(false)
const active = ref<RuntimeDiagnosisRow | null>(null)
const deletingId = ref<string | null>(null)
const detailPayload = ref<unknown>(null)
const findings = ref<{ severity: string; title: string }[]>([])

const filteredReports = computed(() => {
  let list = [...reports.value]
  const kw = keyword.value.trim().toLowerCase()
  if (kw) {
    list = list.filter((r) => {
      const hay = [
        displayTarget(r),
        r.root_cause,
        r.evidence,
        r.work_pod,
        r.resource_kind,
        r.resource_name
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()
      return hay.includes(kw)
    })
  }
  if (levelFilter.value) {
    list = list.filter((r) => (r.diagnosis_level || 'UNKNOWN').toUpperCase() === levelFilter.value)
  }
  return list
})

function displayTarget(row: RuntimeDiagnosisRow): string {
  if (row.target_display?.trim()) return row.target_display.trim()
  if (row.resource_kind && row.resource_name) {
    return `${row.resource_kind}/${row.namespace}/${row.resource_name}`
  }
  return `${row.namespace}/${row.pod}`
}

function formatTime(iso?: string): string {
  if (!iso) return '—'
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

function levelLabel(level?: string): string {
  switch ((level || '').toUpperCase()) {
    case 'CRITICAL':
      return '严重'
    case 'WARN':
      return '警告'
    case 'OK':
      return '正常'
    default:
      return '未知'
  }
}

function levelTagType(level?: string): 'danger' | 'warning' | 'success' | 'info' {
  switch ((level || '').toUpperCase()) {
    case 'CRITICAL':
      return 'danger'
    case 'WARN':
      return 'warning'
    case 'OK':
      return 'success'
    default:
      return 'info'
  }
}

function findingTag(sev: string): 'danger' | 'warning' | 'info' {
  const s = sev.toLowerCase()
  if (s.includes('crit')) return 'danger'
  if (s.includes('warn')) return 'warning'
  return 'info'
}

function prettyJSON(v: unknown): string {
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

function extractFindings(payload: unknown): { severity: string; title: string }[] {
  if (!payload || typeof payload !== 'object') return []
  const p = payload as Record<string, unknown>
  const tf = p.trend_findings
  if (!Array.isArray(tf)) return []
  return tf
    .slice(0, 6)
    .map((item) => {
      const f = item as Record<string, unknown>
      return {
        severity: String(f.severity || 'info'),
        title: String(f.title || '')
      }
    })
    .filter((f) => f.title)
}

const loadCapability = async () => {
  try {
    const data = await getBillingCapabilities()
    capability.value =
      (data.features || []).find((item) => item.feature_key === 'feature.runtime_observe') || null
  } catch {
    capability.value = null
  }
}

const loadReports = async () => {
  if (!canUse.value) {
    reports.value = []
    return
  }
  loading.value = true
  try {
    reports.value = await listRuntimeDiagnoses()
  } catch {
    reports.value = []
    ElMessage.error('加载诊断报告失败')
  } finally {
    loading.value = false
  }
}

const openDetail = async (row: RuntimeDiagnosisRow) => {
  active.value = row
  drawerVisible.value = true
  detailPayload.value = null
  findings.value = []
  try {
    const data = await getRuntimeWatchSamples(row.id)
    const last = data.samples?.length ? data.samples[data.samples.length - 1] : null
    if (last?.payload) {
      detailPayload.value = last.payload
      findings.value = extractFindings(last.payload)
    }
  } catch {
    ElMessage.warning('未能加载报告技术数据')
  }
}

async function confirmDelete(row: RuntimeDiagnosisRow) {
  if (!canUse.value) return
  try {
    await ElMessageBox.confirm(
      `将永久删除该诊断报告（含附带的样本数据）。目标：${displayTarget(row)}`,
      '删除诊断报告',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  deletingId.value = row.id
  try {
    await deleteRuntimeWatchSession(row.id)
    ElMessage.success('已删除')
    if (active.value?.id === row.id) {
      drawerVisible.value = false
      active.value = null
    }
    await loadReports()
  } catch {
    ElMessage.error('删除失败')
  } finally {
    deletingId.value = null
  }
}

onMounted(async () => {
  await loadCapability()
  await loadReports()
})
</script>

<style scoped>
.runtime-diagnose {
  padding: 16px 20px 32px;
}
.page-header h2 {
  margin: 0 0 8px;
  font-size: 20px;
  font-weight: 600;
}
.page-desc--muted {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin: 0 0 12px;
  line-height: 1.5;
}
.billing-strip {
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  margin-bottom: 12px;
}
.toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
  margin-bottom: 12px;
}
.toolbar-search {
  width: 280px;
}
.report-table :deep(.el-table__row) {
  cursor: pointer;
}
.target-text {
  font-weight: 500;
}
.work-pod-hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.root-cause-cell {
  min-height: 2.5em;
}
.root-cause-cell :deep(.safe-md) {
  font-size: 13px;
  line-height: 1.45;
}
.muted {
  color: var(--el-text-color-placeholder);
}
.source-ai {
  color: var(--el-color-primary);
  font-size: 12px;
}
.source-local {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.detail-card :deep(.el-card__header) {
  padding: 12px 16px;
}
.detail-header {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.detail-target {
  font-weight: 600;
  font-size: 14px;
}
.detail-section {
  margin-bottom: 16px;
}
.detail-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.detail-md {
  padding: 10px 12px;
  background: var(--el-fill-color-blank);
  border-radius: 8px;
  border: 1px solid var(--el-border-color-lighter);
}
.detail-evidence-md {
  padding: 10px 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  max-height: 360px;
  overflow: auto;
  font-size: 13px;
}
.detail-evidence-md :deep(pre) {
  white-space: pre-wrap;
  word-break: break-word;
}
.detail-body {
  margin: 0;
  line-height: 1.6;
  font-size: 14px;
}
.root-cause-block {
  font-weight: 500;
}
.detail-evidence {
  margin: 0;
  padding: 12px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 280px;
  overflow: auto;
}
.meta-desc {
  margin-top: 12px;
}
.report-id {
  font-family: ui-monospace, monospace;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.finding-list {
  margin: 0;
  padding: 0;
  list-style: none;
}
.finding-list li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
  line-height: 1.45;
}
.raw-json {
  margin: 0;
  font-size: 11px;
  max-height: 360px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-word;
}
code {
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
}
</style>
