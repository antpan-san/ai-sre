<template>
  <div class="auto-iterations page-shell page-shell--crud-wide">
    <div class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">自动迭代</h2>
      </div>
      <div class="page-head-actions">
        <el-button :loading="loading" @click="loadAll">刷新</el-button>
        <el-button type="primary" @click="openManual">手动创建</el-button>
      </div>
    </div>

    <el-card shadow="never" class="settings-card" v-loading="settingsLoading">
      <template #header>全局设置</template>
      <el-form label-width="120px" size="small" inline>
        <el-form-item label="启用">
          <el-switch v-model="settings.enabled" @change="saveSettings" />
        </el-form-item>
        <el-form-item label="最大并发">
          <el-input-number v-model="settings.max_concurrent" :min="1" :max="20" @change="saveSettings" />
        </el-form-item>
        <el-form-item label="高风险需审批">
          <el-switch v-model="settings.high_risk_requires_approval" @change="saveSettings" />
        </el-form-item>
        <el-form-item label="钉钉">
          <el-tag :type="settings.has_dingtalk_webhook ? 'success' : 'info'" size="small">
            {{ settings.has_dingtalk_webhook ? '已配置' : '未配置' }}
          </el-tag>
        </el-form-item>
      </el-form>
    </el-card>

    <el-row :gutter="16" class="main-row">
      <el-col :xs="24" :lg="10">
        <el-card shadow="never" v-loading="loading" class="list-card">
          <template #header>任务列表</template>
          <el-table
            :data="rows"
            stripe
            border
            size="small"
            highlight-current-row
            empty-text="暂无任务"
            @row-click="selectRow"
          >
            <el-table-column prop="title" label="标题" min-width="140" show-overflow-tooltip />
            <el-table-column prop="status" label="状态" width="110" />
            <el-table-column prop="risk_level" label="风险" width="72" />
            <el-table-column prop="topic" label="Topic" width="88" show-overflow-tooltip />
          </el-table>
          <div class="pager">
            <el-pagination
              v-model:current-page="page"
              v-model:page-size="pageSize"
              :total="total"
              layout="total, prev, pager, next"
              small
              @current-change="loadList"
            />
          </div>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="14">
        <el-card shadow="never" class="detail-card" v-loading="detailLoading">
          <template #header>
            <span>{{ iteration?.title || '任务详情' }}</span>
            <el-tag v-if="sseConnected" type="success" size="small" class="sse-tag">实时</el-tag>
          </template>

          <el-empty v-if="!selectedId" description="请选择左侧任务" />
          <template v-else-if="iteration">
            <el-descriptions :column="2" border size="small" class="meta">
              <el-descriptions-item label="状态">{{ iteration.status }}</el-descriptions-item>
              <el-descriptions-item label="风险">{{ iteration.risk_level }}</el-descriptions-item>
              <el-descriptions-item label="Topic">{{ iteration.topic || '—' }}</el-descriptions-item>
              <el-descriptions-item label="来源">{{ iteration.source }}</el-descriptions-item>
              <el-descriptions-item label="创建人">{{ iteration.created_by || '—' }}</el-descriptions-item>
              <el-descriptions-item label="审批人">{{ iteration.approved_by || '—' }}</el-descriptions-item>
            </el-descriptions>

            <div class="actions">
              <el-button type="primary" size="small" @click="act('start')">开始</el-button>
              <el-button size="small" @click="act('pause')">暂停</el-button>
              <el-button size="small" @click="act('resume')">继续</el-button>
              <el-button size="small" @click="act('cancel')">取消</el-button>
              <el-button type="success" size="small" @click="act('approve')">批准</el-button>
              <el-button type="danger" size="small" @click="act('reject')">驳回</el-button>
              <el-button type="warning" size="small" @click="act('rollback')">回滚</el-button>
              <el-button size="small" @click="act('run-tests')">测试</el-button>
              <el-button size="small" @click="act('sync-github')">GitHub</el-button>
              <el-button size="small" @click="act('resend-notification')">钉钉</el-button>
            </div>

            <el-table :data="events" size="small" stripe border empty-text="暂无日志" max-height="360">
              <el-table-column prop="created_at" label="时间" width="160" />
              <el-table-column prop="event_type" label="类型" width="90" />
              <el-table-column prop="actor_name" label="操作者" width="90" />
              <el-table-column prop="message" label="消息" min-width="200" show-overflow-tooltip />
            </el-table>
          </template>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog v-model="manualOpen" title="手动创建" width="480px">
      <el-form label-width="72px">
        <el-form-item label="标题"><el-input v-model="manual.title" /></el-form-item>
        <el-form-item label="Topic"><el-input v-model="manual.topic" /></el-form-item>
        <el-form-item label="说明"><el-input v-model="manual.description" type="textarea" :rows="3" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="manualOpen = false">取消</el-button>
        <el-button type="primary" :loading="creating" @click="submitManual">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { connectSSE } from '../../utils/sseFetch'
import {
  approveAutoIteration,
  cancelAutoIteration,
  createManualAutoIteration,
  getAutoIteration,
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

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const detailLoading = ref(false)
const settingsLoading = ref(false)
const creating = ref(false)
const rows = ref<AutoIteration[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const selectedId = ref('')
const iteration = ref<AutoIteration | null>(null)
const events = ref<AutoIterationEvent[]>([])
const sseConnected = ref(false)
let sseAbort: AbortController | null = null
const seenEventIds = new Set<string>()

const settings = reactive<AutoIterationSettings>({
  enabled: false,
  max_concurrent: 2,
  high_risk_requires_approval: true,
  has_dingtalk_webhook: false
})
const manualOpen = ref(false)
const manual = reactive({ title: '', topic: '', description: '' })

const mergeEvent = (ev: AutoIterationEvent) => {
  if (!ev?.id || seenEventIds.has(ev.id)) return
  seenEventIds.add(ev.id)
  events.value = [...events.value, ev].sort(
    (a, b) => String(a.created_at).localeCompare(String(b.created_at))
  )
}

const stopSSE = () => {
  sseAbort?.abort()
  sseAbort = null
  sseConnected.value = false
}

const startSSE = () => {
  stopSSE()
  if (!selectedId.value) return
  sseAbort = connectSSE(
    `/api/admin/auto-iterations/${encodeURIComponent(selectedId.value)}/events/stream`,
    {
      onEvent: (name, data) => {
        if (name !== 'log') return
        try {
          mergeEvent(JSON.parse(data) as AutoIterationEvent)
          sseConnected.value = true
        } catch {
          /* ignore */
        }
      },
      onError: () => {
        sseConnected.value = false
      }
    }
  )
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
      high_risk_requires_approval: settings.high_risk_requires_approval
    })
    Object.assign(settings, data.settings)
    ElMessage.success('已保存')
  } catch {
    ElMessage.error('保存失败')
  }
}

const loadList = async () => {
  loading.value = true
  try {
    const data = await listAutoIterations({ page: page.value, page_size: pageSize.value })
    rows.value = data.list || []
    total.value = data.total || 0
    if (selectedId.value && !rows.value.some((r) => r.id === selectedId.value)) {
      selectedId.value = ''
      iteration.value = null
      events.value = []
      stopSSE()
    }
  } catch {
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const loadDetail = async () => {
  if (!selectedId.value) return
  detailLoading.value = true
  try {
    const data = await getAutoIteration(selectedId.value)
    iteration.value = data.iteration
    events.value = data.events || []
    seenEventIds.clear()
    for (const ev of events.value) {
      if (ev.id) seenEventIds.add(ev.id)
    }
    startSSE()
  } catch {
    iteration.value = null
    events.value = []
    stopSSE()
  } finally {
    detailLoading.value = false
  }
}

const selectRow = (row: AutoIteration) => {
  if (!row?.id) return
  selectedId.value = row.id
  router.replace({ query: { ...route.query, id: row.id } })
  void loadDetail()
}

const act = async (name: string) => {
  const i = selectedId.value
  if (!i) return
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
      case 'cancel':
        await cancelAutoIteration(i)
        break
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
        await approveAutoIteration(i, notes)
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
    ElMessage.success('操作成功')
    await Promise.all([loadList(), loadDetail()])
  } catch {
    ElMessage.error('操作失败')
  }
}

const openManual = () => {
  manual.title = ''
  manual.topic = ''
  manual.description = ''
  manualOpen.value = true
}

const submitManual = async () => {
  creating.value = true
  try {
    const data = await createManualAutoIteration({ ...manual })
    manualOpen.value = false
    ElMessage.success('已创建')
    await loadList()
    if (data.iteration?.id) {
      selectedId.value = data.iteration.id
      router.replace({ query: { id: data.iteration.id } })
      await loadDetail()
    }
  } catch {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

const loadAll = () => Promise.all([loadSettings(), loadList()])

watch(
  () => route.query.id,
  (id) => {
    const s = String(id || '')
    if (s && s !== selectedId.value) {
      selectedId.value = s
      void loadDetail()
    }
  }
)

onMounted(async () => {
  const qid = String(route.query.id || '')
  if (qid) selectedId.value = qid
  await loadAll()
  if (selectedId.value) await loadDetail()
})

onUnmounted(() => stopSSE())
</script>

<style scoped>
.settings-card {
  margin-bottom: 16px;
}
.main-row {
  align-items: stretch;
}
.list-card,
.detail-card {
  min-height: 480px;
}
.pager {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}
.page-head-actions {
  display: flex;
  gap: 8px;
}
.meta {
  margin-bottom: 12px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 12px;
}
.sse-tag {
  margin-left: 8px;
}
</style>
