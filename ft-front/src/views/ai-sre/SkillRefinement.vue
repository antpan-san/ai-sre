<template>
  <div class="skill-refinement page-shell page-shell--crud-wide">
    <header class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">技能精炼</h2>
        <p class="page-desc--muted">
          诊断样本池、待增强审查与本地规则命中率。客户端 check 跳过 AI 依赖 CLI 内置本地规则（非「已精炼」按钮）；「精炼」更新服务端技能包 prompt。
        </p>
      </div>
      <div class="page-head-actions">
        <el-button size="small" :loading="loading" @click="reload">刷新</el-button>
        <el-button size="small" link type="primary" @click="goSkillsCatalog">技能包目录</el-button>
        <el-button size="small" link type="primary" @click="goAutoIterations">自动迭代</el-button>
        <el-button size="small" :loading="backfillLoading" @click="runBackfill">回填 PG 样本</el-button>
      </div>
    </header>

    <section v-loading="loading" class="stats-row">
      <article class="stat-tile">
        <span class="stat-label">样本（{{ sampleSummary?.since_hours ?? 24 }}h）</span>
        <strong>{{ sampleSummary?.total_samples ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">CLI check</span>
        <strong>{{ sampleSummary?.cli_check_count ?? 0 }}</strong>
      </article>
      <article class="stat-tile stat-tile--ok">
        <span class="stat-label">本地规则命中率</span>
        <strong>{{ sampleSummary?.rule_hit_rate_pct ?? 0 }}%</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">AI 调用占比</span>
        <strong>{{ sampleSummary?.ai_call_rate_pct ?? 0 }}%</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">降本潜力（规则替代）</span>
        <strong>{{ sampleSummary?.ai_avoidance_pct ?? 0 }}%</strong>
      </article>
      <article class="stat-tile stat-tile--warn">
        <span class="stat-label">待增强</span>
        <strong>{{ enhancementSummary?.open_count ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">高优审查</span>
        <strong>{{ enhancementSummary?.high_priority ?? 0 }}</strong>
      </article>
      <article class="stat-tile">
        <span class="stat-label">节省潜力分</span>
        <strong>{{ enhancementSummary?.total_savings_score ?? 0 }}</strong>
      </article>
    </section>

    <el-tabs v-model="activeTab" class="view-tabs">
      <el-tab-pane label="待增强审查" name="reviews">
        <el-table :data="reviews" stripe border size="small" empty-text="暂无待增强项">
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ formatTime(row.time) }}</template>
          </el-table-column>
          <el-table-column prop="topic" label="Topic" width="100" />
          <el-table-column prop="command_kind" label="类型" width="80" />
          <el-table-column label="优先级" width="88" align="center">
            <template #default="{ row }">
              <el-tag :type="priorityTag(row.priority)" size="small">{{ row.priority }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="savings_score" label="潜力" width="72" align="center" />
          <el-table-column prop="similar_recent_count" label="相似" width="72" align="center" />
          <el-table-column label="建议" min-width="200" show-overflow-tooltip>
            <template #default="{ row }">{{ (row.recommendations || []).join('；') || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220" align="center">
            <template #default="{ row }">
              <el-button v-if="row.request_id" link type="primary" size="small" @click="openByRequest(row.request_id)">复盘</el-button>
              <el-button link type="primary" size="small" @click="openRefine(row.topic)">精炼</el-button>
              <el-button link size="small" @click="markReview(row, 'refined')">关闭审查</el-button>
              <el-button link type="danger" size="small" @click="markReview(row, 'dismissed')">忽略</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="诊断样本" name="samples">
        <div class="toolbar">
          <el-input v-model="sampleTopic" clearable placeholder="Topic 过滤" class="filter-input" @keyup.enter="loadSamples" />
          <el-select v-model="sampleHours" size="small" class="filter-hours" @change="loadSamples">
            <el-option :value="24" label="24h" />
            <el-option :value="168" label="7d" />
            <el-option :value="720" label="30d" />
          </el-select>
          <el-button type="primary" size="small" @click="loadSamples">查询</el-button>
        </div>
        <el-table v-loading="samplesLoading" :data="samples" stripe border size="small" empty-text="暂无样本">
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ formatTime(row.time) }}</template>
          </el-table-column>
          <el-table-column prop="topic" label="Topic" width="96" />
          <el-table-column prop="target" label="目标" min-width="120" show-overflow-tooltip />
          <el-table-column label="规则/AI" width="96" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.local_rule_hit || row.rule_hit" size="small" type="success">规则</el-tag>
              <el-tag v-else-if="row.used_ai" size="small" type="warning">AI</el-tag>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column prop="evidence_completeness" label="证据" width="88" />
          <el-table-column prop="cli_version" label="CLI" width="72" />
          <el-table-column label="结论摘要" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.answer_head || '—' }}</template>
          </el-table-column>
          <el-table-column label="复盘" width="80" align="center">
            <template #default="{ row }">
              <el-button v-if="row.execution_id" link type="primary" size="small" @click="openExecution(row.execution_id)">
                查看
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="CLI 反馈" name="feedbacks">
        <el-table :data="feedbacks" stripe border size="small" empty-text="暂无 CLI 反馈">
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column prop="topic" label="Topic" width="96" />
          <el-table-column prop="source" label="来源" width="100" show-overflow-tooltip />
          <el-table-column prop="classification" label="分类" width="140" show-overflow-tooltip />
          <el-table-column label="有用" width="72" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.helpful === true" type="success" size="small">是</el-tag>
              <el-tag v-else-if="row.helpful === false" type="warning" size="small">否</el-tag>
              <span v-else>—</span>
            </template>
          </el-table-column>
          <el-table-column label="需迭代" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.need_iteration ? 'warning' : 'info'" size="small">{{ row.need_iteration ? '是' : '否' }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="摘要" min-width="180" show-overflow-tooltip>
            <template #default="{ row }">{{ row.summary || row.user_message || '—' }}</template>
          </el-table-column>
          <el-table-column label="操作" width="160" align="center">
            <template #default="{ row }">
              <el-button v-if="row.execution_id" link type="primary" size="small" @click="openExecution(row.execution_id)">复盘</el-button>
              <el-button v-else-if="row.request_id" link type="primary" size="small" @click="openByRequest(row.request_id)">复盘</el-button>
              <el-button v-if="row.auto_iteration_id" link type="primary" size="small" @click="goAutoIterationDetail(row.auto_iteration_id)">迭代</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="质量趋势" name="trend">
        <div class="toolbar">
          <el-select v-model="trendHours" size="small" class="filter-hours" @change="loadTrend">
            <el-option :value="168" label="7d" />
            <el-option :value="720" label="30d" />
          </el-select>
          <el-button type="primary" size="small" @click="loadTrend">刷新趋势</el-button>
        </div>
        <div v-if="!trendBuckets.length" class="muted trend-empty">暂无趋势数据（可先执行回填 PG 样本）</div>
        <div v-else class="trend-chart">
          <div v-for="row in trendBuckets" :key="row.bucket_start" class="trend-row">
            <span class="trend-label">{{ formatTrendLabel(row.bucket_start) }}</span>
            <div class="trend-bars">
              <span class="trend-bar trend-bar--total" :style="{ width: trendBarWidth(row.total) }" :title="`样本 ${row.total}`" />
              <span class="trend-bar trend-bar--rule" :style="{ width: trendBarWidth(row.rule_hit) }" :title="`规则 ${row.rule_hit}`" />
              <span class="trend-bar trend-bar--ai" :style="{ width: trendBarWidth(row.used_ai) }" :title="`AI ${row.used_ai}`" />
            </div>
            <span class="trend-counts">{{ row.total }} / {{ row.rule_hit }} / {{ row.used_ai }}</span>
          </div>
          <div class="trend-legend">
            <span><i class="dot dot--total" />样本</span>
            <span><i class="dot dot--rule" />规则命中</span>
            <span><i class="dot dot--ai" />AI</span>
          </div>
        </div>
      </el-tab-pane>

      <el-tab-pane label="高频 Topic" name="topics">
        <el-table :data="sampleSummary?.top_topics || []" stripe border size="small" empty-text="暂无数据">
          <el-table-column prop="topic" label="Topic" />
          <el-table-column prop="count" label="样本数" width="120" align="right" />
          <el-table-column label="待增强" width="120" align="right">
            <template #default="{ row }">{{ enhancementSummary?.by_topic?.[row.topic] ?? 0 }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-dialog v-model="refineOpen" title="触发技能精炼" width="640px" @closed="refineDraftYaml = ''">
      <el-form label-width="88px" size="small">
        <el-form-item label="Topic">
          <el-input v-model="refineTopic" />
        </el-form-item>
        <el-form-item label="提示">
          <el-input v-model="refineHint" type="textarea" :rows="3" placeholder="可选：补充精炼方向" />
        </el-form-item>
        <el-form-item label="Dry run">
          <el-switch v-model="refineDryRun" />
        </el-form-item>
      </el-form>
      <div v-if="refineDraftYaml" class="draft-block">
        <h4>Dry run 草稿 YAML</h4>
        <pre class="draft-yaml">{{ refineDraftYaml }}</pre>
      </div>
      <template #footer>
        <el-button @click="refineOpen = false">关闭</el-button>
        <el-button type="primary" :loading="refineLoading" @click="submitRefine">开始精炼</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  adminRefineSkill,
  backfillAdminDiagnoseSamples,
  getAdminDiagnoseSampleSummary,
  getAdminDiagnoseSampleTrend,
  getAdminSkillEnhancementSummary,
  listAdminAutoIterationFeedbacks,
  listAdminDiagnoseSamples,
  listAdminSkillEnhancementReviews,
  lookupExecutionByRequestID,
  updateSkillEnhancementStatus,
  type AutoIterationFeedbackItem,
  type DiagnoseSample,
  type DiagnoseSampleSummary,
  type DiagnoseSampleTrendBucket,
  type SkillEnhancementReview,
  type SkillEnhancementSummary
} from '../../api/skillRefinement'

const router = useRouter()
const loading = ref(false)
const samplesLoading = ref(false)
const activeTab = ref('reviews')
const sampleTopic = ref('')
const sampleHours = ref(168)
const sampleSummary = ref<DiagnoseSampleSummary | null>(null)
const enhancementSummary = ref<SkillEnhancementSummary | null>(null)
const reviews = ref<SkillEnhancementReview[]>([])
const samples = ref<DiagnoseSample[]>([])
const feedbacks = ref<AutoIterationFeedbackItem[]>([])
const refineOpen = ref(false)
const refineLoading = ref(false)
const refineTopic = ref('')
const refineHint = ref('')
const refineDryRun = ref(true)
const refineDraftYaml = ref('')
const backfillLoading = ref(false)
const trendHours = ref(168)
const trendBuckets = ref<DiagnoseSampleTrendBucket[]>([])
const trendMax = ref(1)

const formatTime = (t?: string) => (t ? String(t).replace('T', ' ').slice(0, 19) : '—')
const formatTrendLabel = (t: string) => String(t).replace('T', ' ').slice(0, 10)
const trendBarWidth = (n: number) => `${Math.max(4, Math.round((n / trendMax.value) * 100))}%`
const priorityTag = (p?: string) => (p === 'high' ? 'danger' : p === 'medium' ? 'warning' : 'info')

const loadSamples = async () => {
  samplesLoading.value = true
  try {
    const data = await listAdminDiagnoseSamples({
      topic: sampleTopic.value.trim() || undefined,
      limit: 80,
      hours: sampleHours.value
    })
    samples.value = data.samples || []
  } catch {
    samples.value = []
  } finally {
    samplesLoading.value = false
  }
}

const loadFeedbacks = async () => {
  try {
    const data = await listAdminAutoIterationFeedbacks(50)
    feedbacks.value = data.feedbacks || []
  } catch {
    feedbacks.value = []
  }
}

const loadTrend = async () => {
  try {
    const data = await getAdminDiagnoseSampleTrend(trendHours.value, 24)
    trendBuckets.value = data.buckets || []
    trendMax.value = Math.max(1, ...trendBuckets.value.map((b) => b.total))
  } catch {
    trendBuckets.value = []
    trendMax.value = 1
  }
}

const reload = async () => {
  loading.value = true
  try {
    const [sum, enh, list] = await Promise.all([
      getAdminDiagnoseSampleSummary(24),
      getAdminSkillEnhancementSummary(30),
      listAdminSkillEnhancementReviews(50, true)
    ])
    sampleSummary.value = sum
    enhancementSummary.value = enh
    reviews.value = list.reviews || []
    if (activeTab.value === 'samples') {
      await loadSamples()
    }
    if (activeTab.value === 'feedbacks') {
      await loadFeedbacks()
    }
    if (activeTab.value === 'trend') {
      await loadTrend()
    }
  } catch {
    sampleSummary.value = null
    enhancementSummary.value = null
    reviews.value = []
  } finally {
    loading.value = false
  }
}

const goSkillsCatalog = () => router.push({ path: '/admin/billing/ai-sre-skills', query: { tab: 'enhancement' } })
const goAutoIterations = () => router.push('/admin/auto-iterations')
const runBackfill = async () => {
  backfillLoading.value = true
  try {
    const res = await backfillAdminDiagnoseSamples()
    ElMessage.success(`回填完成：新增 ${res.inserted}，跳过 ${res.skipped}`)
    await reload()
  } catch {
    ElMessage.error('回填失败')
  } finally {
    backfillLoading.value = false
  }
}
const goAutoIterationDetail = (id: string) => router.push(`/admin/auto-iterations?id=${id}`)
const openExecution = (id: string) => router.push(`/admin/ai-sre/executions/${id}`)

const openByRequest = async (requestId: string) => {
  try {
    const data = await lookupExecutionByRequestID(requestId)
    if (data.execution_id) {
      openExecution(data.execution_id)
      return
    }
    ElMessage.warning('未找到关联执行记录')
  } catch {
    ElMessage.error('查询执行记录失败')
  }
}

const openRefine = (topic: string) => {
  refineTopic.value = topic
  refineHint.value = ''
  refineDryRun.value = true
  refineDraftYaml.value = ''
  refineOpen.value = true
}

const submitRefine = async () => {
  const topic = refineTopic.value.trim()
  if (!topic) return
  refineLoading.value = true
  try {
    const res = await adminRefineSkill({
      topic,
      user_hint: refineHint.value.trim() || undefined,
      dry_run: refineDryRun.value,
      max_samples: 12,
      max_feedback: 8,
      timeout_sec: 120
    })
    const draft = String(res.draft_yaml || '')
    if (refineDryRun.value && draft) {
      refineDraftYaml.value = draft
      ElMessage.success('Dry run 完成，草稿 YAML 已展示')
    } else {
      refineDraftYaml.value = ''
      ElMessage.success('精炼任务已完成')
      refineOpen.value = false
    }
    await reload()
  } catch {
    ElMessage.error('精炼失败')
  } finally {
    refineLoading.value = false
  }
}

const markReview = async (row: SkillEnhancementReview, status: 'refined' | 'dismissed') => {
  const reviewKey = row.review_key?.trim()
  const requestId = row.request_id?.trim()
  if (!requestId && !reviewKey) {
    ElMessage.warning('缺少 review_key，无法更新状态')
    return
  }
  try {
    await updateSkillEnhancementStatus({
      request_id: requestId || undefined,
      review_key: reviewKey || undefined,
      topic: row.topic,
      status,
      note: status === 'refined' ? '管理员关闭审查项' : '管理员忽略'
    })
    const wantsLocalRule = (row.suggested_actions || []).includes('local_rule')
    if (status === 'refined' && wantsLocalRule) {
      ElMessage.success({
        message:
          '已关闭审查项。若要客户端 check 不再调用 AI，须发布含对应本地规则的 ai-sre 新版本；「精炼」仅更新服务端技能包。',
        duration: 6000
      })
    } else {
      ElMessage.success('已更新')
    }
    await reload()
  } catch {
    ElMessage.error('更新失败')
  }
}

onMounted(() => {
  void reload()
})

watch(activeTab, (tab) => {
  if (tab === 'feedbacks' && !feedbacks.value.length) {
    void loadFeedbacks()
  }
  if (tab === 'samples' && !samples.value.length) {
    void loadSamples()
  }
  if (tab === 'trend' && !trendBuckets.value.length) {
    void loadTrend()
  }
})
</script>

<style scoped>
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.page-head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.stats-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 10px;
  margin-bottom: 16px;
}
.stat-tile {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}
.stat-tile--warn strong {
  color: var(--el-color-warning);
}
.stat-tile--ok strong {
  color: var(--el-color-success);
}
.stat-label {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
}
.filter-input {
  width: 160px;
}
.filter-hours {
  width: 88px;
}
.muted {
  color: var(--el-text-color-placeholder);
}
.draft-block {
  margin-top: 8px;
}
.draft-block h4 {
  margin: 0 0 6px;
  font-size: 13px;
}
.draft-yaml {
  max-height: 320px;
  overflow: auto;
  margin: 0;
  padding: 10px;
  font-size: 12px;
  line-height: 1.45;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-word;
}
.trend-empty {
  padding: 24px 0;
  text-align: center;
}
.trend-chart {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.trend-row {
  display: grid;
  grid-template-columns: 88px 1fr 96px;
  gap: 10px;
  align-items: center;
  font-size: 12px;
}
.trend-label {
  color: var(--el-text-color-secondary);
}
.trend-bars {
  position: relative;
  height: 18px;
  background: var(--el-fill-color-light);
  border-radius: 4px;
  overflow: hidden;
}
.trend-bar {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  min-width: 2px;
  opacity: 0.85;
}
.trend-bar--total {
  background: var(--el-color-primary-light-5);
  z-index: 1;
}
.trend-bar--rule {
  background: var(--el-color-success);
  z-index: 2;
}
.trend-bar--ai {
  background: var(--el-color-warning);
  z-index: 3;
}
.trend-counts {
  text-align: right;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.trend-legend {
  display: flex;
  gap: 16px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 2px;
  margin-right: 4px;
  vertical-align: middle;
}
.dot--total {
  background: var(--el-color-primary-light-5);
}
.dot--rule {
  background: var(--el-color-success);
}
.dot--ai {
  background: var(--el-color-warning);
}
</style>
