<template>
  <div class="error-codes-lookup page-shell page-shell--fill">
    <header class="toolbar-top">
      <div class="toolbar-top__left">
        <h2 class="toolbar-top__title">错误码</h2>
        <el-tag v-if="!loading" type="success" size="small" effect="plain">{{ codes.length }}</el-tag>
        <span class="toolbar-top__meta">与 CLI 同源</span>
      </div>
      <el-popover placement="bottom-end" :width="340" trigger="click">
        <template #reference>
          <el-button type="primary" link size="small">数据来源与用法</el-button>
        </template>
        <div class="help-popover">
          <p>目录由内置技能包 <strong>opsfleet_error_codes_v1</strong> 提供，与 CLI 使用同一 <code>GET /api/ai/error-codes</code>。</p>
          <p>日志中 <code>[ERROR-CODE] OPSFLEET_…</code> 可复制到左侧搜索；支持 URL 参数 <code>?code=</code> 深链打开。</p>
        </div>
      </el-popover>
    </header>

    <div class="lookup-layout">
      <aside class="lookup-pane lookup-pane--list">
        <div class="pane-toolbar">
          <el-input
            v-model="keyword"
            clearable
            placeholder="搜索代码、摘要、根因…"
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
        <div class="list-body">
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
          <el-empty v-if="!loading && filteredCodes.length === 0" description="无匹配项" />
        </div>
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
  const line = `ai-sre check code ${code}`
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
      selected.value = filteredCodes.value[0] ?? null
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
.help-popover p {
  margin: 0 0 8px;
  font-size: 13px;
  line-height: 1.55;
  color: #334155;
}

.help-popover p:last-child {
  margin-bottom: 0;
}

.help-popover code {
  font-size: 12px;
  padding: 1px 5px;
  border-radius: 4px;
  background: #f1f5f9;
}

.error-codes-lookup {
  gap: 8px;
}

.toolbar-top {
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 2px 0 4px;
  min-height: 0;
}

.toolbar-top__left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 12px;
  min-width: 0;
}

.toolbar-top__title {
  margin: 0;
  font-size: var(--page-header-title-max, 18px);
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--layout-sidebar-text-strong, #111827);
}

.toolbar-top__meta {
  font-size: 12px;
  color: #64748b;
  white-space: nowrap;
}

.toolbar-top__meta code {
  font-size: 11px;
  padding: 0 4px;
  border-radius: 4px;
  background: #f1f5f9;
  color: #0f172a;
}

.lookup-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: minmax(260px, 320px) minmax(0, 1fr);
  gap: 12px;
  overflow: hidden;
}

@media (max-width: 1100px) {
  .lookup-layout {
    grid-template-columns: 1fr;
    grid-template-rows: minmax(200px, 38vh) minmax(0, 1fr);
  }
}

.lookup-pane {
  background: #fff;
  border-radius: 12px;
  border: 1px solid var(--layout-sidebar-border, #e5e7eb);
  box-shadow: 0 1px 3px rgb(0 0 0 / 6%);
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.lookup-pane--list {
  min-height: 0;
}

.pane-toolbar {
  flex: 0 0 auto;
  padding: 10px 10px 8px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-bottom: 1px solid #f1f5f9;
}

.kind-filter {
  width: 100%;
}

.list-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
}

.code-list {
  list-style: none;
  margin: 0;
  padding: 8px 8px 12px;
}

.code-list-item {
  padding: 10px 10px;
  margin-bottom: 6px;
  border-radius: 8px;
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
  box-shadow: 0 2px 8px rgba(255, 105, 0, 0.1);
}

.code-list-item__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.code-chip {
  font-size: 11px;
  font-weight: 600;
  color: #0f172a;
  word-break: break-all;
}

.kind-pill {
  flex-shrink: 0;
}

.code-list-item__summary {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.4;
  color: #64748b;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.lookup-pane--detail {
  padding: 14px 16px 16px;
  overflow-y: auto;
  overflow-x: hidden;
  -webkit-overflow-scrolling: touch;
}

.detail-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.detail-head-titles {
  min-width: 0;
  flex: 1;
}

.detail-code {
  margin: 0 0 4px;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: #0f172a;
  word-break: break-all;
}

.detail-summary {
  margin: 0;
  font-size: 14px;
  line-height: 1.45;
  color: #334155;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  flex-shrink: 0;
}

.detail-block {
  margin: 0;
  padding: 12px 14px;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: #1e293b;
}

.detail-block--shell {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;
  font-size: 12px;
}

.evidence-list {
  margin: 0;
  padding-left: 16px;
  color: #475569;
  font-size: 12px;
}

.evidence-list li {
  margin-bottom: 6px;
}

.evidence-list code {
  font-size: 11px;
  background: #f1f5f9;
  padding: 2px 5px;
  border-radius: 4px;
  color: #0f172a;
}

.related-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.related-tag {
  cursor: pointer;
}

.detail-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 160px;
  color: #64748b;
  font-size: 13px;
}

.spin {
  font-size: 20px;
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.lookup-pane--detail :deep(.el-divider) {
  margin: 12px 0 10px;
}
</style>
