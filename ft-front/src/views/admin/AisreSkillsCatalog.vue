<template>
  <div class="aisre-skills page-shell page-shell--crud-wide">
    <div class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">ai-sre 技能包</h2>
        <p class="page-desc--muted">
          注册表展示已发布技能；「待审资产」来自 CLI 只读诊断任务单，超级管理员审核通过后会写入 generated 并参与诊断匹配。
        </p>
      </div>
    </div>

    <el-tabs v-model="activeTab" class="skills-tabs" @tab-change="onTabChange">
      <el-tab-pane label="注册表" name="registry">
        <div class="tab-toolbar">
          <el-input
            v-model="keyword"
            clearable
            class="skills-filter"
            placeholder="按名称 / 展示名 / topic 过滤"
          />
        </div>
        <el-card shadow="never" v-loading="loading">
          <template v-if="dataDir">
            <p class="data-dir-hint page-desc--muted">技能数据目录：<code>{{ dataDir }}</code></p>
          </template>
          <el-table
            :data="filteredRows"
            stripe
            border
            size="small"
            empty-text="暂无技能包"
            @row-click="onRegistryRowClick"
          >
            <el-table-column prop="display_name" label="展示名" min-width="160" show-overflow-tooltip />
            <el-table-column prop="name" label="名称" min-width="160" show-overflow-tooltip />
            <el-table-column label="Topics" min-width="200">
              <template #default="{ row }">
                {{ (row.topics || []).join(', ') || '—' }}
              </template>
            </el-table-column>
            <el-table-column label="来源" width="110" align="center">
              <template #default="{ row }">
                <el-tag :type="row.source === 'generated' ? 'success' : 'info'" size="small">{{ row.source }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="version" label="版本" width="120" show-overflow-tooltip />
            <el-table-column label="操作" width="96" align="center">
              <template #default="{ row }">
                <el-button type="primary" link size="small" @click.stop="openRegistryDetail(row.name)">查看</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane name="review">
        <template #label>
          <span>待审资产</span>
          <el-badge v-if="reviewPendingCount > 0" :value="reviewPendingCount" class="review-badge" />
        </template>
        <div class="tab-toolbar">
          <el-select v-model="reviewStatus" class="review-status-select" @change="loadReview">
            <el-option label="待审核" value="review" />
            <el-option label="已通过" value="approved" />
            <el-option label="已驳回" value="deprecated" />
            <el-option label="全部" value="" />
          </el-select>
          <el-button :loading="reviewLoading" @click="loadReview">刷新</el-button>
        </div>
        <el-card shadow="never" v-loading="reviewLoading">
          <el-table :data="reviewRows" stripe border size="small" empty-text="暂无技能资产">
            <el-table-column prop="display_name" label="展示名" min-width="140" show-overflow-tooltip />
            <el-table-column prop="topic" label="Topic" width="90" />
            <el-table-column prop="created_by" label="提交人" width="100" show-overflow-tooltip />
            <el-table-column label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="assetStatusTag(row.status)" size="small">{{ assetStatusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="observation_summary" label="观察摘要" min-width="200" show-overflow-tooltip />
            <el-table-column label="操作" width="200" align="center" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link size="small" @click="openAssetDetail(row.id)">详情</el-button>
                <el-button
                  v-if="row.status === 'review'"
                  type="success"
                  link
                  size="small"
                  :loading="approvingId === row.id"
                  @click="onApprove(row)"
                >
                  通过
                </el-button>
                <el-button
                  v-if="row.status === 'review'"
                  type="danger"
                  link
                  size="small"
                  :loading="rejectingId === row.id"
                  @click="onReject(row)"
                >
                  驳回
                </el-button>
              </template>
            </el-table-column>
          </el-table>
          <div class="review-pager">
            <el-pagination
              v-model:current-page="reviewPage"
              v-model:page-size="reviewPageSize"
              :total="reviewTotal"
              layout="total, prev, pager, next"
              small
              @current-change="loadReview"
            />
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="registryDetailOpen" :title="registryDetailTitle" size="min(720px, 92vw)" destroy-on-close>
      <div v-loading="registryDetailLoading" class="drawer-body">
        <template v-if="registryDetailSkill">
          <el-descriptions :column="1" border size="small" class="skill-meta">
            <el-descriptions-item label="名称">{{ registryDetailSkill.pack.name }}</el-descriptions-item>
            <el-descriptions-item label="展示名">{{ registryDetailSkill.pack.display_name }}</el-descriptions-item>
            <el-descriptions-item label="来源">{{ registryDetailSkill.source }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ registryDetailSkill.version }}</el-descriptions-item>
            <el-descriptions-item v-if="registryDetailSkill.path" label="路径">{{ registryDetailSkill.path }}</el-descriptions-item>
          </el-descriptions>
          <div class="json-block-wrap">
            <div class="json-block-head">
              <span>技能包内容（JSON）</span>
              <el-button size="small" @click="copyRegistryJson">复制</el-button>
            </div>
            <pre class="json-pre">{{ registryDetailJson }}</pre>
          </div>
        </template>
      </div>
    </el-drawer>

    <el-drawer v-model="assetDetailOpen" title="技能资产详情" size="min(720px, 92vw)" destroy-on-close>
      <div v-loading="assetDetailLoading" class="drawer-body">
        <template v-if="assetDetail">
          <el-descriptions :column="1" border size="small" class="skill-meta">
            <el-descriptions-item label="ID">{{ assetDetail.id }}</el-descriptions-item>
            <el-descriptions-item label="名称">{{ assetDetail.name }}</el-descriptions-item>
            <el-descriptions-item label="Topic">{{ assetDetail.topic }}</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="assetStatusTag(assetDetail.status)" size="small">{{ assetStatusLabel(assetDetail.status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="提交人">{{ assetDetail.created_by || '—' }}</el-descriptions-item>
            <el-descriptions-item v-if="assetDetail.observation_summary" label="观察摘要">
              {{ assetDetail.observation_summary }}
            </el-descriptions-item>
          </el-descriptions>
          <div class="json-block-wrap">
            <div class="json-block-head">
              <span>版本内容</span>
              <el-button size="small" @click="copyAssetJson">复制</el-button>
            </div>
            <pre class="json-pre">{{ assetDetailJson }}</pre>
          </div>
          <div v-if="assetDetail.status === 'review'" class="asset-actions">
            <el-button type="success" :loading="approvingId === assetDetail.id" @click="onApprove(assetDetail)">
              审核通过并发布
            </el-button>
            <el-button type="danger" :loading="rejectingId === assetDetail.id" @click="onReject(assetDetail)">
              驳回
            </el-button>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminAiSkillDetail, getAdminAiSkills, type RegisteredSkill, type SkillSummary } from '../../api/aiSkills'
import {
  approveAdminSkillAsset,
  getAdminSkillAsset,
  listAdminSkillAssets,
  rejectAdminSkillAsset,
  type SkillAssetDetail,
  type SkillAssetListItem
} from '../../api/skillAssets'
import { copyTextToClipboard } from '../../utils/clipboard'

const activeTab = ref('registry')

const loading = ref(false)
const rows = ref<SkillSummary[]>([])
const dataDir = ref('')
const keyword = ref('')

const reviewLoading = ref(false)
const reviewRows = ref<SkillAssetListItem[]>([])
const reviewTotal = ref(0)
const reviewPage = ref(1)
const reviewPageSize = ref(20)
const reviewStatus = ref('review')
const reviewPendingCount = ref(0)
const approvingId = ref('')
const rejectingId = ref('')

const registryDetailOpen = ref(false)
const registryDetailLoading = ref(false)
const registryDetailSkill = ref<RegisteredSkill | null>(null)
const registryDetailTitle = ref('技能包详情')

const assetDetailOpen = ref(false)
const assetDetailLoading = ref(false)
const assetDetail = ref<SkillAssetDetail | null>(null)

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((r) => {
    const blob = [r.name, r.display_name, ...(r.topics || [])].join(' ').toLowerCase()
    return blob.includes(q)
  })
})

const registryDetailJson = computed(() => {
  if (!registryDetailSkill.value) return ''
  try {
    return JSON.stringify(registryDetailSkill.value.pack, null, 2)
  } catch {
    return ''
  }
})

const assetDetailJson = computed(() => {
  if (!assetDetail.value) return ''
  try {
    return JSON.stringify(assetDetail.value.content, null, 2)
  } catch {
    return ''
  }
})

const assetStatusLabel = (s: string) => {
  switch (s) {
    case 'review':
      return '待审核'
    case 'approved':
      return '已通过'
    case 'deprecated':
      return '已驳回'
    default:
      return s || '—'
  }
}

const assetStatusTag = (s: string) => {
  switch (s) {
    case 'review':
      return 'warning'
    case 'approved':
      return 'success'
    case 'deprecated':
      return 'info'
    default:
      return 'info'
  }
}

const loadRegistry = async () => {
  loading.value = true
  try {
    const data = await getAdminAiSkills()
    rows.value = data.skills || []
    dataDir.value = data.data_dir || ''
  } catch {
    rows.value = []
    dataDir.value = ''
  } finally {
    loading.value = false
  }
}

const loadReviewPendingCount = async () => {
  try {
    const data = await listAdminSkillAssets({ status: 'review', page: 1, page_size: 1 })
    reviewPendingCount.value = data.total || 0
  } catch {
    reviewPendingCount.value = 0
  }
}

const loadReview = async () => {
  reviewLoading.value = true
  try {
    const data = await listAdminSkillAssets({
      status: reviewStatus.value || undefined,
      page: reviewPage.value,
      page_size: reviewPageSize.value
    })
    reviewRows.value = data.items || []
    reviewTotal.value = data.total || 0
    if (reviewStatus.value === 'review') {
      reviewPendingCount.value = data.total || 0
    }
  } catch {
    reviewRows.value = []
    reviewTotal.value = 0
  } finally {
    reviewLoading.value = false
  }
}

const onTabChange = (name: string | number) => {
  if (name === 'review') {
    void loadReview()
  }
}

const openRegistryDetail = async (name: string) => {
  registryDetailTitle.value = name
  registryDetailSkill.value = null
  registryDetailOpen.value = true
  registryDetailLoading.value = true
  try {
    const data = await getAdminAiSkillDetail(name)
    registryDetailSkill.value = data.skill
    const dn = data.skill?.pack?.display_name
    if (dn) registryDetailTitle.value = `${dn} (${name})`
  } catch {
    registryDetailOpen.value = false
  } finally {
    registryDetailLoading.value = false
  }
}

const onRegistryRowClick = (row: SkillSummary) => {
  void openRegistryDetail(row.name)
}

const openAssetDetail = async (id: string) => {
  assetDetail.value = null
  assetDetailOpen.value = true
  assetDetailLoading.value = true
  try {
    const data = await getAdminSkillAsset(id)
    assetDetail.value = data.asset
  } catch {
    assetDetailOpen.value = false
  } finally {
    assetDetailLoading.value = false
  }
}

const onApprove = async (row: SkillAssetListItem | SkillAssetDetail) => {
  try {
    await ElMessageBox.confirm(
      '通过后将把诊断沉淀写入 generated 技能包（默认与当前注册表同 topic 技能合并，保留原有分析步骤），是否继续？',
      '审核通过',
      { type: 'warning', confirmButtonText: '通过并发布', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  approvingId.value = row.id
  try {
    const res = await approveAdminSkillAsset(row.id, { merge_with_registry: true })
    ElMessage.success(res.merged ? `已合并发布至 ${res.path}` : `已发布至 ${res.path || 'generated'}`)
    assetDetailOpen.value = false
    await Promise.all([loadReview(), loadReviewPendingCount(), loadRegistry()])
  } catch {
    ElMessage.error('审核失败')
  } finally {
    approvingId.value = ''
  }
}

const onReject = async (row: SkillAssetListItem | SkillAssetDetail) => {
  let reason = ''
  try {
    const { value } = await ElMessageBox.prompt('可选：填写驳回原因', '驳回技能资产', {
      confirmButtonText: '驳回',
      cancelButtonText: '取消',
      inputPlaceholder: '原因（可选）'
    })
    reason = value || ''
  } catch {
    return
  }
  rejectingId.value = row.id
  try {
    await rejectAdminSkillAsset(row.id, { reason })
    ElMessage.success('已驳回')
    assetDetailOpen.value = false
    await Promise.all([loadReview(), loadReviewPendingCount()])
  } catch {
    ElMessage.error('驳回失败')
  } finally {
    rejectingId.value = ''
  }
}

const copyRegistryJson = async () => {
  if (!registryDetailJson.value) return
  try {
    await copyTextToClipboard(registryDetailJson.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

const copyAssetJson = async () => {
  if (!assetDetailJson.value) return
  try {
    await copyTextToClipboard(assetDetailJson.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  void loadRegistry()
  void loadReviewPendingCount()
})
</script>

<style scoped>
.page-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.page-head-copy {
  flex: 1;
  min-width: 220px;
}

.skills-tabs {
  margin-top: 4px;
}

.tab-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.skills-filter {
  width: min(320px, 100%);
}

.review-status-select {
  width: 160px;
}

.review-badge {
  margin-left: 6px;
}

.review-pager {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
}

.data-dir-hint {
  margin: 0 0 12px;
  font-size: 13px;
}

.data-dir-hint code {
  font-size: 12px;
}

.drawer-body {
  min-height: 120px;
}

.skill-meta {
  margin-bottom: 16px;
}

.json-block-wrap {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  overflow: hidden;
}

.json-block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.json-pre {
  margin: 0;
  padding: 12px;
  max-height: min(68vh, 640px);
  overflow: auto;
  font-size: 12px;
  line-height: 1.5;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
}

.asset-actions {
  margin-top: 16px;
  display: flex;
  gap: 12px;
}

:deep(.el-table__row) {
  cursor: pointer;
}
</style>
