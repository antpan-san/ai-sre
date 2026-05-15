<template>
  <div class="job-center page-shell">
    <div class="page-header">
      <div class="page-header__titles">
        <h2>作业中心</h2>
        <p class="page-header__sub">
          选择一个或多个<strong>在线</strong> Agent 机器，下发 shell；也可在装有 ai-sre 的控制机执行
          <code>ai-sre job run … --print-console-url</code>
          ，用浏览器打开控制台链接在此处查看同源结果。
        </p>
      </div>
      <div class="page-header__actions">
        <el-tooltip content="刷新在线列表（文本解析目标的快照）" placement="bottom-end">
          <button
            type="button"
            class="job-icon-btn"
            :disabled="machinesLoading"
            aria-label="刷新机器列表"
            @click="refreshMachines"
          >
            <el-icon class="job-icon-btn__icon" :class="{ 'job-icon-btn__icon--spin': machinesLoading }">
              <RefreshRight />
            </el-icon>
          </button>
        </el-tooltip>
      </div>
    </div>

    <el-card class="job-card" shadow="hover">
      <!-- ① 目标机器 -->
      <section class="job-section">
        <div class="section-head">
          <h3>目标机器</h3>
          <span class="section-meta">{{ machines.length }} 台在线 · 已选 {{ selectedMachines.length }} 台</span>
        </div>
        <p class="field-hint">
          在下方文本框填入 <strong>UUID</strong>、<strong>IP</strong> 或<strong>主机名</strong>（空格 / 逗号 / 换行分隔），系统将按<strong>当前在线</strong>快照自动解析<strong>已选</strong>；可先点顶栏旁刷新以保持列表最新。确认执行时会再次即时解析一遍。
        </p>
        <div class="targets-body" v-loading="machinesLoading">
          <el-input
            v-model="machineTargetsRaw"
            type="textarea"
            :autosize="{ minRows: 2, maxRows: 8 }"
            placeholder="示例：550e8400-e29b… &#10;192.168.1.10&#10;k8s-worker-02"
            class="targets-textarea"
          />
          <div class="targets-actions">
            <el-button size="small" @click="clearTargets">清空目标</el-button>
          </div>
        </div>
      </section>

      <!-- ③ 选项 -->
      <el-collapse v-model="optionsOpen" class="job-collapse">
        <el-collapse-item title="执行选项" name="opt">
          <div class="opt-grid">
            <div class="opt-field">
              <span class="opt-label">单任务超时（秒）</span>
              <el-select v-model="jobTimeoutSec" placeholder="超时" style="width: 160px">
                <el-option label="60" :value="60" />
                <el-option label="120" :value="120" />
                <el-option label="300（5min）" :value="300" />
                <el-option label="600（10min）" :value="600" />
                <el-option label="1800（30min）" :value="1800" />
                <el-option label="3600（1h）" :value="3600" />
              </el-select>
            </div>
            <div class="opt-field opt-field--switch">
              <el-switch v-model="confirmDangerPatterns" />
              <span class="opt-label-inline">检测到 rm -rf / 交互命令等时再二次确认（建议开启）</span>
            </div>
            <div class="opt-field opt-field--switch">
              <el-switch v-model="blockIfUnresolvedTargets" />
              <span class="opt-label-inline">文本目标中存在无法识别的 token 时阻止执行（直到改正）</span>
            </div>
          </div>
        </el-collapse-item>
      </el-collapse>

      <el-divider class="job-divider" />

      <div class="job-split">
        <!-- ② 指令 -->
        <section class="job-section job-section--cmd">
          <div class="section-head">
            <h3>命令 / 脚本</h3>
          </div>
          <div class="cmd-shell">
            <span class="cmd-shell__prompt" aria-hidden="true">$</span>
            <el-input
              v-model="commandText"
              type="textarea"
              :autosize="{ minRows: 10, maxRows: 22 }"
              placeholder="多行 shell；避免 vim / top 等需 TTY 的交互命令"
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
            <el-dropdown v-if="commandHistory.length" trigger="click" :max-height="280">
              <el-button size="default">
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
            <el-button type="primary" :disabled="executeDisabled" :loading="executing" @click="executeCommands">
              <el-icon><CirclePlus /></el-icon>
              执行
            </el-button>
            <el-button @click="clearCommand">清空命令</el-button>
            <el-button @click="openScriptDialog" :disabled="!canGenerateScript">
              <el-icon><DocumentCopy /></el-icon>
              一键脚本
            </el-button>
          </div>
          <div v-if="lastJobId" class="job-id-bar">
            <span class="job-id-bar__label">最近一次 jobId:</span>
            <code class="job-id-bar__code">{{ lastJobId }}</code>
            <el-button link type="primary" size="small" @click="copyText(lastJobLink)">复制控制台链接</el-button>
          </div>
          <p class="job-hint">Ctrl+Enter 执行 · Ctrl+K 清空命令 · Ctrl+R 刷新在线快照 · Ctrl+L 清空右侧结果</p>
        </section>

        <!-- ④⑤ 输出 -->
        <section class="job-section job-section--out">
          <div class="section-head">
            <h3>执行结果</h3>
            <div class="section-head__tools">
              <el-select v-model="resultFilter" placeholder="筛选" size="small" class="filter-select">
                <el-option label="全部" value="all" />
                <el-option label="成功" value="success" />
                <el-option label="失败" value="failed" />
              </el-select>
              <el-button size="small" text type="primary" :disabled="!filteredResults.length" @click="copyAllResults">
                复制全部
              </el-button>
              <el-button size="small" @click="clearResult">清空</el-button>
            </div>
          </div>

          <div v-if="polling" class="poll-banner">
            <el-icon class="poll-banner__icon"><Loading /></el-icon>
            等待 Agent 回传中…（与 ai-sre <code>job run --wait</code> 数据源相同）
          </div>

          <div class="out-scroll">
            <div v-if="resultFilter !== 'all'" class="out-filter-tip">
              筛选：{{ resultFilter === 'success' ? '成功' : '失败' }} · {{ filteredResults.length }} 条
            </div>
            <div v-if="!filteredResults.length" class="out-empty">
              {{
                executionResults.length
                  ? '无筛选匹配'
                  : '从本页发起执行或通过 ?jobId= / ai-sre 链接导入后在此查看'
              }}
            </div>

            <article
              v-for="(result, index) in filteredResults"
              :key="result.machineId + '-' + index + '-' + result.executionTime + '-' + (result.jobId || '')"
              class="out-card"
            >
              <header class="out-card__head">
                <div class="out-card__title">
                  <el-tag v-if="result.sourceLabel" size="small" effect="dark" type="info">{{ result.sourceLabel }}</el-tag>
                  <span class="out-card__name">{{ result.machineName }}</span>
                  <span class="out-card__id">{{ result.machineIP || result.machineId }}</span>
                  <el-tag :type="tagType(result)" size="small">{{ statusLabel(result) }}</el-tag>
                  <el-tag v-if="result.exitCode != null" size="small" effect="plain" type="info">
                    exit {{ result.exitCode }}
                  </el-tag>
                  <el-tag v-if="result.jobId" size="small" effect="plain">{{ result.jobId.slice(0, 8) }}…</el-tag>
                </div>
                <div class="out-card__meta">
                  <span class="out-card__time">{{ formatDate(result.executionTime) }}</span>
                  <el-button text type="primary" size="small" @click="copyOneResult(result)">复制</el-button>
                </div>
              </header>
              <div v-if="result.stdout" class="out-block out-block--stdout">
                <div class="out-block__label">stdout</div>
                <pre>{{ result.stdout }}</pre>
              </div>
              <div v-if="result.stderr" class="out-block out-block--stderr">
                <div class="out-block__label">stderr / 错误</div>
                <pre>{{ result.stderr }}</pre>
              </div>
            </article>
          </div>
        </section>
      </div>
    </el-card>

    <el-dialog v-model="scriptDialogVisible" title="一键脚本（需在控制机上设置令牌）" width="640px" destroy-on-close>
      <p class="script-dialog-help">
        将下方脚本保存为 <code>job-run.sh</code>，在已安装 ai-sre 或已配置令牌的环境运行；依赖 <code>curl</code>、<code>python3</code>（解析 JSON）。
        勿把含令牌的脚本提交到 Git。
      </p>
      <el-input v-model="generatedScriptText" type="textarea" readonly :autosize="{ minRows: 16, maxRows: 26 }" class="script-ta" />
      <template #footer>
        <el-button @click="scriptDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="copyGeneratedScript">复制脚本</el-button>
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
const optionsOpen = ref(['opt'])

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
    ElMessage.error('请先修正无法识别的目标，或关闭上方「文本目标中存在无法识别的 token 时阻止执行」')
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
.job-center {
  width: 100%;
  box-sizing: border-box;
  padding: var(--page-padding-y) var(--page-padding-x) 24px;
  max-width: 1440px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.page-header__titles h2 {
  margin: 0 0 4px;
  font-size: var(--page-header-title-max);
  font-weight: 600;
  color: var(--apple-ink);
}

.page-header__sub {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
  max-width: 62ch;
}

.page-header__sub code {
  font-size: 12px;
  padding: 1px 4px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
}

.page-header__actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.job-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
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
  border-radius: 12px;
}

.job-section {
  margin: 0;
}

.field-hint {
  margin: 0 0 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}

.targets-textarea {
  margin-bottom: 8px;
}

.targets-textarea :deep(.el-textarea__inner) {
  font-family: ui-monospace, Menlo, Consolas, monospace;
  font-size: 12px;
}

.targets-actions {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.job-collapse {
  margin: 12px 0 0;
  border: none;
  --el-collapse-header-height: 40px;
}

.job-collapse :deep(.el-collapse-item__header) {
  font-weight: 600;
  font-size: 13px;
}

.opt-grid {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 4px 0 8px;
}

.opt-field {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.opt-field--switch {
  align-items: flex-start;
}

.opt-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
  min-width: 120px;
}

.opt-label-inline {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  flex: 1;
  line-height: 1.45;
}

.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

.section-head h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
}

.section-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.section-head__tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.filter-select {
  width: 100px;
}

.job-divider {
  margin: 18px 0;
}

.job-split {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(300px, 1.05fr);
  gap: 20px;
  align-items: start;
}

@media (max-width: 1024px) {
  .job-split {
    grid-template-columns: 1fr;
  }
}

.targets-body {
  min-height: 48px;
  border-radius: 10px;
}

.cmd-shell {
  display: flex;
  gap: 8px;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  padding: 10px;
  background: var(--el-fill-color-light);
  min-height: 180px;
}

.cmd-shell:focus-within {
  border-color: var(--el-color-primary-light-5);
}

.cmd-shell__prompt {
  font-family: ui-monospace, monospace;
  font-weight: 700;
  color: var(--el-color-primary);
  line-height: 22px;
}

.cmd-shell__input {
  flex: 1;
  min-width: 0;
}

.cmd-shell__input :deep(.el-textarea__inner) {
  font-family: ui-monospace, monospace;
  font-size: 13px;
  line-height: 1.55;
  background: transparent;
  border: none !important;
  box-shadow: none !important;
  min-height: 160px;
}

.cmd-warn {
  margin-top: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-5);
}

.cmd-warn__line {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: var(--el-color-danger-dark-2);
  margin-bottom: 4px;
}

.cmd-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.job-id-bar {
  margin-top: 10px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.job-id-bar__code {
  font-size: 11px;
  background: var(--el-fill-color);
  padding: 2px 6px;
  border-radius: 4px;
}

.job-hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.history-dd__pre {
  margin: 0;
  max-width: 360px;
  max-height: 72px;
  overflow: auto;
  font-size: 12px;
  white-space: pre-wrap;
}

.poll-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--el-color-primary);
  margin-bottom: 10px;
}

.poll-banner__icon {
  animation: job-spin 1s linear infinite;
}

.out-scroll {
  max-height: min(62vh, 720px);
  overflow-y: auto;
  padding: 4px 2px 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  background: var(--el-fill-color-lighter);
}

.out-filter-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  padding: 6px;
}

.out-empty {
  text-align: center;
  padding: 32px 12px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.out-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 12px;
  margin-bottom: 10px;
}

.out-card__head {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.out-card__title {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}

.out-card__name {
  font-weight: 600;
  font-size: 14px;
}

.out-card__id {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, monospace;
}

.out-card__time {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.out-block__label {
  font-size: 11px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.out-block pre {
  margin: 0;
  padding: 10px;
  border-radius: 8px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 280px;
  overflow: auto;
}

.out-block--stdout pre {
  background: var(--el-fill-color-light);
  border-left: 3px solid var(--el-color-primary);
}

.out-block--stderr pre {
  background: var(--el-color-danger-light-9);
  border-left: 3px solid var(--el-color-danger);
}

.script-dialog-help {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  margin: 0 0 10px;
  line-height: 1.5;
}

.script-ta :deep(.el-textarea__inner) {
  font-family: ui-monospace, monospace;
  font-size: 11px;
}
</style>
