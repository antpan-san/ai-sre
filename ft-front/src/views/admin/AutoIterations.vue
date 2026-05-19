<template>
  <div class="auto-iterations page-shell page-shell--fill">
    <div class="page-toolbar">
      <h2 class="page-toolbar__title">自动迭代</h2>
      <div class="page-toolbar__actions">
        <el-button type="primary" size="small" @click="createDrawerOpen = true">新建需求</el-button>
        <el-popover placement="bottom-end" :width="300" trigger="click" @show="loadSettings">
          <template #reference>
            <el-button size="small" :loading="settingsLoading">设置</el-button>
          </template>
          <div v-loading="settingsLoading" class="settings-popover">
            <el-form label-width="100px" size="small">
              <el-form-item label="启用">
                <el-switch v-model="settings.enabled" @change="scheduleSaveSettings" />
              </el-form-item>
              <el-form-item label="最大并发">
                <el-input-number
                  v-model="settings.max_concurrent"
                  :min="1"
                  :max="20"
                  controls-position="right"
                  @change="scheduleSaveSettings"
                />
              </el-form-item>
              <el-form-item label="高风险需审批">
                <el-switch v-model="settings.high_risk_requires_approval" @change="scheduleSaveSettings" />
              </el-form-item>
              <el-form-item label="自动派发">
                <el-switch v-model="settings.auto_dispatch_enabled" @change="scheduleSaveSettings" />
              </el-form-item>
              <el-form-item label="低风险自动上线">
                <el-switch v-model="settings.low_risk_auto_deploy_enabled" @change="scheduleSaveSettings" />
              </el-form-item>
              <el-form-item label="GitHub 同步">
                <el-switch v-model="settings.github_sync_enabled" @change="scheduleSaveSettings" />
              </el-form-item>
              <el-form-item label="钉钉通知">
                <el-switch v-model="settings.dingtalk_notify_enabled" @change="scheduleSaveSettings" />
              </el-form-item>
              <el-form-item label="钉钉">
                <el-tag :type="settings.has_dingtalk_webhook ? 'success' : 'info'" size="small">
                  {{ settings.has_dingtalk_webhook ? '已配置' : '未配置' }}
                </el-tag>
              </el-form-item>
            </el-form>
            <p v-if="!settings.enabled" class="settings-warn">未启用时无法提交新任务。</p>
          </div>
        </el-popover>
        <el-tooltip content="刷新任务列表" placement="bottom">
          <el-button size="small" :loading="listLoading" :icon="RefreshRight" circle @click="refreshList" />
        </el-tooltip>
      </div>
    </div>

    <div class="workbench">
      <el-row :gutter="16" class="main-row">
        <el-col :xs="24" :lg="9" class="pane-col">
          <el-card shadow="never" v-loading="listLoading" class="pane-card list-card">
          <template #header>
            <div class="card-header">
              <span>任务列表</span>
              <el-tag v-if="total > 0" size="small" type="info">{{ total }}</el-tag>
            </div>
          </template>
          <div class="list-filters">
            <el-input
              v-model="listFilters.keyword"
              placeholder="搜索标题 / 需求"
              clearable
              size="small"
              class="filter-grow"
              @keyup.enter="applyListFilters"
              @clear="applyListFilters"
            />
            <el-select
              v-model="listFilters.status"
              placeholder="状态"
              clearable
              size="small"
              class="filter-status"
              @change="applyListFilters"
            >
              <el-option
                v-for="opt in STATUS_FILTER_OPTIONS"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
            <el-select
              v-model="listFilters.source"
              placeholder="来源"
              clearable
              size="small"
              class="filter-source"
              @change="applyListFilters"
            >
              <el-option
                v-for="opt in SOURCE_FILTER_OPTIONS"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
            <el-button type="primary" size="small" @click="applyListFilters">筛选</el-button>
            <el-button size="small" link @click="resetListFilters">重置</el-button>
          </div>
          <div class="pane-scroll list-scroll">
            <el-table
              :data="rows"
              stripe
              size="small"
              highlight-current-row
              :current-row-key="selectedId"
              row-key="id"
              :row-class-name="rowClassName"
              empty-text="暂无任务"
              class="list-table"
              @row-click="selectRow"
            >
              <el-table-column prop="title" label="标题" min-width="140" show-overflow-tooltip />
              <el-table-column prop="status" label="状态" width="88">
                <template #default="{ row }">
                  <el-tag :type="statusTagType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="时间" width="132">
                <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
              </el-table-column>
            </el-table>
          </div>
          <div class="pager">
            <el-pagination
              v-model:current-page="page"
              v-model:page-size="pageSize"
              :page-sizes="[10, 20, 50]"
              :total="total"
              layout="total, prev, pager, next"
              small
              @size-change="loadList"
              @current-change="loadList"
            />
          </div>
          </el-card>
        </el-col>

        <el-col :xs="24" :lg="15" class="pane-col">
          <el-card shadow="never" class="pane-card detail-card">
          <template #header>
            <div class="detail-header">
              <span class="detail-title">{{ iteration?.title || '任务详情' }}</span>
              <div v-if="iteration" class="detail-header-tags">
                <el-tag :type="statusTagType(iteration.status)" size="small">
                  {{ statusLabel(iteration.status) }}
                </el-tag>
                <el-tag v-if="sseConnected" type="success" size="small" effect="plain">实时</el-tag>
                <el-tag v-if="detailRefreshing" size="small" type="info" effect="plain">更新中</el-tag>
                <span v-if="phaseLabel" class="phase-text">{{ phaseLabel }}</span>
              </div>
            </div>
          </template>

          <el-empty v-if="!selectedId" class="detail-empty" description="选择左侧任务查看详情与日志" :image-size="48" />
          <div v-else class="detail-body" v-loading="detailLoading">
            <div v-if="iteration" class="detail-scroll">
            <el-alert
              v-if="statusHint"
              :type="statusHint.type"
              :title="statusHint.title"
              :closable="false"
              show-icon
              class="status-hint"
            />

            <div class="meta-chips">
              <span v-if="iteration.risk_level" class="meta-chip">风险 {{ iteration.risk_level }}</span>
              <span v-if="iteration.topic" class="meta-chip">Topic {{ iteration.topic }}</span>
              <span class="meta-chip">{{ sourceLabel(iteration.source) }}</span>
              <span v-if="iteration.created_by" class="meta-chip">创建 {{ iteration.created_by }}</span>
            </div>

            <div class="action-bar">
              <el-button-group v-if="canStart || canRequeue">
                <el-button v-if="canStart" type="primary" size="small" @click="act('start')">
                  启动开发
                </el-button>
                <el-button v-if="canRequeue" type="primary" size="small" plain @click="act('start')">
                  重新入队
                </el-button>
              </el-button-group>
              <el-button v-if="canApprove" type="success" size="small" @click="act('approve')">
                批准上线
              </el-button>
              <el-button size="small" :disabled="!canPause" @click="act('pause')">暂停</el-button>
              <el-button size="small" :disabled="!canResume" @click="act('resume')">继续</el-button>
              <el-button size="small" :disabled="!canCancel" @click="act('cancel')">取消</el-button>
              <el-dropdown trigger="click" @command="(cmd: string) => act(cmd)">
                <el-button size="small">
                  更多
                  <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item v-if="canEmergencyApprove" command="approve-emergency">
                      应急批准
                    </el-dropdown-item>
                    <el-dropdown-item :disabled="!canReject" command="reject">驳回</el-dropdown-item>
                    <el-dropdown-item :disabled="!canRollback" command="rollback">回滚</el-dropdown-item>
                    <el-dropdown-item divided :disabled="!canRunTests" command="run-tests">运行测试</el-dropdown-item>
                    <el-dropdown-item :disabled="!canSyncGitHub" command="sync-github">同步 GitHub</el-dropdown-item>
                    <el-dropdown-item :disabled="!canResendNotification" command="resend-notification">
                      重发钉钉
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>

            <el-collapse v-model="detailCollapse" class="detail-sections">
              <el-collapse-item v-if="displayRequirement || iteration.summary || iteration.last_error" name="req">
                <template #title>
                  <span class="collapse-title">需求与摘要</span>
                </template>
                <div v-if="displayRequirement" class="text-block">{{ displayRequirement }}</div>
                <div v-if="iteration.summary" class="text-block text-block--ok">
                  <span class="block-label">摘要</span>{{ iteration.summary }}
                </div>
                <div v-if="iteration.last_error" class="text-block text-block--err">
                  <span class="block-label">错误</span>{{ iteration.last_error }}
                </div>
              </el-collapse-item>
              <el-collapse-item v-if="sampleContext?.similar_samples?.length" name="samples">
                <template #title>
                  <span class="collapse-title">触发样本</span>
                  <el-tag size="small" type="info" class="log-count">
                    {{ sampleContext.similar_samples?.length ?? 0 }}
                  </el-tag>
                </template>
                <p v-if="sampleContext?.sample_classification" class="text-block">
                  分类：{{ sampleContext.sample_classification }}
                  <span v-if="sampleContext.similar_recent_count != null"> · 相似 {{ sampleContext.similar_recent_count }}</span>
                </p>
                <el-table :data="sampleContext?.similar_samples || []" size="small" stripe empty-text="暂无样本">
                  <el-table-column label="时间" width="148">
                    <template #default="{ row }">{{ formatTime(String(row.time || '')) }}</template>
                  </el-table-column>
                  <el-table-column prop="target" label="Target" width="120" show-overflow-tooltip />
                  <el-table-column label="规则" width="72" align="center">
                    <template #default="{ row }">{{ row.local_rule_hit || row.rule_hit ? '命中' : '—' }}</template>
                  </el-table-column>
                  <el-table-column label="AI" width="56" align="center">
                    <template #default="{ row }">{{ row.used_ai ? '是' : '否' }}</template>
                  </el-table-column>
                  <el-table-column label="摘要" min-width="160" show-overflow-tooltip>
                    <template #default="{ row }">{{ row.answer_head || row.root_cause_digest || '—' }}</template>
                  </el-table-column>
                </el-table>
              </el-collapse-item>
              <el-collapse-item name="logs">
                <template #title>
                  <span class="collapse-title">执行日志</span>
                  <el-tag v-if="events.length" size="small" type="info" class="log-count">{{ events.length }}</el-tag>
                </template>
                <el-table :data="events" size="small" stripe empty-text="暂无日志">
                  <el-table-column prop="created_at" label="时间" width="148">
                    <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
                  </el-table-column>
                  <el-table-column prop="message" label="消息" min-width="200" show-overflow-tooltip />
                  <el-table-column prop="event_type" label="类型" width="80" />
                </el-table>
              </el-collapse-item>
            </el-collapse>
            </div>
          </div>
        </el-card>
        </el-col>
      </el-row>
    </div>

    <el-drawer
      v-model="createDrawerOpen"
      title="新建需求"
      size="min(480px, 92vw)"
      destroy-on-close
      @closed="onCreateDrawerClosed"
    >
      <el-form label-position="top" size="default" v-loading="creating" @submit.prevent>
        <el-form-item label="需求标题" required>
          <el-input v-model="requirement.title" placeholder="简要描述目标" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="需求内容" required>
          <el-input
            v-model="requirement.content"
            type="textarea"
            :rows="6"
            :placeholder="requirementPlaceholder"
            maxlength="2000"
            show-word-limit
          />
        </el-form-item>
        <el-form-item label="Topic（可选）">
          <el-input v-model="requirement.topic" placeholder="billing / monitoring" maxlength="80" />
        </el-form-item>
        <p class="drawer-hint">
          提交后自动附加省 Token 规范；按「目标 / 范围 / 验收」填写，勿贴大段日志。
        </p>
        <div class="drawer-actions">
          <el-button
            type="primary"
            :loading="creating"
            :disabled="!settings.enabled || !requirement.content.trim()"
            @click="submitAndClose(true)"
          >
            提交并启动
          </el-button>
          <el-button
            :loading="creating"
            :disabled="!settings.enabled || !requirement.content.trim()"
            @click="submitAndClose(false)"
          >
            仅草稿
          </el-button>
          <el-button @click="createDrawerOpen = false">取消</el-button>
        </div>
        <p v-if="!settings.enabled" class="drawer-warn">请先在「设置」中启用自动迭代。</p>
      </el-form>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, RefreshRight } from '@element-plus/icons-vue'
import { connectSSE } from '../../utils/sseFetch'
import {
  approveAutoIteration,
  cancelAutoIteration,
  createManualAutoIteration,
  getAutoIteration,
  getAutoIterationSamples,
  getAutoIterationSettings,
  listAutoIterations,
  pauseAutoIteration,
  rejectAutoIteration,
  resendAutoIterationNotification,
  resumeAutoIteration,
  rollbackAutoIteration,
  runAutoIterationTests,
  startAutoIteration,
  syncAutoIterationGitHub,
  updateAutoIterationSettings,
  type AutoIteration,
  type AutoIterationEvent,
  type AutoIterationSettings
} from '../../api/autoIterations'

const listLoading = ref(false)
const detailLoading = ref(false)
const detailRefreshing = ref(false)
const settingsLoading = ref(false)
const creating = ref(false)
const createDrawerOpen = ref(false)
const detailCollapse = ref(['req', 'logs'])
const rows = ref<AutoIteration[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const listFilters = reactive({
  keyword: '',
  status: '',
  source: '',
  topic: ''
})
const selectedId = ref('')
const iteration = ref<AutoIteration | null>(null)
const sampleContext = ref<{
  topic?: string
  similar_recent_count?: number
  sample_classification?: string
  similar_samples?: Record<string, unknown>[]
} | null>(null)
const events = ref<AutoIterationEvent[]>([])
const sseConnected = ref(false)
let sseAbort: AbortController | null = null
let sseRetryTimer: ReturnType<typeof setTimeout> | null = null
let sseRetryAttempt = 0
let detailPollTimer: ReturnType<typeof setInterval> | null = null
let settingsSaveTimer: ReturnType<typeof setTimeout> | null = null
const seenEventIds = new Set<string>()
const TERMINAL_STATUSES = new Set(['completed', 'approved', 'rejected', 'cancelled', 'failed'])
const ACTIVE_STATUSES = new Set(['pending', 'running', 'paused', 'awaiting_approval'])

const settings = reactive<AutoIterationSettings>({
  enabled: false,
  max_concurrent: 2,
  high_risk_requires_approval: true,
  auto_dispatch_enabled: true,
  low_risk_auto_deploy_enabled: false,
  github_sync_enabled: true,
  dingtalk_notify_enabled: true,
  has_dingtalk_webhook: false
})

const requirement = reactive({
  title: '',
  content: '',
  topic: ''
})

const requirementPlaceholder = `目标: （一句）
范围: （模块或路径）
验收:
- 
- 
不做: （可选）`

const STATUS_LABELS: Record<string, string> = {
  draft: '草稿',
  pending: '待处理',
  running: '开发中',
  paused: '已暂停',
  awaiting_approval: '待审批',
  approved: '已批准',
  rejected: '已驳回',
  cancelled: '已取消',
  completed: '已完成',
  failed: '失败',
  rollback_required: '待回滚',
  rolled_back: '已回滚'
}

const STATUS_FILTER_OPTIONS = Object.entries(STATUS_LABELS).map(([value, label]) => ({ value, label }))

const SOURCE_LABELS: Record<string, string> = {
  manual: '手动',
  cli_feedback: 'CLI 反馈',
  cli_capability_gap: '能力缺口',
  skill_refine: '技能精炼',
  rule_candidate: '规则候选',
  diagnosis_insufficient: '诊断不足',
  ai_cost_reduction: 'AI 成本优化'
}

const SOURCE_FILTER_OPTIONS = Object.entries(SOURCE_LABELS).map(([value, label]) => ({ value, label }))

const statusLabel = (s: string) => STATUS_LABELS[s] || s
const sourceLabel = (s: string) => SOURCE_LABELS[s] || s

const formatTime = (t?: string) => {
  if (!t) return '—'
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return t
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const rowClassName = ({ row }: { row: AutoIteration }) =>
  row.id === selectedId.value ? 'list-row--active' : ''

const statusTagType = (s: string): 'success' | 'warning' | 'danger' | 'info' | 'primary' => {
  if (s === 'completed' || s === 'approved') return 'success'
  if (s === 'running' || s === 'pending') return 'primary'
  if (s === 'awaiting_approval') return 'warning'
  if (s === 'failed' || s === 'rejected' || s === 'cancelled') return 'danger'
  return 'info'
}

/** Strip inlined agent spec from command; prefer user-facing description. */
const stripAgentSpec = (cmd: string) => {
  const t = cmd.trim()
  if (!t) return ''
  const needIdx = t.indexOf('## 需求')
  if (needIdx >= 0) return t.slice(needIdx).trim()
  if (t.startsWith('【省 Token')) {
    const parts = t.split('\n\n')
    return parts.length > 1 ? parts.slice(1).join('\n\n').trim() : ''
  }
  return t
}

const displayRequirement = computed(() => {
  const i = iteration.value
  if (!i) return ''
  const desc = i.description?.trim()
  if (desc) return desc
  return stripAgentSpec(i.command || '')
})

const phaseLabel = computed(() => {
  const s = iteration.value?.status
  if (!s) return ''
  if (s === 'draft') return '草稿'
  if (s === 'pending') return '等待 Worker'
  if (s === 'running') return '开发中'
  if (s === 'paused') return '已暂停'
  if (s === 'awaiting_approval') return '待审批'
  if (s === 'approved' || s === 'completed') return '已上线'
  if (s === 'failed') return '失败'
  if (s === 'rejected') return '已驳回'
  if (s === 'cancelled') return '已取消'
  return ''
})

const statusHint = computed((): { type: 'info' | 'warning' | 'success' | 'error'; title: string } | null => {
  const s = iteration.value?.status
  if (!s) return null
  if (s === 'draft') {
    return { type: 'warning', title: '草稿未入队 — 点击「启动开发」后 Worker 才会拉取' }
  }
  if (s === 'pending') {
    return { type: 'info', title: '已在队列，等待本机 code-agent-worker 拉取' }
  }
  if (s === 'running' && !iteration.value?.assigned_agent_id) {
    return { type: 'warning', title: '状态为开发中但尚未分配 Worker，可尝试「重新入队」' }
  }
  return null
})

const canStart = computed(() => {
  const s = iteration.value?.status
  return s === 'draft' || s === 'paused'
})
const canRequeue = computed(() => {
  const s = iteration.value?.status
  return s === 'pending' || (s === 'running' && !iteration.value?.assigned_agent_id)
})
const canPause = computed(() => iteration.value?.status === 'running')
const canResume = computed(() => iteration.value?.status === 'paused')
const canApprove = computed(() => iteration.value?.status === 'awaiting_approval')
const canEmergencyApprove = computed(() => iteration.value?.status === 'pending')
const canReject = computed(() => iteration.value?.status === 'awaiting_approval')
const canCancel = computed(() => {
  const s = iteration.value?.status
  return !!s && !TERMINAL_STATUSES.has(s)
})
const canRollback = computed(() => {
  const s = iteration.value?.status
  return s === 'approved' || s === 'completed' || s === 'failed'
})
const canRunTests = computed(() => {
  const s = iteration.value?.status
  return !!s && s !== 'draft' && !TERMINAL_STATUSES.has(s)
})
const canSyncGitHub = computed(() => canRunTests.value)
const canResendNotification = computed(() => !!iteration.value?.id)
const needsDetailPoll = computed(() => ACTIVE_STATUSES.has(iteration.value?.status || ''))

const mergeEvent = (ev: AutoIterationEvent) => {
  if (!ev?.id || seenEventIds.has(ev.id)) return
  seenEventIds.add(ev.id)
  events.value = [...events.value, ev].sort(
    (a, b) => String(a.created_at).localeCompare(String(b.created_at))
  )
}

const lastEventIdForSSE = () => {
  if (!events.value.length) return ''
  const last = events.value[events.value.length - 1]
  return last?.id || ''
}

const applyIterationSnapshot = (row: AutoIteration) => {
  if (!row?.id || row.id !== selectedId.value) return
  iteration.value = row
}

const stopSSE = (opts?: { resetRetry?: boolean }) => {
  if (sseRetryTimer) {
    clearTimeout(sseRetryTimer)
    sseRetryTimer = null
  }
  sseAbort?.abort()
  sseAbort = null
  sseConnected.value = false
  if (opts?.resetRetry !== false) {
    sseRetryAttempt = 0
  }
}

const scheduleSSEReconnect = () => {
  if (!selectedId.value || sseRetryTimer) return
  const delay = Math.min(30000, 1000 * 2 ** sseRetryAttempt)
  sseRetryAttempt += 1
  sseRetryTimer = setTimeout(() => {
    sseRetryTimer = null
    if (selectedId.value) startSSE()
  }, delay)
}

const startSSE = () => {
  stopSSE({ resetRetry: false })
  if (!selectedId.value) return
  const after = lastEventIdForSSE()
  const qs = after ? `?after_id=${encodeURIComponent(after)}` : ''
  sseAbort = connectSSE(
    `/api/admin/auto-iterations/${encodeURIComponent(selectedId.value)}/events/stream${qs}`,
    {
      onEvent: (name, data) => {
        sseRetryAttempt = 0
        sseConnected.value = true
        try {
          if (name === 'status') {
            applyIterationSnapshot(JSON.parse(data) as AutoIteration)
            return
          }
          if (name === 'log') {
            mergeEvent(JSON.parse(data) as AutoIterationEvent)
          }
        } catch {
          /* ignore */
        }
      },
      onError: () => {
        sseConnected.value = false
        scheduleSSEReconnect()
      },
      onClose: () => {
        sseConnected.value = false
        scheduleSSEReconnect()
      }
    }
  )
}

const stopDetailPoll = () => {
  if (detailPollTimer) {
    clearInterval(detailPollTimer)
    detailPollTimer = null
  }
}

const startDetailPoll = () => {
  stopDetailPoll()
  if (!needsDetailPoll.value || !selectedId.value) return
  detailPollTimer = setInterval(() => {
    if (selectedId.value && needsDetailPoll.value) {
      void refreshDetailQuiet()
    }
  }, 12000)
}

const refreshDetailQuiet = async () => {
  if (!selectedId.value) return
  try {
    const data = await getAutoIteration(selectedId.value)
    iteration.value = data.iteration
    const incoming = data.events || []
    for (const ev of incoming) {
      mergeEvent(ev)
    }
  } catch {
    /* ignore */
  }
}

const loadSettings = async () => {
  settingsLoading.value = true
  try {
    const data = await getAutoIterationSettings()
    Object.assign(settings, data.settings)
  } finally {
    settingsLoading.value = false
  }
}

const saveSettings = async () => {
  try {
    const data = await updateAutoIterationSettings({
      enabled: settings.enabled,
      max_concurrent: settings.max_concurrent,
      high_risk_requires_approval: settings.high_risk_requires_approval,
      auto_dispatch_enabled: settings.auto_dispatch_enabled,
      low_risk_auto_deploy_enabled: settings.low_risk_auto_deploy_enabled,
      github_sync_enabled: settings.github_sync_enabled,
      dingtalk_notify_enabled: settings.dingtalk_notify_enabled
    })
    Object.assign(settings, data.settings)
    ElMessage.success('已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

const scheduleSaveSettings = () => {
  if (settingsSaveTimer) clearTimeout(settingsSaveTimer)
  settingsSaveTimer = setTimeout(() => {
    settingsSaveTimer = null
    void saveSettings()
  }, 400)
}

const listQueryParams = () => {
  const p: {
    page: number
    page_size: number
    status?: string
    topic?: string
    source?: string
    keyword?: string
  } = { page: page.value, page_size: pageSize.value }
  const status = listFilters.status.trim()
  const topic = listFilters.topic.trim()
  const source = listFilters.source.trim()
  const keyword = listFilters.keyword.trim()
  if (status) p.status = status
  if (topic) p.topic = topic
  if (source) p.source = source
  if (keyword) p.keyword = keyword
  return p
}

const applyListFilters = () => {
  page.value = 1
  void loadList()
}

const resetListFilters = () => {
  listFilters.keyword = ''
  listFilters.status = ''
  listFilters.source = ''
  listFilters.topic = ''
  applyListFilters()
}

const taskIdFromUrl = () => new URLSearchParams(window.location.search).get('id') || ''

const syncTaskIdToUrl = (id: string) => {
  const url = new URL(window.location.href)
  if (id) url.searchParams.set('id', id)
  else url.searchParams.delete('id')
  const next = url.pathname + url.search + url.hash
  const cur = window.location.pathname + window.location.search + window.location.hash
  if (next !== cur) window.history.replaceState(window.history.state, '', next)
}

const clearDetail = () => {
  iteration.value = null
  events.value = []
  seenEventIds.clear()
  stopSSE()
  stopDetailPoll()
}

const applyListRowPreview = (row: AutoIteration) => {
  iteration.value = { ...row }
}

const loadList = async (opts?: { silent?: boolean }) => {
  if (!opts?.silent) listLoading.value = true
  try {
    const data = await listAutoIterations(listQueryParams())
    rows.value = data.list || []
    total.value = data.total || 0
    if (selectedId.value && !rows.value.some((r) => r.id === selectedId.value)) {
      selectedId.value = ''
      syncTaskIdToUrl('')
      clearDetail()
    } else if (selectedId.value && iteration.value) {
      const hit = rows.value.find((r) => r.id === selectedId.value)
      if (hit) applyListRowPreview(hit)
    }
  } catch {
    rows.value = []
    total.value = 0
  } finally {
    listLoading.value = false
  }
}

const refreshList = () => loadList()

const loadDetail = async (opts?: { initial?: boolean }) => {
  if (!selectedId.value) return
  const initial = opts?.initial ?? !iteration.value
  if (initial) detailLoading.value = true
  else detailRefreshing.value = true
  try {
    const data = await getAutoIteration(selectedId.value)
    iteration.value = data.iteration
    events.value = data.events || []
    try {
      sampleContext.value = await getAutoIterationSamples(selectedId.value)
    } catch {
      sampleContext.value = null
    }
    seenEventIds.clear()
    for (const ev of events.value) {
      if (ev.id) seenEventIds.add(ev.id)
    }
    if (!displayRequirement.value && !iteration.value?.summary && !iteration.value?.last_error) {
      detailCollapse.value = sampleContext.value?.similar_samples?.length ? ['samples', 'logs'] : ['logs']
    } else {
      detailCollapse.value = sampleContext.value?.similar_samples?.length ? ['req', 'samples', 'logs'] : ['req', 'logs']
    }
    startSSE()
    startDetailPoll()
  } catch {
    iteration.value = null
    events.value = []
    stopSSE()
    stopDetailPoll()
  } finally {
    detailLoading.value = false
    detailRefreshing.value = false
  }
}

const selectRow = (row: AutoIteration) => {
  if (!row?.id || row.id === selectedId.value) return
  stopSSE()
  stopDetailPoll()
  selectedId.value = row.id
  syncTaskIdToUrl(row.id)
  applyListRowPreview(row)
  events.value = []
  seenEventIds.clear()
  void loadDetail()
}

const act = async (name: string) => {
  const i = selectedId.value
  if (!i) return
  const statusBefore = iteration.value?.status
  try {
    switch (name) {
      case 'start':
        await startAutoIteration(i)
        break
      case 'pause':
        await pauseAutoIteration(i)
        break
      case 'resume':
        await resumeAutoIteration(i)
        break
      case 'cancel': {
        try {
          await ElMessageBox.confirm('确定取消该任务？', '取消任务', {
            confirmButtonText: '取消任务',
            cancelButtonText: '返回',
            type: 'warning'
          })
        } catch {
          return
        }
        await cancelAutoIteration(i)
        break
      }
      case 'approve': {
        let notes = ''
        try {
          const r = await ElMessageBox.prompt('审批备注（可选）', '批准上线', {
            confirmButtonText: '批准',
            cancelButtonText: '取消'
          })
          notes = r.value || ''
        } catch {
          return
        }
        await approveAutoIteration(i, notes, false)
        break
      }
      case 'approve-emergency': {
        try {
          await ElMessageBox.confirm(
            '该任务尚未完成开发/测试。应急批准将直接标记为已批准并跳过 Worker 流程，仅用于紧急上线。确定继续？',
            '应急批准',
            { confirmButtonText: '应急批准', cancelButtonText: '返回', type: 'warning' }
          )
        } catch {
          return
        }
        let notes = ''
        try {
          const r = await ElMessageBox.prompt('应急批准备注（建议填写原因）', '应急批准', {
            confirmButtonText: '确认',
            cancelButtonText: '取消'
          })
          notes = r.value || ''
        } catch {
          return
        }
        await approveAutoIteration(i, notes, true)
        break
      }
      case 'reject': {
        let reason = ''
        try {
          const r = await ElMessageBox.prompt('驳回原因', '驳回', {
            confirmButtonText: '驳回',
            cancelButtonText: '取消'
          })
          reason = r.value || ''
        } catch {
          return
        }
        await rejectAutoIteration(i, reason)
        break
      }
      case 'rollback': {
        let reason = ''
        try {
          const r = await ElMessageBox.prompt('回滚原因（可选）', '回滚', {
            confirmButtonText: '确认',
            cancelButtonText: '取消'
          })
          reason = r.value || ''
        } catch {
          return
        }
        await rollbackAutoIteration(i, reason)
        break
      }
      case 'run-tests':
        await runAutoIterationTests(i)
        break
      case 'sync-github':
        await syncAutoIterationGitHub(i)
        break
      case 'resend-notification':
        await resendAutoIterationNotification(i)
        break
    }
    if (name === 'start') {
      if (statusBefore === 'pending' || statusBefore === 'running') {
        ElMessage.success('任务已在队列或开发中，Worker 将继续处理')
      } else {
        ElMessage.success('已加入队列，等待本机 Worker 拉取')
      }
    } else {
      ElMessage.success('操作成功')
    }
    await Promise.all([loadList({ silent: true }), loadDetail()])
  } catch (e) {
    const msg = e instanceof Error ? e.message : ''
    if (!msg || msg === 'cancel' || msg.includes('取消')) return
  }
}

const submitRequirement = async (autoStart: boolean) => {
  const content = requirement.content.trim()
  if (!content) {
    ElMessage.warning('请填写需求内容')
    return false
  }
  creating.value = true
  try {
    const title = requirement.title.trim() || content.slice(0, 40) + (content.length > 40 ? '…' : '')
    const data = await createManualAutoIteration({
      title,
      description: content,
      command: content,
      topic: requirement.topic.trim(),
      auto_start: autoStart
    })
    requirement.title = ''
    requirement.content = ''
    requirement.topic = ''
    const st = data.iteration?.status
    ElMessage.success(
      autoStart
        ? st === 'pending' || st === 'running'
          ? '已提交，Worker 将拉取并执行'
          : '已提交'
        : '草稿已保存'
    )
    await loadList()
    if (data.iteration?.id) {
      selectedId.value = data.iteration.id
      syncTaskIdToUrl(data.iteration.id)
      applyListRowPreview(data.iteration)
      await loadDetail({ initial: true })
    }
    return true
  } catch {
    ElMessage.error('提交失败')
    return false
  } finally {
    creating.value = false
  }
}

const submitAndClose = async (autoStart: boolean) => {
  const ok = await submitRequirement(autoStart)
  if (ok) createDrawerOpen.value = false
}

const onCreateDrawerClosed = () => {
  if (!creating.value) {
    requirement.title = ''
    requirement.content = ''
    requirement.topic = ''
  }
}

const onPopState = () => {
  const qid = taskIdFromUrl()
  if (qid === selectedId.value) return
  stopSSE()
  stopDetailPoll()
  selectedId.value = qid
  if (!qid) {
    clearDetail()
    return
  }
  const hit = rows.value.find((r) => r.id === qid)
  if (hit) applyListRowPreview(hit)
  else iteration.value = null
  events.value = []
  seenEventIds.clear()
  void loadDetail({ initial: !iteration.value })
}

watch(needsDetailPoll, (active) => {
  if (active) startDetailPoll()
  else stopDetailPoll()
})

onMounted(async () => {
  const qid = taskIdFromUrl()
  if (qid) selectedId.value = qid
  window.addEventListener('popstate', onPopState)
  await loadSettings()
  await loadList()
  if (selectedId.value) {
    const hit = rows.value.find((r) => r.id === selectedId.value)
    if (hit) applyListRowPreview(hit)
    await loadDetail({ initial: true })
  }
})

onUnmounted(() => {
  window.removeEventListener('popstate', onPopState)
  stopSSE()
  stopDetailPoll()
  if (settingsSaveTimer) clearTimeout(settingsSaveTimer)
})
</script>

<style scoped>
.auto-iterations {
  gap: 10px;
}
.page-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
  min-height: 32px;
}
.workbench {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.page-toolbar__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.25;
  white-space: nowrap;
}
.page-toolbar__actions {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.settings-popover .el-form-item {
  margin-bottom: 12px;
}
.settings-popover .el-form-item:last-child {
  margin-bottom: 0;
}
.settings-warn {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--el-color-warning);
}
.main-row {
  flex: 1;
  min-height: 0;
  height: 100%;
  align-items: stretch;
}
.main-row :deep(.el-row) {
  height: 100%;
}
.pane-col {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.pane-card {
  flex: 1;
  min-height: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
}
.pane-card :deep(.el-card__header) {
  flex-shrink: 0;
}
.pane-card :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.list-filters,
.pager,
.action-bar,
.status-hint,
.meta-chips {
  flex-shrink: 0;
}
.pane-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.detail-body {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.detail-empty {
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}
.detail-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
}
.list-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
  align-items: center;
}
.filter-grow {
  flex: 1 1 160px;
  min-width: 120px;
}
.filter-status {
  width: 108px;
  flex-shrink: 0;
}
.pager {
  margin-top: 8px;
  display: flex;
  justify-content: flex-end;
}
.detail-card :deep(.el-card__header) {
  padding: 10px 14px;
}
.detail-header {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.detail-title {
  font-weight: 600;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.detail-header-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.phase-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.status-hint {
  margin-bottom: 10px;
}
.status-hint :deep(.el-alert__content) {
  padding: 0;
}
.meta-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 10px;
}
.meta-chip {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.action-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
  align-items: center;
}
.detail-sections {
  border: none;
}
.detail-sections :deep(.el-collapse-item__header) {
  border-bottom: 1px solid var(--el-border-color-lighter);
  height: 40px;
  font-size: 13px;
}
.detail-sections :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}
.collapse-title {
  font-weight: 600;
}
.log-count {
  margin-left: 8px;
}
.text-block {
  padding: 10px 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 6px;
  font-size: 13px;
  line-height: 1.55;
  white-space: pre-wrap;
  word-break: break-word;
  margin-bottom: 8px;
}
.text-block:last-child {
  margin-bottom: 0;
}
.text-block--ok {
  background: var(--el-color-success-light-9);
}
.text-block--err {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger);
}
.block-label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  margin-bottom: 4px;
  opacity: 0.85;
}
.drawer-hint {
  margin: 0 0 16px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.drawer-warn {
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-color-warning);
}
.drawer-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
:deep(.list-row--active > td) {
  background-color: var(--el-color-primary-light-9) !important;
}
</style>
