<template>
  <div class="runtime-observe">
    <div class="page-header">
      <h2>进程观测</h2>
    </div>
    <p v-if="billingAlertTitle" class="billing-strip page-desc--muted">{{ billingAlertTitle }}</p>
    <p class="page-desc--muted">
      在 Pod 所在节点运行 <code>ai-sre diagnose go-process</code>，将多采样结果上报到本平台。用于持续观察 RSS、FD、CPU
      时间等信号（非侵入式，仅读 procfs/cgroup）。
    </p>

    <div class="toolbar">
      <el-button type="primary" :disabled="!canUse" @click="openCreate">新建观测会话</el-button>
      <el-button :disabled="!canUse" @click="loadSessions">刷新列表</el-button>
    </div>

    <el-table v-loading="loading" :data="sessions" stripe style="width: 100%; margin-top: 12px">
      <el-table-column prop="namespace" label="命名空间" width="120" />
      <el-table-column prop="pod" label="Pod" min-width="160" />
      <el-table-column prop="container" label="容器" width="120" />
      <el-table-column prop="interval_sec" label="间隔(s)" width="100" />
      <el-table-column prop="status" label="状态" width="100" />
      <el-table-column prop="created_at" label="创建时间" width="180" />
      <el-table-column label="操作" width="220" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="selectSession(row)">样本</el-button>
          <el-button v-if="row.status === 'active'" link type="warning" @click="stopSession(row.id)">停止</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="drawerVisible" title="样本时间线" size="60%">
      <template v-if="selectedId">
        <p class="page-desc--muted">自动刷新（约 5s）。原始 JSON 见表格「摘要」列。</p>
        <el-table :data="samples" max-height="520" stripe>
          <el-table-column prop="observed_at" label="时间" width="200" />
          <el-table-column label="摘要" min-width="240">
            <template #default="{ row }">
              {{ sampleSummary(row.payload) }}
            </template>
          </el-table-column>
        </el-table>
      </template>
    </el-drawer>

    <el-dialog v-model="createVisible" title="新建观测会话" width="520px" @closed="resetCreate">
      <el-form :model="createForm" label-width="100px">
        <el-form-item label="命名空间" required>
          <el-input v-model="createForm.namespace" placeholder="default" />
        </el-form-item>
        <el-form-item label="Pod" required>
          <el-input v-model="createForm.pod" placeholder="例如 my-app-7d4f8" />
        </el-form-item>
        <el-form-item label="容器">
          <el-input v-model="createForm.container" placeholder="可选，默认第一个容器" />
        </el-form-item>
        <el-form-item label="间隔(秒)">
          <el-input-number v-model="createForm.interval_sec" :min="5" :max="3600" />
        </el-form-item>
        <el-form-item label="机器备注">
          <el-input v-model="createForm.machine_note" type="textarea" rows="2" placeholder="可选：填写执行 ai-sre 的节点" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createLoading" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="tokenVisible" title="写入令牌（仅显示一次）" width="640px">
      <p>请立即复制；关闭后需重新创建会话才能获取新令牌。</p>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="会话 ID">{{ created?.id }}</el-descriptions-item>
        <el-descriptions-item label="上报 URL">{{ uploadURLHint }}</el-descriptions-item>
        <el-descriptions-item label="令牌">{{ created?.sample_write_token }}</el-descriptions-item>
      </el-descriptions>
      <p style="margin-top: 12px"><strong>示例命令</strong>（在 Pod 所在节点执行）：</p>
      <el-input type="textarea" :rows="5" readonly :model-value="cliExample" />
      <template #footer>
        <el-button type="primary" @click="copyCli">复制命令</el-button>
        <el-button @click="tokenVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  listRuntimeWatchSessions,
  createRuntimeWatchSession,
  getRuntimeWatchSamples,
  stopRuntimeWatchSession,
  type RuntimeWatchSessionRow,
  type RuntimeWatchSampleRow,
  type CreateRuntimeWatchSessionResult
} from '../../api/runtimeWatch'
import { createCheckoutSession, getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'
import { copyTextToClipboard } from '../../utils/clipboard'

const loading = ref(false)
const sessions = ref<RuntimeWatchSessionRow[]>([])
const capability = ref<BillingCapabilityFeature | null>(null)
const observePackKey = computed(() => capability.value?.pack_key || 'pack.runtime_observe')
const canUse = computed(() => capability.value?.can_execute ?? false)
const billingAlertTitle = computed(() => {
  if (!capability.value) return '能力信息载入中…'
  if (canUse.value) return '已开通 · 可创建会话并接收节点上报的样本'
  const st = capability.value.execute_state as Record<string, unknown> | undefined
  return String(st?.msg || `需订阅 ${observePackKey.value} 后使用进程观测`)
})

const createVisible = ref(false)
const createLoading = ref(false)
const createForm = reactive({
  namespace: 'default',
  pod: '',
  container: '',
  interval_sec: 15,
  machine_note: ''
})

const tokenVisible = ref(false)
const created = ref<CreateRuntimeWatchSessionResult | null>(null)

const drawerVisible = ref(false)
const selectedId = ref('')
const samples = ref<RuntimeWatchSampleRow[]>([])
let pollTimer: ReturnType<typeof setInterval> | null = null

const apiBase = import.meta.env.VITE_BASE_API || '/ft-api'
const uploadURLHint = computed(() => {
  const b = String(apiBase).replace(/\/$/, '')
  if (b.startsWith('http')) return `${b}/api/runtime-watch/sample`
  const origin = typeof window !== 'undefined' ? window.location.origin : ''
  const prefix = b.startsWith('/') ? b : `/${b}`
  return `${origin}${prefix}/api/runtime-watch/sample`
})

const cliExample = computed(() => {
  const c = created.value
  if (!c) return ''
  return [
    `ai-sre diagnose go-process \\`,
    `  --namespace ${c.namespace} --pod ${c.pod}${c.container ? ` --container ${c.container}` : ''} \\`,
    `  --watch-samples 12 --watch-interval ${c.interval_sec}s \\`,
    `  --upload-url "${uploadURLHint.value}" \\`,
    `  --session-id ${c.id} \\`,
    `  --sample-token ${c.sample_write_token}`
  ].join('\n')
})

const loadCapability = async () => {
  try {
    const data = await getBillingCapabilities()
    capability.value =
      (data.features || []).find((item) => item.feature_key === 'feature.runtime_observe') || null
  } catch {
    capability.value = null
  }
}

const loadSessions = async () => {
  if (!canUse.value) {
    sessions.value = []
    return
  }
  loading.value = true
  try {
    sessions.value = await listRuntimeWatchSessions()
  } catch {
    sessions.value = []
  } finally {
    loading.value = false
  }
}

const openCreate = () => {
  if (!canUse.value) {
    void goSubscribe()
    return
  }
  createVisible.value = true
}

const resetCreate = () => {
  createForm.namespace = 'default'
  createForm.pod = ''
  createForm.container = ''
  createForm.interval_sec = 15
  createForm.machine_note = ''
}

const submitCreate = async () => {
  if (!createForm.namespace.trim() || !createForm.pod.trim()) {
    ElMessage.warning('请填写命名空间与 Pod')
    return
  }
  createLoading.value = true
  try {
    const res = await createRuntimeWatchSession({
      namespace: createForm.namespace.trim(),
      pod: createForm.pod.trim(),
      container: createForm.container.trim() || undefined,
      interval_sec: createForm.interval_sec,
      machine_note: createForm.machine_note.trim() || undefined
    })
    created.value = res
    createVisible.value = false
    tokenVisible.value = true
    await loadSessions()
  } finally {
    createLoading.value = false
  }
}

const goSubscribe = async () => {
  try {
    const resp = await createCheckoutSession({ pack_key: observePackKey.value })
    const url = (resp as { url?: string })?.url
    if (url) window.location.href = url
  } catch {
    /* interceptor */
  }
}

const copyCli = async () => {
  try {
    await copyTextToClipboard(cliExample.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const selectSession = async (row: RuntimeWatchSessionRow) => {
  selectedId.value = row.id
  drawerVisible.value = true
  await refreshSamples()
}

const refreshSamples = async () => {
  if (!selectedId.value || !canUse.value) return
  try {
    const data = await getRuntimeWatchSamples(selectedId.value)
    samples.value = data.samples || []
  } catch {
    samples.value = []
  }
}

const stopSession = async (id: string) => {
  try {
    await stopRuntimeWatchSession(id)
    ElMessage.success('已停止')
    await loadSessions()
    if (selectedId.value === id) drawerVisible.value = false
  } catch {
    /* */
  }
}

function sampleSummary(payload: unknown): string {
  if (!payload || typeof payload !== 'object') return ''
  const p = payload as Record<string, unknown>
  const samples = p.samples as unknown[] | undefined
  const last = samples?.length ? (samples[samples.length - 1] as Record<string, unknown>) : null
  const snap = last?.snapshot as Record<string, unknown> | undefined
  const st = snap?.status as Record<string, unknown> | undefined
  const rss = st?.vm_rss_bytes
  const fd = (snap?.fd as Record<string, unknown> | undefined)?.open
  const tf = p.trend_findings as unknown[] | undefined
  return `samples=${Array.isArray(samples) ? samples.length : 0} rss_bytes=${rss ?? '-'} fd=${fd ?? '-'} trend=${tf?.length ?? 0}`
}

onMounted(async () => {
  await loadCapability()
  await loadSessions()
})

watch(drawerVisible, (v) => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  if (v && selectedId.value) {
    pollTimer = setInterval(() => {
      void refreshSamples()
    }, 5000)
  }
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<style scoped>
.runtime-observe {
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
}
.toolbar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
code {
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
}
</style>
