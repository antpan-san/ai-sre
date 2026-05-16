<template>
  <div class="auto-iteration-detail page-shell page-shell--crud-wide" v-loading="loading">
    <div class="page-head">
      <el-button link type="primary" @click="$router.push('/admin/auto-iterations')">返回列表</el-button>
      <h2 class="page-title">{{ iteration?.title || '迭代详情' }}</h2>
    </div>

    <template v-if="iteration">
      <el-descriptions :column="2" border size="small" class="meta">
        <el-descriptions-item label="状态">{{ iteration.status }}</el-descriptions-item>
        <el-descriptions-item label="风险">{{ iteration.risk_level }}</el-descriptions-item>
        <el-descriptions-item label="Topic">{{ iteration.topic || '—' }}</el-descriptions-item>
        <el-descriptions-item label="来源">{{ iteration.source }}</el-descriptions-item>
        <el-descriptions-item label="创建人">{{ iteration.created_by || '—' }}</el-descriptions-item>
        <el-descriptions-item label="审批人">{{ iteration.approved_by || '—' }}</el-descriptions-item>
      </el-descriptions>

      <div class="actions">
        <el-button type="primary" @click="act('start')">开始</el-button>
        <el-button @click="act('pause')">暂停</el-button>
        <el-button @click="act('resume')">继续</el-button>
        <el-button @click="act('cancel')">取消</el-button>
        <el-button type="success" @click="act('approve')">批准上线</el-button>
        <el-button type="danger" @click="act('reject')">驳回</el-button>
        <el-button type="warning" @click="act('rollback')">回滚</el-button>
        <el-button @click="act('run-tests')">重新测试</el-button>
        <el-button @click="act('sync-github')">GitHub 同步</el-button>
        <el-button @click="act('resend-notification')">重发钉钉</el-button>
      </div>

      <el-card shadow="never" class="log-card">
        <template #header>
          <span>事件日志</span>
          <el-tag v-if="sseConnected" type="success" size="small" class="sse-tag">实时</el-tag>
          <el-button size="small" @click="loadDetail">刷新</el-button>
        </template>
        <el-table :data="events" size="small" stripe border empty-text="暂无日志">
          <el-table-column prop="created_at" label="时间" width="170" />
          <el-table-column prop="event_type" label="类型" width="100" />
          <el-table-column prop="actor_name" label="操作者" width="100" />
          <el-table-column prop="message" label="消息" min-width="280" show-overflow-tooltip />
        </el-table>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { connectSSE } from '../../utils/sseFetch'
import {
  approveAutoIteration,
  cancelAutoIteration,
  getAutoIteration,
  pauseAutoIteration,
  rejectAutoIteration,
  resendAutoIterationNotification,
  resumeAutoIteration,
  rollbackAutoIteration,
  runAutoIterationTests,
  startAutoIteration,
  syncAutoIterationGitHub,
  type AutoIteration,
  type AutoIterationEvent
} from '../../api/autoIterations'

const route = useRoute()
const loading = ref(false)
const iteration = ref<AutoIteration | null>(null)
const events = ref<AutoIterationEvent[]>([])
const sseConnected = ref(false)
let sseAbort: AbortController | null = null
const seenEventIds = new Set<string>()

const id = () => String(route.params.id || '')

const mergeEvent = (ev: AutoIterationEvent) => {
  if (!ev?.id || seenEventIds.has(ev.id)) return
  seenEventIds.add(ev.id)
  events.value = [...events.value, ev].sort(
    (a, b) => String(a.created_at).localeCompare(String(b.created_at))
  )
}

const startSSE = () => {
  stopSSE()
  const taskId = id()
  if (!taskId) return
  sseAbort = connectSSE(`/api/admin/auto-iterations/${encodeURIComponent(taskId)}/events/stream`, {
    onEvent: (name, data) => {
      if (name !== 'log') return
      try {
        mergeEvent(JSON.parse(data) as AutoIterationEvent)
        sseConnected.value = true
      } catch {
        /* ignore malformed chunk */
      }
    },
    onError: () => {
      sseConnected.value = false
    }
  })
}

const stopSSE = () => {
  sseAbort?.abort()
  sseAbort = null
  sseConnected.value = false
}

const loadDetail = async () => {
  loading.value = true
  try {
    const data = await getAutoIteration(id())
    iteration.value = data.iteration
    events.value = data.events || []
    seenEventIds.clear()
    for (const ev of events.value) {
      if (ev.id) seenEventIds.add(ev.id)
    }
  } catch {
    iteration.value = null
    events.value = []
  } finally {
    loading.value = false
  }
}

const act = async (name: string) => {
  const i = id()
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
          const r = await ElMessageBox.prompt('审批备注（可选）', '批准上线', { confirmButtonText: '批准', cancelButtonText: '取消' })
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
          const r = await ElMessageBox.prompt('驳回原因', '驳回', { confirmButtonText: '驳回', cancelButtonText: '取消' })
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
          const r = await ElMessageBox.prompt('回滚原因（可选）', '回滚', { confirmButtonText: '确认', cancelButtonText: '取消' })
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
    await loadDetail()
  } catch {
    ElMessage.error('操作失败')
  }
}

onMounted(async () => {
  await loadDetail()
  startSSE()
})

onUnmounted(() => stopSSE())
</script>

<style scoped>
.meta {
  margin-bottom: 16px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 16px;
}
.log-card {
  margin-top: 8px;
}
.sse-tag {
  margin-left: 8px;
  vertical-align: middle;
}
</style>
