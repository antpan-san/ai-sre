<template>
  <div class="job-center page-shell page-shell--fill">
    <div class="page-header">
      <div class="page-header__titles">
        <h2>作业中心</h2>
        <p class="page-header__sub">对在线 Agent 执行 shell；CLI：<code>ai-sre job run</code>，<code>?jobId=</code> 可同步结果。</p>
      </div>
      <div class="page-header__actions">
        <el-tooltip content="刷新在线列表" placement="bottom-end">
          <button
            type="button"
            class="job-icon-btn"
            :disabled="machinesLoading"
            aria-label="刷新"
            @click="refreshMachines"
          >
            <el-icon class="job-icon-btn__icon" :class="{ 'job-icon-btn__icon--spin': machinesLoading }">
              <RefreshRight />
            </el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>

    <el-card class="job-card" shadow="never">
      <div class="job-card-inner">
        <section class="job-section job-section--targets">
          <div class="section-head">
            <h3>目标</h3>
            <span class="section-meta">{{ machines.length }} 在线 · {{ selectedMachines.length }} 已选</span>
          </div>
          <p class="field-hint">UUID / IP / 主机名，空格或换行分隔；执行前自动解析。</p>
          <div class="targets-body" v-loading="machinesLoading">
            <el-input
              v-model="machineTargetsRaw"
              type="textarea"
              :autosize="{ minRows: 1, maxRows: 2 }"
              placeholder="每行一台"
              class="targets-textarea"
            />
            <el-button size="small" @click="clearTargets">清空</el-button>
          </div>
        </section>

        <div class="opt-toolbar">
          <span class="opt-label">超时(s)</span>
          <el-select v-model="jobTimeoutSec" size="small" class="opt-timeout">
            <el-option label="60" :value="60" />
            <el-option label="120" :value="120" />
            <el-option label="300" :value="300" />
            <el-option label="600" :value="600" />
            <el-option label="1800" :value="1800" />
            <el-option label="3600" :value="3600" />
          </el-select>
          <el-switch v-model="confirmDangerPatterns" size="small" />
          <span class="opt-inline">危险命令确认</span>
          <el-switch v-model="blockIfUnresolvedTargets" size="small" />
          <span class="opt-inline">未识别拦截</span>
        </div>

        <el-divider class="job-divider" />

        <div class="job-split">
          <section class="job-section job-section--cmd">
            <div class="section-head">
              <h3>命令</h3>
            </div>
            <div class="cmd-shell">
              <span class="cmd-shell__prompt" aria-hidden="true">$</span>
              <el-input
                v-model="commandText"
                type="textarea"
                :autosize="{ minRows: 4, maxRows: 4 }"
                placeholder="shell 一行或多行"
                clearable
                class="cmd-shell__input"
              />
            </div>

            <div v-if="commandErrors.length" class="cmd-warn" role="alert">
              <div v-for="(error, index) in commandErrors" :key="index" class="cmd-warn__line">
                <el-icon class="cmd-warn__icon"><Warning /></el-icon>
                <span>{{ error }}</span>
              </div>
            </div>

            <div class="cmd-actions">
              <el-dropdown v-if="commandHistory.length" trigger="click" :max-height="200">
                <el-button size="small">
                  <el-icon><Clock /></el-icon>
                  历史
                  <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="(command, index) in commandHistory"
                      :key="index"
                      class="history-dd"
                      @click="selectFromHistory(command)"
                    >
                      <pre class="history-dd__pre">{{ command }}</pre>
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
              <el-button type="primary" size="small" :disabled="executeDisabled" :loading="executing" @click="executeCommands">
                <el-icon><CirclePlus /></el-icon>
                执行
              </el-button>
              <el-button size="small" @click="clearCommand">清空</el-button>
              <el-button size="small" @click="openScriptDialog" :disabled="!canGenerateScript">
                <el-icon><DocumentCopy /></el-icon>
                脚本
              </el-button>
            </div>
            <div v-if="lastJobId" class="job-id-bar">
              <code class="job-id-bar__code">{{ lastJobId }}</code>
              <el-button link type="primary" size="small" @click="copyText(lastJobLink)">复制链接</el-button>
            </div>
            <p class="job-hint">Ctrl+Enter 执行 · Ctrl+K 清空 · Ctrl+R 刷新 · Ctrl+L 清结果</p>
          </section>

          <section class="job-section job-section--out">
            <div class="section-head">
              <h3>结果</h3>
              <div class="section-head__tools">
                <el-select v-model="resultFilter" size="small" class="filter-select">
                  <el-option label="全部" value="all" />
                  <el-option label="成功" value="success" />
                  <el-option label="失败" value="failed" />
                </el-select>
                <el-button size="small" text type="primary" :disabled="!filteredResults.length" @click="copyAllResults">
                  复制
                </el-button>
                <el-button size="small" text @click="clearResult">清空</el-button>
              </div>
            </div>

            <div v-if="polling" class="poll-banner">
              <el-icon class="poll-banner__icon"><Loading /></el-icon>
              回传中…
            </div>

            <div class="out-panel">
              <div v-if="resultFilter !== 'all'" class="out-filter-tip">
                {{ resultFilter === 'success' ? '成功' : '失败' }} · {{ filteredResults.length }} 条
              </div>
              <div v-if="!filteredResults.length" class="out-empty">
                {{ executionResults.length ? '无匹配' : '执行或 ?jobId=' }}
              </div>

              <article
                v-for="(result, index) in filteredResults"
                :key="result.machineId + '-' + index + '-' + result.executionTime + '-' + (result.jobId || '')"
                class="out-card"
              >
                <header class="out-card__head">
                  <div class="out-card__title">
                    <el-tag v-if="result.sourceLabel" size="small" type="info">{{ result.sourceLabel }}</el-tag>
                    <span class="out-card__name">{{ result.machineName }}</span>
                    <span class="out-card__id">{{ result.machineIP || result.machineId }}</span>
                    <el-tag :type="tagType(result)" size="small">{{ statusLabel(result) }}</el-tag>
                    <el-tag v-if="result.exitCode != null" size="small" effect="plain" type="info">exit {{ result.exitCode }}</el-tag>
                  </div>
                  <div class="out-card__meta">
                    <el-button text type="primary" size="small" @click="copyOneResult(result)">复制</el-button>
                  </div>
                </header>
                <div v-if="result.stdout" class="out-block out-block--stdout">
                  <div class="out-block__label">stdout</div>
                  <pre>{{ result.stdout }}</pre>
                </div>
                <div v-if="result.stderr" class="out-block out-block--stderr">
                  <div class="out-block__label">stderr</div>
                  <pre>{{ result.stderr }}</pre>
                </div>
              </article>
            </div>
          </section>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="scriptDialogVisible" title="一键脚本" width="560px" destroy-on-close>
      <p class="script-dialog-help">控制机运行；勿提交含令牌内容。</p>
      <el-input v-model="generatedScriptText" type="textarea" readonly :autosize="{ minRows: 12, maxRows: 12 }" class="script-ta" />
      <template #footer>
        <el-button @click="scriptDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="copyGeneratedScript">复制</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  RefreshRight,
  CirclePlus,
  Clock,
  ArrowDown,
  Warning,
  Loading,
  DocumentCopy
} from '@element-plus/icons-vue'
import { executeCommand, getExecutionResult, getAvailableMachines } from '../../api/job'
import type { JobSubTaskResult } from '../../api/job'
import { copyTextToClipboard } from '../../utils/clipboard'
import type { Machine } from '../../types'

const route = useRoute()
const router = useRouter()

const COMMAND_HISTORY_KEY = 'jobCenterCommandHistory'
const MAX_HISTORY = 20
const POLL_MS = 900
const MAX_WAIT_MS = 180_000

const executing = ref(false)
const polling = ref(false)
const machinesLoading = ref(false)
const machines = ref<Machine[]>([])

const machineTargetsRaw = ref('')
const selectedMachines = ref<Machine[]>([])
const unresolvedTokens = ref<string[]>([])

const jobTimeoutSec = ref(120)
const confirmDangerPatterns = ref(true)
const blockIfUnresolvedTargets = ref(true)

const lastJobId = ref('')

const commandText = ref('')
const commandErrors = ref<string[]>([])
const commandHistory = ref<string[]>([])

interface ExecutionResult {
  machineId: string
  machineName: string
  machineIP: string
  success: boolean
  stdout: string
  stderr: string
  executionTime: string
  status: string
  exitCode?: number | null
  jobId?: string
  sourceLabel?: string
}

const executionResults = ref<ExecutionResult[]>([])
const resultFilter = ref('all')

const filteredResults = computed(() => {
  if (resultFilter.value === 'all') return executionResults.value
  if (resultFilter.value === 'success') return executionResults.value.filter((r) => r.success)
  return executionResults.value.filter((r) => !r.success)
})

const shellJobBasePath = computed(() => (route.path.startsWith('/app') ? '/app' : '/admin'))

const lastJobLink = computed(() => {
  if (!lastJobId.value) return ''
  return `${window.location.origin}${shellJobBasePath.value}/job/center?jobId=${encodeURIComponent(lastJobId.value)}`
})

const hasBlockingUnresolved = computed(() => blockIfUnresolvedTargets.value && unresolvedTokens.value.length > 0)

const executeDisabled = computed(
  () =>
    selectedMachines.value.length === 0 ||
    !commandText.value.trim() ||
    executing.value ||
    hasBlockingUnresolved.value
)

const canGenerateScript = computed(
  () => selectedMachines.value.length > 0 && commandText.value.trim().length > 0
)

const scriptDialogVisible = ref(false)
const generatedScriptText = ref('')

function tokenizeTargets(s: string): string[] {
  return s
    .split(/[\s,;，；]+/)
    .map((t) => t.trim())
    .filter(Boolean)
}

function findMachineByToken(t: string): Machine | undefined {
  const tl = t.toLowerCase()
  return machines.value.find(
    (m) => m.id === t || m.id.toLowerCase() === tl || m.ip === t || (m.name && m.name.toLowerCase() === tl)
  )
}

function reconcileTargetsFromTextarea(notify: boolean) {
  const tokens = tokenizeTargets(machineTargetsRaw.value)
  unresolvedTokens.value = []
  const ids: string[] = []
  const seen = new Set<string>()
  for (const t of tokens) {
    const m = findMachineByToken(t)
    if (m && !seen.has(m.id)) {
      seen.add(m.id)
      ids.push(m.id)
    } else if (!m) {
      unresolvedTokens.value.push(t)
    }
  }
  selectedMachines.value = machines.value.filter((m) => ids.includes(m.id))
  if (!notify) return
  if (unresolvedTokens.value.length) {
    ElMessage.warning(`未识别的目标（请核对在线 UUID / IP / 名称）：${unresolvedTokens.value.join(', ')}`)
  } else if (tokens.length) {
    ElMessage.success(`已选 ${ids.length} 台机器`)
  }
}

let targetReconcileTimer: ReturnType<typeof setTimeout> | null = null

function scheduleTargetReconcile() {
  if (targetReconcileTimer) clearTimeout(targetReconcileTimer)
  targetReconcileTimer = setTimeout(() => {
    targetReconcileTimer = null
    reconcileTargetsFromTextarea(false)
  }, 380)
}

function flushTargetReconcile() {
  if (targetReconcileTimer) {
    clearTimeout(targetReconcileTimer)
    targetReconcileTimer = null
  }
  reconcileTargetsFromTextarea(false)
}

function syncTargetsTextFromSelection() {
  if (!selectedMachines.value.length) {
    machineTargetsRaw.value = ''
    return
  }
  machineTargetsRaw.value = selectedMachines.value.map((m) => `${m.ip}  ${m.name}  ${m.id}`).join('\n')
  unresolvedTokens.value = []
}

function clearTargets() {
  if (targetReconcileTimer) {
    clearTimeout(targetReconcileTimer)
    targetReconcileTimer = null
  }
  selectedMachines.value = []
  machineTargetsRaw.value = ''
  unresolvedTokens.value = []
}

const checkCommandSyntax = (command: string) => {
  const errors: string[] = []
  const lines = command.trim().split('\n')
  const dangerous = ['rm -rf', 'format', 'mkfs', 'dd if=/dev/zero']
  const interactive = ['vi', 'vim', 'nano', 'emacs', 'top', 'htop']

  lines.forEach((line, index) => {
    const t = line.trim()
    if (!t) return
    dangerous.forEach((d) => {
      if (t.includes(d)) errors.push(`第 ${index + 1} 行：含高风险片段「${d}」`)
    })
    const firstTok = t.split(/\s+/)[0] || ''
    interactive.forEach((c) => {
      if (firstTok === c || t.startsWith(c + ' ')) errors.push(`第 ${index + 1} 行：可能需要 TTY 的交互命令「${c}」`)
    })
  })
  commandErrors.value = errors
  return errors
}

watch(commandText, (v) => checkCommandSyntax(v))

watch(machineTargetsRaw, () => scheduleTargetReconcile())

const loadCommandHistory = () => {
  try {
    let raw = localStorage.getItem(COMMAND_HISTORY_KEY)
    if (!raw) {
      raw = localStorage.getItem('commandHistory')
      if (raw) localStorage.setItem(COMMAND_HISTORY_KEY, raw)
    }
    commandHistory.value = raw ? JSON.parse(raw) : []
  } catch {
    commandHistory.value = []
  }
}

const saveCommandHistory = () => {
  localStorage.setItem(COMMAND_HISTORY_KEY, JSON.stringify(commandHistory.value))
}

const addToHistory = (command: string) => {
  const c = command.trim()
  if (!c) return
  const i = commandHistory.value.indexOf(c)
  if (i > -1) commandHistory.value.splice(i, 1)
  commandHistory.value.unshift(c)
  if (commandHistory.value.length > MAX_HISTORY) commandHistory.value.pop()
  saveCommandHistory()
}

const selectFromHistory = (command: string) => {
  commandText.value = command
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms))

async function waitForJobResults(jobId: string): Promise<JobSubTaskResult[]> {
  const start = Date.now()
  let latest: JobSubTaskResult[] = []
  const isTerminal = (s: string) =>
    s === 'success' || s === 'failed' || s === 'cancelled' || s === 'timeout'

  while (Date.now() - start < MAX_WAIT_MS) {
    const data = await getExecutionResult(jobId)
    latest = data.results ?? []
    if (latest.length === 0) {
      await sleep(320)
      continue
    }
    if (latest.every((r) => isTerminal(r.status))) return latest
    await sleep(POLL_MS)
  }
  ElMessage.warning('等待结果超时（任务仍在跑时可刷新或去「执行记录」查看）')
  return latest
}

function mapRows(
  rows: JobSubTaskResult[],
  finishedAt: string,
  jobId: string,
  sourceLabel: string
): ExecutionResult[] {
  return rows.map((r) => {
    const ok = r.status === 'success'
    const errParts = [r.error?.trim(), r.exit_code != null && !ok ? `exit_code=${r.exit_code}` : ''].filter(
      Boolean
    ) as string[]
    return {
      machineId: r.machine_id,
      machineName: r.machine_name?.trim() || `机器 ${r.machine_id}`,
      machineIP: r.machine_ip?.trim() || '',
      success: ok,
      stdout: (r.output || '').trimEnd(),
      stderr: errParts.join('\n').trim(),
      executionTime: finishedAt,
      status: r.status,
      exitCode: r.exit_code,
      jobId,
      sourceLabel
    }
  })
}

const formatDate = (iso: string) => {
  if (!iso) return ''
  try {
    return new Date(iso).toLocaleString('zh-CN', { hour12: false })
  } catch {
    return iso
  }
}

const statusLabel = (r: ExecutionResult) => {
  if (r.status === 'success') return '成功'
  if (r.status === 'failed') return '失败'
  if (r.status === 'cancelled') return '已取消'
  if (r.status === 'timeout') return '超时'
  return r.status || '进行中'
}

const tagType = (r: ExecutionResult) => {
  if (r.success) return 'success'
  if (r.status === 'cancelled') return 'info'
  if (r.status === 'pending' || r.status === 'running' || r.status === 'dispatched') return 'warning'
  return 'danger'
}

const copyText = async (s: string) => {
  if (!s) return
  try {
    await copyTextToClipboard(s)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const copyOneResult = async (r: ExecutionResult) => {
  const text = [`# ${r.machineName} (${r.machineIP || r.machineId})`, r.stdout && `--- stdout ---\n${r.stdout}`, r.stderr && `--- stderr ---\n${r.stderr}`]
    .filter(Boolean)
    .join('\n\n')
  await copyText(text)
}

const copyAllResults = async () => {
  if (!filteredResults.value.length) return
  const parts = filteredResults.value.map((r, i) => {
    const head = `## ${i + 1}. ${r.machineName} (${r.machineIP || r.machineId}) [${statusLabel(r)}]`
    const body = [r.stdout && `stdout:\n${r.stdout}`, r.stderr && `stderr:\n${r.stderr}`].filter(Boolean).join('\n\n')
    return `${head}\n${body}`
  })
  await copyText(parts.join('\n\n---\n\n'))
  ElMessage.success('已复制全部输出')
}

const loadMachineList = async () => {
  const prev = new Set(selectedMachines.value.map((m) => m.id))
  machinesLoading.value = true
  try {
    machines.value = await getAvailableMachines()
    const nextIds = machines.value.filter((m) => prev.has(m.id)).map((m) => m.id)
    selectedMachines.value = machines.value.filter((m) => nextIds.includes(m.id))
    syncTargetsTextFromSelection()
  } catch (e: any) {
    ElMessage.error('获取机器列表失败: ' + (e?.message || e?.msg || '未知错误'))
  } finally {
    machinesLoading.value = false
  }
}

const refreshMachines = () => void loadMachineList()

const executeCommands = async () => {
  flushTargetReconcile()
  if (hasBlockingUnresolved.value) {
    ElMessage.error('请修正未识别目标，或关闭「未识别拦截」')
    return
  }
  if (!selectedMachines.value.length) {
    ElMessage.warning('请至少选择一台机器')
    return
  }
  if (!commandText.value.trim()) {
    ElMessage.warning('请输入命令')
    return
  }

  const errors = checkCommandSyntax(commandText.value)
  if (errors.length && confirmDangerPatterns.value) {
    try {
      await ElMessageBox.confirm(`检测到 ${errors.length} 条风险提示，仍要执行？\n\n${errors.join('\n')}`, '确认执行', {
        confirmButtonText: '继续',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch {
      return
    }
  }

  addToHistory(commandText.value.trim())
  executing.value = true
  polling.value = true
  const finishedAt = new Date().toISOString()
  try {
    const { jobId } = await executeCommand({
      machine_ids: selectedMachines.value.map((m) => m.id),
      command: commandText.value.trim(),
      timeout: jobTimeoutSec.value
    })
    lastJobId.value = jobId
    const rows = await waitForJobResults(jobId)
    const mapped = mapRows(rows, finishedAt, jobId, '控制台')
    executionResults.value.unshift(...mapped)
    const ok = mapped.filter((x) => x.success).length
    const bad = mapped.length - ok
    ElMessage.success(`已完成：${ok} 成功${bad ? `，${bad} 台异常` : ''}`)
  } catch (e: any) {
    ElMessage.error('执行失败: ' + (e?.message || e?.msg || '未知错误'))
  } finally {
    polling.value = false
    executing.value = false
  }
}

async function importResultsForJob(jobId: string, sourceLabel: string) {
  const id = jobId.trim()
  if (!id) return
  try {
    const data = await getExecutionResult(id)
    const mapped = mapRows(data.results ?? [], new Date().toISOString(), id, sourceLabel)
    if (!mapped.length) {
      ElMessage.info('尚未有子任务结果，请稍后刷新本页或通过 ?jobId= 再次打开')
      return
    }
    executionResults.value.unshift(...mapped)
    lastJobId.value = id
    ElMessage.success(`已载入任务 ${id.slice(0, 8)}… 的输出`)
  } catch (e: any) {
    ElMessage.error('载入任务结果失败: ' + (e?.message || ''))
  }
}

const clearCommand = () => {
  commandText.value = ''
}

const clearResult = () => {
  executionResults.value = []
}

function utf8ToB64(s: string): string {
  return btoa(unescape(encodeURIComponent(s)))
}

function buildExecutePayloadObject() {
  return {
    machine_ids: selectedMachines.value.map((m) => m.id),
    command: commandText.value.trim(),
    timeout: jobTimeoutSec.value
  }
}

function openScriptDialog() {
  flushTargetReconcile()
  const b64 = utf8ToB64(JSON.stringify(buildExecutePayloadObject()))
  const ids = selectedMachines.value.map((m) => m.id).join(',')
  const escaped = commandText.value.trim().replace(/\\/g, '\\\\').replace(/'/g, "'\\''")

  generatedScriptText.value = `#!/usr/bin/env bash
# OpsFleet 作业中心 — 在控制机保存执行；请勿提交含令牌的内容到 Git。
set -euo pipefail

# export OPSFLEET_API_URL="https://控制台:端口/ft-api"
# export OPSFLEET_TOKEN="登录令牌（与 ~/.config/ai-sre/opsfleet_token 一致）"

if ! command -v ai-sre >/dev/null 2>&1; then
  echo "未找到 ai-sre，请先安装或改用注释中的 curl 示例" >&2
  exit 1
fi

ai-sre job run --machines "${ids}" --timeout ${jobTimeoutSec.value} --print-console-url --wait -c '${escaped}'

# ─── 备选：curl + base64 请求体（BODY_B64 为当前页面生成的 JSON 经 base64）───
# BODY_B64='${b64}'
# BODY=$(printf '%s' "$BODY_B64" | base64 -d)
# RESP=$(curl -fsS "$API_BASE/api/job/execute" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$BODY")
# echo "$RESP" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['jobId'])"
`
  scriptDialogVisible.value = true
}

async function copyGeneratedScript() {
  await copyText(generatedScriptText.value)
  ElMessage.success('脚本已复制')
}

const setupKeyboardShortcuts = () => {
  const onKey = (e: KeyboardEvent) => {
    if (!e.ctrlKey) return
    if (e.key === 'Enter') {
      e.preventDefault()
      if (!executeDisabled.value) void executeCommands()
    }
    if (e.key === 'k') {
      e.preventDefault()
      clearCommand()
    }
    if (e.key === 'r') {
      e.preventDefault()
      void loadMachineList()
    }
    if (e.key === 'l') {
      e.preventDefault()
      clearResult()
    }
  }
  window.addEventListener('keydown', onKey)
  return () => window.removeEventListener('keydown', onKey)
}

let offKeys: (() => void) | null = null
const importedJobIds = new Set<string>()

watch(
  () => route.query.jobId,
  async (jid) => {
    const raw = typeof jid === 'string' ? jid : Array.isArray(jid) ? jid[0] : ''
    const id = (raw || '').trim()
    if (!id || importedJobIds.has(id)) return
    importedJobIds.add(id)
    await importResultsForJob(id, '链接/CLI')
    const q = { ...route.query } as Record<string, unknown>
    delete q.jobId
    router.replace({ path: route.path, query: q })
  },
  { immediate: true }
)

onMounted(() => {
  loadCommandHistory()
  void loadMachineList()
  offKeys = setupKeyboardShortcuts()
})

onUnmounted(() => {
  offKeys?.()
  offKeys = null
})
</script>

<style scoped>
/* 作业中心：紧凑、无内部滚动条（溢出裁剪；完整输出用「复制」） */
.job-center {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 4px 10px 6px;
  max-width: none;
  margin: 0;
  overflow: hidden;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.job-center::-webkit-scrollbar,
.job-center *::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  flex-shrink: 0;
}

.page-header__titles h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.page-header__sub {
  margin: 2px 0 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.35;
}

.page-header__sub code {
  font-size: 11px;
  padding: 0 3px;
}

.page-header__actions {
  display: flex;
  flex-shrink: 0;
}

.job-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-blank);
  cursor: pointer;
  color: var(--el-text-color-regular);
}

.job-icon-btn:hover:not(:disabled) {
  border-color: var(--el-color-primary);
  color: var(--el-color-primary);
}

.job-icon-btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.job-icon-btn__icon--spin {
  animation: job-spin 0.9s linear infinite;
}

@keyframes job-spin {
  to {
    transform: rotate(360deg);
  }
}

.job-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-radius: var(--el-border-radius-base);
}

.job-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  padding: 8px 10px !important;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.job-card-inner {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  overflow: hidden;
}

.job-section--targets {
  flex-shrink: 0;
}

.field-hint {
  margin: 0 0 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
  line-height: 1.3;
}

.targets-body {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-height: 0;
}

.targets-textarea {
  flex: 1;
  min-width: 0;
}

.targets-textarea :deep(.el-textarea__inner) {
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 1.35;
  padding: 4px 8px;
  overflow-y: hidden !important;
  resize: none !important;
}

.opt-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 10px;
  padding: 4px 0;
  flex-shrink: 0;
  border-top: 1px solid var(--el-border-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.opt-label {
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.opt-timeout {
  width: 88px;
}

.opt-inline {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.job-divider {
  margin: 4px 0;
  flex-shrink: 0;
}

.job-split {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
  overflow: hidden;
}

@media (max-width: 1024px) {
  .job-split {
    grid-template-columns: 1fr;
  }
}

.job-section {
  margin: 0;
  min-height: 0;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
  flex-shrink: 0;
}

.section-head h3 {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
}

.section-meta {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.section-head__tools {
  display: flex;
  align-items: center;
  gap: 4px;
}

.filter-select {
  width: 72px;
}

.cmd-shell {
  display: flex;
  gap: 6px;
  flex: 1;
  min-height: 0;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  padding: 6px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}

.cmd-shell__prompt {
  font-family: ui-monospace, monospace;
  font-weight: 700;
  font-size: 12px;
  color: var(--el-color-primary);
  line-height: 20px;
  flex-shrink: 0;
}

.cmd-shell__input {
  flex: 1;
  min-width: 0;
  min-height: 0;
}

.cmd-shell__input :deep(.el-textarea__inner) {
  font-family: ui-monospace, monospace;
  font-size: 11px;
  line-height: 1.4;
  background: transparent;
  border: none !important;
  box-shadow: none !important;
  min-height: 4.2rem !important;
  max-height: 4.2rem !important;
  padding: 2px 4px;
  overflow-y: hidden !important;
}

.cmd-warn {
  margin-top: 4px;
  padding: 4px 8px;
  border-radius: var(--el-border-radius-base);
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-5);
  flex-shrink: 0;
  max-height: 3.6em;
  overflow: hidden;
}

.cmd-warn__line {
  display: flex;
  gap: 6px;
  font-size: 11px;
  color: var(--el-color-danger-dark-2);
}

.cmd-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
  flex-shrink: 0;
}

.job-id-bar {
  margin-top: 4px;
  font-size: 11px;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.job-id-bar__code {
  font-size: 10px;
  background: var(--el-fill-color);
  padding: 1px 4px;
  border-radius: 3px;
}

.job-hint {
  margin: 4px 0 0;
  font-size: 10px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

.history-dd__pre {
  margin: 0;
  max-width: 280px;
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.poll-banner {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--el-color-primary);
  margin-bottom: 4px;
  flex-shrink: 0;
}

.poll-banner__icon {
  animation: job-spin 1s linear infinite;
}

.out-panel {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 4px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-lighter);
}

.out-filter-tip {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 2px;
  flex-shrink: 0;
}

.out-empty {
  text-align: center;
  padding: 12px 8px;
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.out-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  padding: 6px;
  margin-bottom: 6px;
}

.out-card:last-child {
  margin-bottom: 0;
}

.out-card__head {
  display: flex;
  justify-content: space-between;
  gap: 6px;
  margin-bottom: 4px;
  padding-bottom: 4px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.out-card__title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.out-card__name {
  font-weight: 600;
  font-size: 12px;
}

.out-card__id {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, monospace;
}

.out-block__label {
  font-size: 10px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 2px;
}

.out-block pre {
  margin: 0;
  padding: 4px 6px;
  border-radius: 4px;
  font-size: 10px;
  line-height: 1.35;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 3.6em;
  overflow: hidden;
}

.out-block--stdout pre {
  background: var(--el-fill-color-light);
  border-left: 2px solid var(--el-color-primary);
}

.out-block--stderr pre {
  background: var(--el-color-danger-light-9);
  border-left: 2px solid var(--el-color-danger);
}

.script-dialog-help {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin: 0 0 8px;
}

.script-ta :deep(.el-textarea__inner) {
  font-family: ui-monospace, monospace;
  font-size: 10px;
  overflow-y: hidden !important;
}
</style>
