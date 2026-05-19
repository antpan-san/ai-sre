<template>
  <div class="client-exec-detail page-shell" v-loading="loading">
    <header class="page-head">
      <el-button link type="primary" @click="goBack">← 返回列表</el-button>
      <h2 class="page-title">执行复盘</h2>
    </header>

    <template v-if="detail">
      <el-card shadow="never" class="block">
        <template #header><span>执行结论</span></template>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="命令" :span="2">{{ rec.command }}</el-descriptions-item>
          <el-descriptions-item label="目标">{{ rec.target_host || rec.resource_name || meta.diagnosis_target || '—' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="statusTag(rec.status)" size="small">{{ rec.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="耗时">{{ formatDuration(rec.duration_ms) }}</el-descriptions-item>
          <el-descriptions-item label="CLI 版本">{{ meta.version || '—' }}</el-descriptions-item>
          <el-descriptions-item label="用户">{{ rec.trigger_user || rec.created_by || '—' }}</el-descriptions-item>
          <el-descriptions-item label="机器">{{ rec.target_host || meta.hostname || '—' }}</el-descriptions-item>
          <el-descriptions-item label="Topic">{{ meta.topic || rec.category || '—' }}</el-descriptions-item>
          <el-descriptions-item label="根因摘要" :span="2">{{ meta.root_cause || meta.summary || rec.stdout_summary || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-tag v-if="detail.legacy_kind" type="info" class="legacy-tag">{{ legacyLabel(detail.legacy_kind) }}</el-tag>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header><span>诊断证据</span></template>
        <p class="evidence-line">
          证据完整度：
          <el-tag :type="evidenceTag(meta.evidence_completeness)" size="small">{{ meta.evidence_completeness || '未知' }}</el-tag>
        </p>
        <el-collapse v-if="rec.stdout_summary || rec.stderr_summary">
          <el-collapse-item title="输出摘要" name="out">
            <pre class="mono">{{ rec.stdout_summary || '—' }}</pre>
            <pre v-if="rec.stderr_summary" class="mono err">{{ rec.stderr_summary }}</pre>
          </el-collapse-item>
        </el-collapse>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header><span>执行链路</span></template>
        <el-timeline>
          <el-timeline-item v-for="(t, i) in detail.timeline || []" :key="i" :timestamp="formatTime(t.time)">
            <strong>{{ t.phase }}</strong> — {{ t.message }}
          </el-timeline-item>
        </el-timeline>
      </el-card>

      <el-card v-if="(detail.children || []).length" shadow="never" class="block">
        <template #header><span>AI 分析阶段</span></template>
        <el-table :data="detail.children" size="small" border stripe>
          <el-table-column prop="category" label="类型" width="120" />
          <el-table-column prop="status" label="状态" width="90" />
          <el-table-column prop="stdout_summary" label="回答摘要" min-width="240" show-overflow-tooltip />
        </el-table>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header><span>关联结果</span></template>
        <div class="links">
          <el-link v-if="detail.runtime_report" type="primary" @click="goRuntime">
            运行时报告：{{ detail.runtime_report.target_display || detail.runtime_report.session_id }}
          </el-link>
          <el-link v-if="detail.auto_iteration_id && isSuperAdmin" type="primary" @click="goAutoIteration">
            自动迭代任务
          </el-link>
          <el-link type="primary" @click="goErrorCodes">错误码目录</el-link>
        </div>
        <div v-if="detail.enhancement_review" class="enhancement">
          <h4>技能包增强审查</h4>
          <pre class="mono">{{ JSON.stringify(detail.enhancement_review, null, 2) }}</pre>
        </div>
      </el-card>

      <el-card shadow="never" class="block">
        <template #header><span>审计与回滚</span></template>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="回滚能力">{{ rec.rollback_capability || '—' }}</el-descriptions-item>
          <el-descriptions-item label="回滚建议">{{ rec.rollback_advice || '—' }}</el-descriptions-item>
        </el-descriptions>
        <el-table v-if="(detail.events || []).length" :data="detail.events" size="small" border stripe class="events-table">
          <el-table-column label="时间" width="168">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="phase" label="阶段" width="100" />
          <el-table-column prop="message" label="消息" min-width="200" show-overflow-tooltip />
        </el-table>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getAISreExecutionDetail, type ClientExecutionDetail } from '../../api/aisreExecutions'

const route = useRoute()
const router = useRouter()
const shellPrefix = route.path.startsWith('/app') ? '/app' : '/admin'
const isSuperAdmin = computed(() => {
  try {
    const u = JSON.parse(localStorage.getItem('userInfo') || '{}')
    return u?.role === 'super_admin'
  } catch {
    return false
  }
})

const loading = ref(false)
const detail = ref<ClientExecutionDetail | null>(null)
const rec = computed(() => (detail.value?.record || {}) as Record<string, any>)
const meta = computed(() => (rec.value.metadata || {}) as Record<string, any>)

const load = async () => {
  const id = String(route.params.id || '')
  if (!id) return
  loading.value = true
  try {
    detail.value = await getAISreExecutionDetail(id)
  } catch {
    detail.value = null
  } finally {
    loading.value = false
  }
}

const goBack = () => router.push(`${shellPrefix}/ai-sre/executions`)
const goRuntime = () => router.push(`${shellPrefix}/advanced/runtime-observe`)
const goAutoIteration = () => router.push({ path: '/admin/auto-iterations', query: { id: detail.value?.auto_iteration_id } })
const goErrorCodes = () => router.push(`${shellPrefix}/help/error-codes`)

const formatTime = (t?: string) => (t ? String(t).replace('T', ' ').slice(0, 19) : '')
const formatDuration = (ms?: number) => {
  if (ms == null || ms <= 0) return '—'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}
const statusTag = (s: string) => (s === 'success' ? 'success' : s === 'failed' ? 'danger' : 'info')
const evidenceTag = (v?: string) => {
  if (v === 'complete') return 'success'
  if (v === 'partial') return 'warning'
  if (v === 'missing' || v === 'incomplete') return 'danger'
  return 'info'
}
const legacyLabel = (k: string) => {
  if (k === 'legacy_ai_diagnose') return '历史 AI 诊断'
  if (k === 'legacy_go_runtime') return '历史运行时执行'
  if (k === 'legacy_cli') return '历史 CLI 记录'
  return k
}

onMounted(() => void load())
</script>

<style scoped>
.page-head {
  margin-bottom: 14px;
}
.block {
  margin-bottom: 14px;
}
.mono {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  margin: 0;
}
.mono.err {
  color: var(--el-color-danger);
}
.links {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}
.enhancement {
  margin-top: 12px;
}
.events-table {
  margin-top: 12px;
}
.legacy-tag {
  margin-top: 8px;
}
</style>
