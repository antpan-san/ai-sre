<template>
  <div class="error-codes-lookup page-shell page-shell--dashboard">
    <header class="page-header">
      <div class="page-header-inner">
        <span class="page-kicker">OpsFleet · Knowledge</span>
        <h2 class="page-title">部署错误码查询</h2>
        <p class="page-desc">
          目录与根因说明由平台内置技能包 <strong>opsfleet_error_codes_v1</strong> 提供，与
          <code>ai-sre analyze code &lt;CODE&gt;</code> 同源；下列条目均含完整字段，可直接检索与复制。
        </p>
      </div>
    </header>

    <el-alert type="success" show-icon :closable="false" class="catalog-banner">
      <template #title>
        已加载 <strong>{{ codes.length }}</strong> 条结构化错误码（与安装 / 下载 / K8s 编排链对齐）
      </template>
      <template #default>
        <span class="banner-hint">
          日志中出现以 <code>[ERROR-CODE]</code> 开头的行时，将后面的 <code>OPSFLEET_*</code> 复制到下方搜索即可定位。
        </span>
      </template>
    </el-alert>

    <div class="lookup-layout">
      <aside class="lookup-pane lookup-pane--list">
        <div class="pane-toolbar">
          <el-input
            v-model="keyword"
            clearable
            placeholder="搜索代码、摘要、根因关键词…"
            :prefix-icon="Search"
            @clear="applyQueryFromRoute"
          />
          <el-select v-model="kindFilter" placeholder="类型" class="kind-filter">
            <el-option label="全部" value="all" />
            <el-option label="错误 E_" value="error" />
            <el-option label="信息 I_" value="info" />
            <el-option label="下载 DL_" value="dl" />
            <el-option label="其它" value="other" />
          </el-select>
        </div>
        <el-scrollbar class="list-scroll">
          <ul class="code-list" role="list">
            <li
              v-for="row in filteredCodes"
              :key="row.code"
              role="listitem"
              :class="['code-list-item', { 'is-active': selected?.code === row.code }]"
              @click="selectCode(row)"
            >
              <div class="code-list-item__row">
                <code class="code-chip">{{ row.code }}</code>
                <el-tag :type="kindTagType(row.code)" size="small" effect="plain" class="kind-pill">
                  {{ kindLabel(row.code) }}
                </el-tag>
              </div>
              <p class="code-list-item__summary">{{ row.summary }}</p>
            </li>
          </ul>
          <el-empty v-if="!loading && filteredCodes.length === 0" description="无匹配项，请调整筛选条件" />
        </el-scrollbar>
      </aside>

      <section class="lookup-pane lookup-pane--detail">
        <template v-if="selected">
          <div class="detail-head">
            <div class="detail-head-titles">
              <h3 class="detail-code">{{ selected.code }}</h3>
              <p class="detail-summary">{{ selected.summary }}</p>
            </div>
            <div class="detail-actions">
              <el-button type="primary" plain size="small" :icon="DocumentCopy" @click="copyCode(selected.code)">
                复制代码
              </el-button>
              <el-button size="small" :icon="DocumentCopy" @click="copyCliHint(selected.code)">复制 CLI 命令</el-button>
            </div>
          </div>

          <el-divider content-position="left">根因</el-divider>
          <pre class="detail-block detail-block--prose">{{ selected.root_cause?.trim() || '—' }}</pre>

          <template v-if="selected.typical_evidence?.length">
            <el-divider content-position="left">典型日志片段</el-divider>
            <ul class="evidence-list">
              <li v-for="(line, idx) in selected.typical_evidence" :key="idx">
                <code>{{ line }}</code>
              </li>
            </ul>
          </template>

          <el-divider content-position="left">立即恢复（一行）</el-divider>
          <pre class="detail-block detail-block--shell">{{ selected.recovery_one_liner?.trim() || '—' }}</pre>

          <el-divider content-position="left">平台改进（代码落点）</el-divider>
          <pre class="detail-block detail-block--prose">{{ selected.platform_followup?.trim() || '—' }}</pre>

          <template v-if="selected.related_codes?.length">
            <el-divider content-position="left">关联错误码</el-divider>
            <div class="related-wrap">
              <el-tag
                v-for="rel in selected.related_codes"
                :key="rel"
                class="related-tag"
                effect="plain"
                type="info"
                @click="jumpToRelated(rel)"
              >
                {{ rel }}
              </el-tag>
            </div>
          </template>
        </template>
        <el-empty v-else-if="!loading" description="请从左侧选择一条错误码" />
        <div v-if="loading" class="detail-loading">
          <el-icon class="spin"><Loading /></el-icon>
          <span>加载目录中…</span>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Search, DocumentCopy, Loading } from '@element-plus/icons-vue'
import { copyTextToClipboard } from '../../utils/clipboard'
import { getErrorCodesCatalog, type OpsfleetErrorCode } from '../../api/error-codes'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const codes = ref<OpsfleetErrorCode[]>([])
const selected = ref<OpsfleetErrorCode | null>(null)
const keyword = ref('')
const kindFilter = ref<'all' | 'error' | 'info' | 'dl' | 'other'>('all')

function classifyCode(code: string): 'error' | 'info' | 'dl' | 'other' {
  const u = code.toUpperCase()
  if (u.includes('_I_')) return 'info'
  if (u.startsWith('OPSFLEET_DL_')) return 'dl'
  if (u.includes('_E_')) return 'error'
  return 'other'
}

function kindLabel(code: string) {
  const k = classifyCode(code)
  return { error: '错误', info: '信息', dl: '下载', other: '其它' }[k]
}

function kindTagType(code: string): 'danger' | 'success' | 'warning' | 'info' {
  const k = classifyCode(code)
  if (k === 'error') return 'danger'
  if (k === 'info') return 'success'
  if (k === 'dl') return 'warning'
  return 'info'
}

const filteredCodes = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return codes.value.filter(c => {
    if (kindFilter.value !== 'all' && classifyCode(c.code) !== kindFilter.value) return false
    if (!q) return true
    const blob = [
      c.code,
      c.summary,
      c.root_cause,
      ...(c.typical_evidence || []),
      c.recovery_one_liner,
      c.platform_followup,
      ...(c.related_codes || []),
    ]
      .join('\n')
      .toLowerCase()
    return blob.includes(q)
  })
})

function selectCode(row: OpsfleetErrorCode) {
  selected.value = row
  const next = { ...route.query, code: row.code }
  router.replace({ query: next })
}

function jumpToRelated(code: string) {
  const found = codes.value.find(c => c.code.toUpperCase() === code.trim().toUpperCase())
  keyword.value = ''
  kindFilter.value = 'all'
  if (found) {
    selectCode(found)
    return
  }
  keyword.value = code
  ElMessage.info('关联码在当前目录中无独立卡片，已用搜索关键字展示')
}

async function copyCode(text: string) {
  try {
    await copyTextToClipboard(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

async function copyCliHint(code: string) {
  const line = `ai-sre analyze code ${code}`
  await copyCode(line)
}

function applyQueryFromRoute() {
  const raw = typeof route.query.code === 'string' ? route.query.code : ''
  const want = raw.trim().toUpperCase()
  if (!want || codes.value.length === 0) return
  const hit = codes.value.find(c => c.code.toUpperCase() === want)
  if (hit) {
    selected.value = hit
    return
  }
  keyword.value = raw
}

async function load() {
  loading.value = true
  try {
    const data = await getErrorCodesCatalog()
    codes.value = Array.isArray(data.codes) ? data.codes : []
    if (!codes.value.length) {
      ElMessage.warning('错误码目录为空，请检查后端技能包是否加载')
    }
    applyQueryFromRoute()
    if (!selected.value && filteredCodes.value.length) {
      selected.value = filteredCodes.value[0]
    }
  } catch {
    codes.value = []
  } finally {
    loading.value = false
  }
}

onMounted(load)

watch(
  () => route.query.code,
  () => {
    applyQueryFromRoute()
  }
)
</script>

<style scoped>
.error-codes-lookup {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.page-header {
  padding-bottom: 4px;
}

.page-header-inner {
  max-width: 960px;
}

.page-kicker {
  display: inline-block;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--el-color-primary);
  margin-bottom: 6px;
}

.page-title {
  font-size: 26px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--layout-sidebar-text-strong, #111827);
  margin: 0 0 8px;
}

.page-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
  color: #64748b;
  max-width: 880px;
}

.page-desc code {
  font-size: 13px;
  padding: 1px 6px;
  border-radius: 6px;
  background: #f1f5f9;
  color: #0f172a;
}

.catalog-banner {
  border-radius: 12px;
}

.banner-hint {
  font-size: 13px;
  color: #475569;
}

.lookup-layout {
  display: grid;
  grid-template-columns: minmax(300px, 380px) minmax(0, 1fr);
  gap: 20px;
  align-items: stretch;
  min-height: 520px;
}

@media (max-width: 1100px) {
  .lookup-layout {
    grid-template-columns: 1fr;
  }
}

.lookup-pane {
  background: #fff;
  border-radius: 14px;
  border: 1px solid var(--layout-sidebar-border, #e5e7eb);
  box-shadow: 0 1px 3px rgb(0 0 0 / 6%);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.lookup-pane--list {
  max-height: min(78vh, 900px);
}

.pane-toolbar {
  padding: 14px 14px 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  border-bottom: 1px solid #f1f5f9;
  flex-shrink: 0;
}

.kind-filter {
  width: 100%;
}

.list-scroll {
  flex: 1;
  min-height: 0;
}

.list-scroll :deep(.el-scrollbar__wrap) {
  max-height: calc(78vh - 120px);
}

.code-list {
  list-style: none;
  margin: 0;
  padding: 8px 10px 16px;
}

.code-list-item {
  padding: 12px 12px;
  margin-bottom: 8px;
  border-radius: 10px;
  cursor: pointer;
  border: 1px solid transparent;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    box-shadow 0.15s ease;
}

.code-list-item:hover {
  background: #f8fafc;
  border-color: #e2e8f0;
}

.code-list-item.is-active {
  background: linear-gradient(135deg, rgba(255, 105, 0, 0.06), rgba(255, 140, 66, 0.08));
  border-color: rgba(255, 105, 0, 0.35);
  box-shadow: 0 2px 10px rgba(255, 105, 0, 0.12);
}

.code-list-item__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.code-chip {
  font-size: 12px;
  font-weight: 600;
  color: #0f172a;
  word-break: break-all;
}

.kind-pill {
  flex-shrink: 0;
}

.code-list-item__summary {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.45;
  color: #64748b;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.lookup-pane--detail {
  padding: 20px 22px 28px;
  max-height: min(78vh, 900px);
  overflow: auto;
}

.detail-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
}

.detail-head-titles {
  min-width: 0;
  flex: 1;
}

.detail-code {
  margin: 0 0 6px;
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: #0f172a;
  word-break: break-all;
}

.detail-summary {
  margin: 0;
  font-size: 15px;
  line-height: 1.5;
  color: #334155;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  flex-shrink: 0;
}

.detail-block {
  margin: 0;
  padding: 14px 16px;
  border-radius: 10px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 13px;
  line-height: 1.65;
  white-space: pre-wrap;
  word-break: break-word;
  color: #1e293b;
}

.detail-block--shell {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12.5px;
}

.evidence-list {
  margin: 0;
  padding-left: 18px;
  color: #475569;
  font-size: 13px;
}

.evidence-list li {
  margin-bottom: 8px;
}

.evidence-list code {
  font-size: 12px;
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 6px;
  color: #0f172a;
}

.related-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.related-tag {
  cursor: pointer;
}

.detail-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-height: 240px;
  color: #64748b;
  font-size: 14px;
}

.spin {
  font-size: 22px;
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
