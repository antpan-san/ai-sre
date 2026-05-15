<template>
  <div class="job-center page-shell page-shell--fill">
    <div class="page-header">
      <div class="page-header__titles">
        <h2>作业中心</h2>
        <p class="page-header__sub">在在线机器上批量执行 shell；结果由 Agent 回传，可能需要数秒至一分钟。</p>
      </div>
      <div class="page-header__actions">
        <el-tooltip content="选择目标机 → 输入命令 → 执行；Ctrl+Enter 快捷执行" placement="bottom-end">
          <el-button text type="primary" size="small">说明</el-button>
        </el-tooltip>
        <el-tooltip content="刷新在线机器列表" placement="bottom-end">
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
      <!-- 机器 -->
      <section class="job-section">
        <div class="section-head">
          <h3>目标机器</h3>
          <span class="section-meta">{{ machines.length }} 台在线 · 已选 {{ selectedMachines.length }} 台</span>
        </div>
        <div class="transfer-wrap" v-loading="machinesLoading">
          <el-transfer
            v-model="transferValue"
            :data="transferData"
            :titles="['待选', '已选']"
            :filterable="true"
            filter-placeholder="搜索 ID / IP / 名称"
            :format="{ noMatch: '无匹配', noData: '暂无在线机器', all: '全部', confirm: '确认' }"
            @change="handleTransferChange"
          >
            <template #default="{ option }">
              <div class="transfer-row">
                <span class="transfer-row__name">{{ option.name }}</span>
                <span class="transfer-row__ip">{{ option.ip }}</span>
                <el-tag type="success" size="small" effect="plain">在线</el-tag>
              </div>
            </template>
          </el-transfer>
        </div>
      </section>

      <el-divider class="job-divider" />

      <div class="job-split">
        <!-- 命令 -->
        <section class="job-section job-section--cmd">
          <div class="section-head">
            <h3>命令</h3>
          </div>
          <div class="cmd-shell">
            <span class="cmd-shell__prompt" aria-hidden="true">$</span>
            <el-input
              v-model="commandText"
              type="textarea"
              :rows="10"
              :autosize="{ minRows: 10, maxRows: 22 }"
              placeholder="每行一条命令；避免 vim/top 等交互式命令"
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
            <el-button
              type="primary"
              :disabled="selectedMachines.length === 0 || !commandText.trim() || executing"
              :loading="executing"
              @click="executeCommands"
            >
              <el-icon><CirclePlus /></el-icon>
              执行
            </el-button>
            <el-button @click="clearCommand">清空</el-button>
          </div>
          <p class="job-hint">Ctrl+Enter 执行 · Ctrl+K 清空命令 · Ctrl+R 刷新机器 · Ctrl+L 清空结果</p>
        </section>

        <!-- 结果 -->
        <section class="job-section job-section--out">
          <div class="section-head">
            <h3>输出</h3>
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
            正在等待 Agent 回传…
          </div>

          <div class="out-scroll">
            <div v-if="resultFilter !== 'all'" class="out-filter-tip">
              当前：{{ resultFilter === 'success' ? '成功' : '失败' }} · {{ filteredResults.length }} 条
            </div>
            <div v-if="!filteredResults.length" class="out-empty">
              {{ executionResults.length ? '无匹配结果' : '执行后在此查看每台机器输出' }}
            </div>

            <article
              v-for="(result, index) in filteredResults"
              :key="result.machineId + '-' + index + '-' + result.executionTime"
              class="out-card"
            >
              <header class="out-card__head">
                <div class="out-card__title">
                  <span class="out-card__name">{{ result.machineName }}</span>
                  <span class="out-card__id">{{ result.machineIP || result.machineId }}</span>
                  <el-tag :type="tagType(result)" size="small">{{ statusLabel(result) }}</el-tag>
                  <el-tag v-if="result.exitCode != null" size="small" effect="plain" type="info">
                    exit {{ result.exitCode }}
                  </el-tag>
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
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  RefreshRight,
  CirclePlus,
  Delete,
  Clock,
  ArrowDown,
  Warning,
  Loading
} from '@element-plus/icons-vue'
import { executeCommand, getExecutionResult, getAvailableMachines } from '../../api/job'
import type { JobSubTaskResult } from '../../api/job'
import { copyTextToClipboard } from '../../utils/clipboard'
import type { Machine } from '../../types'

const COMMAND_HISTORY_KEY = 'jobCenterCommandHistory'
const MAX_HISTORY = 20
const POLL_MS = 900
const MAX_WAIT_MS = 120_000

const executing = ref(false)
const polling = ref(false)
const machinesLoading = ref(false)
const machines = ref<Machine[]>([])

const selectedMachines = ref<Machine[]>([])
const transferValue = ref<string[]>([])

const transferData = computed(() =>
  machines.value.map((m) => ({
    key: m.id,
    label: `${m.name} · ${m.ip}`,
    id: m.id,
    name: m.name,
    ip: m.ip,
    status: m.status
  }))
)

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
}

const executionResults = ref<ExecutionResult[]>([])
const resultFilter = ref('all')

const filteredResults = computed(() => {
  if (resultFilter.value === 'all') return executionResults.value
  if (resultFilter.value === 'success') return executionResults.value.filter((r) => r.success)
  return executionResults.value.filter((r) => !r.success)
})

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
    interactive.forEach((c) => {
      if (t.startsWith(c)) errors.push(`第 ${index + 1} 行：交互命令「${c}」无法在作业中心执行`)
    })
  })
  commandErrors.value = errors
  return errors
}

watch(commandText, (v) => checkCommandSyntax(v))

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
  ElMessage.warning('等待结果超时，任务可能仍在执行，请在「执行记录」中查看')
  return latest
}

function mapRows(rows: JobSubTaskResult[], finishedAt: string): ExecutionResult[] {
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
      exitCode: r.exit_code
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
  return r.status || '未知'
}

const tagType = (r: ExecutionResult) => {
  if (r.success) return 'success'
  if (r.status === 'cancelled') return 'info'
  return 'danger'
}

const copyOneResult = async (r: ExecutionResult) => {
  const text = [`# ${r.machineName} (${r.machineIP || r.machineId})`, r.stdout && `--- stdout ---\n${r.stdout}`, r.stderr && `--- stderr ---\n${r.stderr}`]
    .filter(Boolean)
    .join('\n\n')
  try {
    await copyTextToClipboard(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const copyAllResults = async () => {
  if (!filteredResults.value.length) return
  const parts = filteredResults.value.map((r, i) => {
    const head = `## ${i + 1}. ${r.machineName} (${r.machineIP || r.machineId}) [${statusLabel(r)}]`
    const body = [r.stdout && `stdout:\n${r.stdout}`, r.stderr && `stderr:\n${r.stderr}`].filter(Boolean).join('\n\n')
    return `${head}\n${body}`
  })
  try {
    await copyTextToClipboard(parts.join('\n\n---\n\n'))
    ElMessage.success('已复制全部输出')
  } catch {
    ElMessage.error('复制失败')
  }
}

const loadMachineList = async () => {
  machinesLoading.value = true
  try {
    machines.value = await getAvailableMachines()
    transferValue.value = []
    selectedMachines.value = []
  } catch (e: any) {
    ElMessage.error('获取机器列表失败: ' + (e?.message || e?.msg || '未知错误'))
  } finally {
    machinesLoading.value = false
  }
}

const refreshMachines = () => void loadMachineList()

const handleTransferChange = (value: string[]) => {
  transferValue.value = value
  selectedMachines.value = machines.value.filter((m) => value.includes(m.id))
}

const executeCommands = async () => {
  if (!selectedMachines.value.length) {
    ElMessage.warning('请至少选择一台机器')
    return
  }
  if (!commandText.value.trim()) {
    ElMessage.warning('请输入命令')
    return
  }

  const errors = checkCommandSyntax(commandText.value)
  if (errors.length) {
    try {
      await ElMessageBox.confirm(`检测到 ${errors.length} 条提示，仍要执行？\n\n${errors.join('\n')}`, '确认执行', {
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
      command: commandText.value.trim()
    })
    const rows = await waitForJobResults(jobId)
    const mapped = mapRows(rows, finishedAt)
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

const clearCommand = () => {
  commandText.value = ''
}

const clearResult = () => {
  executionResults.value = []
}

const setupKeyboardShortcuts = () => {
  const onKey = (e: KeyboardEvent) => {
    if (!e.ctrlKey) return
    if (e.key === 'Enter') {
      e.preventDefault()
      if (selectedMachines.value.length && commandText.value.trim() && !executing.value) void executeCommands()
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
  padding: 0 var(--page-padding-x) 20px;
  box-sizing: border-box;
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
  max-width: 52ch;
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
  color: var(--el-text-color-primary);
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
  margin: 16px 0;
}

.job-split {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(300px, 1.1fr);
  gap: 20px;
  align-items: start;
}

@media (max-width: 1024px) {
  .job-split {
    grid-template-columns: 1fr;
  }
}

.transfer-wrap {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 10px;
  background: var(--el-fill-color-blank);
}

.transfer-wrap :deep(.el-transfer) {
  display: flex;
  width: 100%;
  justify-content: space-between;
  align-items: stretch;
  gap: 8px;
}

.transfer-wrap :deep(.el-transfer-panel) {
  flex: 1;
  min-width: 0;
}

.transfer-wrap :deep(.el-transfer-panel__body) {
  height: 260px;
}

.transfer-wrap :deep(.el-transfer__buttons) {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 6px;
  padding: 0 4px;
}

.transfer-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 13px;
}

.transfer-row__name {
  font-weight: 500;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 0 1 auto;
  max-width: 42%;
}

.transfer-row__ip {
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.cmd-shell {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  border: 1px solid var(--el-border-color);
  border-radius: 10px;
  padding: 10px 10px 10px 8px;
  background: var(--el-fill-color-light);
  min-height: 200px;
}

.cmd-shell:focus-within {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 0 0 1px var(--el-color-primary-light-7);
}

.cmd-shell__prompt {
  font-family: ui-monospace, Menlo, Monaco, Consolas, monospace;
  font-weight: 700;
  color: var(--el-color-primary);
  line-height: 22px;
  padding-top: 2px;
  flex-shrink: 0;
}

.cmd-shell__input {
  flex: 1;
  min-width: 0;
}

.cmd-shell__input :deep(.el-textarea__inner) {
  font-family: ui-monospace, Menlo, Monaco, Consolas, monospace;
  font-size: 13px;
  line-height: 1.55;
  background: transparent;
  box-shadow: none !important;
  border: none !important;
  padding: 2px 4px;
  resize: vertical;
  min-height: 180px;
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
  align-items: flex-start;
  font-size: 12px;
  color: var(--el-color-danger-dark-2);
  margin-bottom: 4px;
}
.cmd-warn__line:last-child {
  margin-bottom: 0;
}

.cmd-warn__icon {
  flex-shrink: 0;
  margin-top: 1px;
}

.cmd-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
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
  word-break: break-all;
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
  margin-bottom: 6px;
}

.out-empty {
  text-align: center;
  padding: 36px 12px;
  color: var(--el-text-color-placeholder);
  font-size: 13px;
}

.out-card {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 10px;
  padding: 12px;
  margin-bottom: 10px;
  box-shadow: var(--el-box-shadow-lighter);
}

.out-card:last-child {
  margin-bottom: 0;
}

.out-card__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
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
  min-width: 0;
}

.out-card__name {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.out-card__id {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: ui-monospace, Menlo, Monaco, Consolas, monospace;
}

.out-card__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.out-card__time {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.out-block__label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
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
  color: var(--el-text-color-primary);
  border-left: 3px solid var(--el-color-primary);
}

.out-block--stderr pre {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger-dark-2);
  border-left: 3px solid var(--el-color-danger);
}
</style>
