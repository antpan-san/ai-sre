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
      <el-tab-pane label="能力树" name="tree">
        <div class="tab-toolbar">
          <el-button :loading="treeLoading" @click="loadSkillTree">刷新</el-button>
          <span v-if="treeRev" class="tree-rev page-desc--muted">
            版本：{{ treeRev }}<template v-if="treeSource">（{{ treeSource }}）</template>
          </span>
        </div>
        <el-card shadow="never" v-loading="treeLoading">
          <el-table
            :data="treeRows"
            row-key="path"
            stripe
            border
            size="small"
            default-expand-all
            empty-text="暂无能力树"
            @row-click="onTreeRowClick"
          >
            <el-table-column prop="title" label="能力节点" min-width="240" show-overflow-tooltip />
            <el-table-column prop="path" label="路径" min-width="300" show-overflow-tooltip />
            <el-table-column label="类型" width="110" align="center">
              <template #default="{ row }">
                <el-tag :type="nodeTypeTag(row.node_type)" size="small">{{ nodeTypeLabel(row.node_type) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="资产" width="150" align="center">
              <template #default="{ row }">
                <span class="asset-stat">{{ treeAssetStatText(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="topic" label="Topic" width="110" show-overflow-tooltip />
            <el-table-column prop="problem_key" label="问题模式" width="150" show-overflow-tooltip />
            <el-table-column prop="capability_key" label="能力键" min-width="190" show-overflow-tooltip />
            <el-table-column prop="pack_key" label="订阅包" min-width="170" show-overflow-tooltip />
            <el-table-column prop="execution_mode" label="执行模式" min-width="170" show-overflow-tooltip />
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="row.status === 'disabled' ? 'info' : 'success'" size="small">
                  {{ row.status === 'disabled' ? '停用' : '启用' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="CLI" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="row.cli_visible ? 'success' : 'info'" size="small">
                  {{ row.cli_visible ? '可见' : '隐藏' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-tab-pane>

      <el-tab-pane label="商品包" name="commercial">
        <div class="tab-toolbar">
          <el-button :loading="commercialLoading" @click="loadCommercial">刷新</el-button>
          <span v-if="commercialPolicyRev" class="page-desc--muted">policy_rev: {{ commercialPolicyRev }}</span>
        </div>
        <el-card shadow="never" v-loading="commercialLoading">
          <h4 class="section-title">领域包（skillpack.* / pack.*）</h4>
          <el-table :data="commercialProducts" stripe border size="small" empty-text="暂无商品包">
            <el-table-column prop="product_key" label="商品键" min-width="200" />
            <el-table-column prop="title" label="标题" min-width="180" />
            <el-table-column prop="product_type" label="类型" width="100" />
            <el-table-column prop="price_hint" label="价格提示" width="120" show-overflow-tooltip />
          </el-table>
          <h4 class="section-title bindings-title">树节点绑定</h4>
          <el-table :data="commercialBindings" stripe border size="small" empty-text="暂无绑定">
            <el-table-column prop="product_key" label="商品包" min-width="180" />
            <el-table-column prop="node_path" label="节点路径" min-width="280" show-overflow-tooltip />
            <el-table-column prop="grant_scope" label="范围" width="100" />
            <el-table-column prop="pack_key" label="pack_key" min-width="160" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-tab-pane>

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
          <el-select v-model="reviewTopic" clearable placeholder="Topic" class="review-topic-select" @change="loadReview">
            <el-option label="k8s" value="k8s" />
            <el-option label="go_runtime" value="go_runtime" />
            <el-option label="redis" value="redis" />
            <el-option label="kafka" value="kafka" />
            <el-option label="nginx" value="nginx" />
            <el-option label="mysql" value="mysql" />
            <el-option label="elasticsearch" value="elasticsearch" />
          </el-select>
          <el-input
            v-model="reviewCreatedBy"
            clearable
            class="review-created-by"
            placeholder="提交人"
            @clear="loadReview"
            @keyup.enter="loadReview"
          />
          <el-tag v-if="selectedTreeFilter" closable type="info" @close="clearTreeFilter">
            {{ selectedTreeFilter.title }}
          </el-tag>
          <el-button :loading="reviewLoading" @click="loadReview">刷新</el-button>
        </div>
        <el-card shadow="never" v-loading="reviewLoading">
          <el-table :data="reviewRows" stripe border size="small" empty-text="暂无技能资产">
            <el-table-column prop="display_name" label="展示名" min-width="140" show-overflow-tooltip />
            <el-table-column prop="topic" label="Topic" width="90" />
            <el-table-column prop="problem_key" label="问题模式" width="140" show-overflow-tooltip />
            <el-table-column prop="category_path" label="能力路径" min-width="220" show-overflow-tooltip />
            <el-table-column prop="created_by" label="提交人" width="100" show-overflow-tooltip />
            <el-table-column prop="risk_level" label="风险" width="80" align="center" />
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

      <el-tab-pane label="运营统计" name="usage">
        <div class="tab-toolbar">
          <el-select v-model="usageDays" class="review-status-select" @change="loadUsage">
            <el-option label="近 7 天" :value="7" />
            <el-option label="近 30 天" :value="30" />
            <el-option label="近 90 天" :value="90" />
          </el-select>
          <el-button :loading="usageLoading" @click="loadUsage">刷新</el-button>
          <el-link type="primary" :href="usageCsvHref" target="_blank">导出 CSV</el-link>
        </div>
        <el-card shadow="never" v-loading="usageLoading">
          <h4 class="section-title">诊断任务单</h4>
          <el-table :data="usageStats?.diagnostic_plans || []" stripe border size="small" empty-text="暂无数据">
            <el-table-column prop="label" label="坐标" min-width="240" />
            <el-table-column prop="status" label="状态" width="120" />
            <el-table-column prop="count" label="次数" width="100" align="right" />
          </el-table>
          <h4 class="section-title bindings-title">AI 调用</h4>
          <el-table :data="usageStats?.ai_executions || []" stripe border size="small" empty-text="暂无数据">
            <el-table-column prop="label" label="类别" min-width="200" />
            <el-table-column prop="status" label="状态" width="120" />
            <el-table-column prop="count" label="次数" width="100" align="right" />
          </el-table>
          <h4 class="section-title bindings-title">资产与审核</h4>
          <el-row :gutter="16">
            <el-col :span="12">
              <el-table :data="usageStats?.skill_assets || []" stripe border size="small" empty-text="暂无">
                <el-table-column prop="label" label="Topic" />
                <el-table-column prop="status" label="状态" width="100" />
                <el-table-column prop="count" label="数" width="80" align="right" />
              </el-table>
            </el-col>
            <el-col :span="12">
              <el-table :data="usageStats?.reviews || []" stripe border size="small" empty-text="暂无">
                <el-table-column prop="label" label="动作" />
                <el-table-column prop="count" label="次数" width="100" align="right" />
              </el-table>
            </el-col>
          </el-row>
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
            <el-descriptions-item v-if="assetDetail.category_path" label="能力路径">
              {{ assetDetail.category_path }}
            </el-descriptions-item>
            <el-descriptions-item v-if="assetDetail.skill_key" label="Skill Key">
              {{ assetDetail.skill_key }}
            </el-descriptions-item>
            <el-descriptions-item v-if="assetDetail.problem_key" label="问题模式">
              {{ assetDetail.problem_key }}
            </el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="assetStatusTag(assetDetail.status)" size="small">{{ assetStatusLabel(assetDetail.status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="提交人">{{ assetDetail.created_by || '—' }}</el-descriptions-item>
            <el-descriptions-item v-if="assetDetail.observation_summary" label="观察摘要">
              {{ assetDetail.observation_summary }}
            </el-descriptions-item>
            <el-descriptions-item v-if="assetDetail.published_pack_path" label="发布路径">
              {{ assetDetail.published_pack_path }}
            </el-descriptions-item>
          </el-descriptions>
          <div v-if="assetApproveDiff" class="diff-block">
            <h4 class="section-title">发布预览</h4>
            <p class="page-desc--muted">
              {{ assetApproveDiff.generated_pack_name }}
              <template v-if="assetApproveDiff.registry_pack_name">
                · 注册表 {{ assetApproveDiff.registry_pack_name }}
              </template>
              · 变更 {{ (assetApproveDiff.fields_changed || []).join(', ') || '—' }}
            </p>
          </div>
          <div v-if="assetReviews.length" class="diff-block">
            <h4 class="section-title">审核记录</h4>
            <el-table :data="assetReviews" size="small" border stripe>
              <el-table-column prop="action" label="动作" width="90" />
              <el-table-column prop="actor_name" label="操作人" width="100" />
              <el-table-column prop="notes" label="备注" min-width="140" show-overflow-tooltip />
              <el-table-column prop="created_at" label="时间" width="170" />
            </el-table>
          </div>
          <div class="json-block-wrap">
            <div class="json-block-head">
              <span>版本内容</span>
              <el-button size="small" @click="copyAssetJson">复制</el-button>
            </div>
            <pre class="json-pre">{{ assetDetailJson }}</pre>
          </div>
          <div v-if="assetDetail.status === 'review'" class="asset-actions">
            <el-checkbox v-model="approveMergeRegistry" @change="loadAssetDiff(assetDetail.id)">
              与注册表同 topic 技能合并发布
            </el-checkbox>
            <el-button type="success" :loading="approvingId === assetDetail.id" @click="onApprove(assetDetail)">
              审核通过并发布
            </el-button>
            <el-button type="danger" :loading="rejectingId === assetDetail.id" @click="onReject(assetDetail)">
              驳回
            </el-button>
          </div>
          <div v-else-if="assetDetail.status === 'approved'" class="asset-actions">
            <el-button type="warning" @click="onDeprecate(assetDetail)">下架</el-button>
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
  deprecateAdminSkillAsset,
  exportAdminSkillUsageCSV,
  getAdminSkillAssetApproveDiff,
  getAdminSkillTree,
  getAdminSkillAsset,
  getAdminSkillUsageSummary,
  listAdminSkillAssetReviews,
  listAdminSkillAssets,
  rejectAdminSkillAsset,
  type SkillApproveDiff,
  type SkillAssetDetail,
  type SkillAssetListItem,
  type SkillAssetReviewRow,
  type SkillTreeNode,
  type SkillUsageSummary
} from '../../api/skillAssets'
import {
  listCommercialBindings,
  listCommercialProducts,
  type CommercialProduct,
  type ProductNodeBinding
} from '../../api/skillCommercial'
import { copyTextToClipboard } from '../../utils/clipboard'

const activeTab = ref('registry')

const loading = ref(false)
const rows = ref<SkillSummary[]>([])
const dataDir = ref('')
const keyword = ref('')

type SkillTreeTableNode = SkillTreeNode & { children?: SkillTreeTableNode[] }

const treeLoading = ref(false)
const treeRev = ref('')
const treeSource = ref('')
const treeRows = ref<SkillTreeTableNode[]>([])
const selectedTreeFilter = ref<SkillTreeNode | null>(null)

const commercialLoading = ref(false)
const commercialProducts = ref<CommercialProduct[]>([])
const commercialBindings = ref<ProductNodeBinding[]>([])
const commercialPolicyRev = ref('')

const reviewLoading = ref(false)
const reviewRows = ref<SkillAssetListItem[]>([])
const reviewTotal = ref(0)
const reviewPage = ref(1)
const reviewPageSize = ref(20)
const reviewStatus = ref('review')
const reviewTopic = ref('')
const reviewCreatedBy = ref('')
const reviewPendingCount = ref(0)
const approvingId = ref('')
const rejectingId = ref('')
const approveMergeRegistry = ref(true)
const assetApproveDiff = ref<SkillApproveDiff | null>(null)
const assetReviews = ref<SkillAssetReviewRow[]>([])

const usageLoading = ref(false)
const usageDays = ref(30)
const usageStats = ref<SkillUsageSummary | null>(null)
const usageCsvHref = computed(() => exportAdminSkillUsageCSV(usageDays.value))

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

const nodeTypeLabel = (s: string) => {
  switch (s) {
    case 'category':
      return '大类'
    case 'capability':
      return '能力'
    case 'skill':
      return '技能'
    default:
      return s || '—'
  }
}

const nodeTypeTag = (s: string) => {
  switch (s) {
    case 'category':
      return 'info'
    case 'capability':
      return 'warning'
    case 'skill':
      return 'success'
    default:
      return 'info'
  }
}

const treeAssetStatText = (row: SkillTreeNode) => {
  const s = row.asset_stats
  if (!s || !s.total) return '—'
  return `审${s.review || 0} / 发${s.approved || 0} / 驳${s.deprecated || 0}`
}

const buildSkillTreeRows = (nodes: SkillTreeNode[]): SkillTreeTableNode[] => {
  const byPath = new Map<string, SkillTreeTableNode>()
  const sorted = [...nodes].sort((a, b) => {
    const ao = a.sort_order || 0
    const bo = b.sort_order || 0
    if (ao !== bo) return ao - bo
    return (a.path || '').localeCompare(b.path || '')
  })
  sorted.forEach((n) => {
    byPath.set(n.path, { ...n, children: [] })
  })
  const roots: SkillTreeTableNode[] = []
  sorted.forEach((n) => {
    const row = byPath.get(n.path)
    if (!row) return
    const parent = n.parent_path ? byPath.get(n.parent_path) : null
    if (parent) {
      parent.children = parent.children || []
      parent.children.push(row)
    } else {
      roots.push(row)
    }
  })
  const prune = (rows: SkillTreeTableNode[]) => {
    rows.forEach((row) => {
      if (row.children && row.children.length > 0) {
        prune(row.children)
      } else {
        delete row.children
      }
    })
  }
  prune(roots)
  return roots
}

const loadSkillTree = async () => {
  treeLoading.value = true
  try {
    const data = await getAdminSkillTree()
    treeRev.value = data.tree_rev || ''
    treeSource.value = data.tree_source || ''
    treeRows.value = buildSkillTreeRows(data.nodes || [])
  } catch {
    treeRev.value = ''
    treeSource.value = ''
    treeRows.value = []
  } finally {
    treeLoading.value = false
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
    const treeFilter = selectedTreeFilter.value
    const data = await listAdminSkillAssets({
      status: reviewStatus.value || undefined,
      topic: reviewTopic.value || undefined,
      created_by: reviewCreatedBy.value || undefined,
      category_path: treeFilter?.path || undefined,
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

const loadCommercial = async () => {
  commercialLoading.value = true
  try {
    const [prod, bind] = await Promise.all([listCommercialProducts(), listCommercialBindings()])
    commercialProducts.value = prod.products || []
    commercialPolicyRev.value = prod.policy_rev || ''
    commercialBindings.value = bind.bindings || []
  } catch {
    commercialProducts.value = []
    commercialBindings.value = []
  } finally {
    commercialLoading.value = false
  }
}

const onTabChange = (name: string | number) => {
  if (name === 'review') {
    void loadReview()
  } else if (name === 'tree') {
    void loadSkillTree()
  } else if (name === 'commercial') {
    void loadCommercial()
  } else if (name === 'usage') {
    void loadUsage()
  }
}

const onTreeRowClick = (row: SkillTreeNode) => {
  selectedTreeFilter.value = row
  reviewPage.value = 1
  activeTab.value = 'review'
  void loadReview()
}

const clearTreeFilter = () => {
  selectedTreeFilter.value = null
  reviewPage.value = 1
  void loadReview()
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

const loadAssetReviews = async (id: string) => {
  try {
    const data = await listAdminSkillAssetReviews(id)
    assetReviews.value = data.items || []
  } catch {
    assetReviews.value = []
  }
}

const loadAssetDiff = async (id: string) => {
  try {
    const data = await getAdminSkillAssetApproveDiff(id, approveMergeRegistry.value)
    assetApproveDiff.value = data.diff
  } catch {
    assetApproveDiff.value = null
  }
}

const openAssetDetail = async (id: string) => {
  assetDetail.value = null
  assetApproveDiff.value = null
  assetReviews.value = []
  assetDetailOpen.value = true
  assetDetailLoading.value = true
  try {
    const data = await getAdminSkillAsset(id)
    assetDetail.value = data.asset
    await Promise.all([loadAssetReviews(id), loadAssetDiff(id)])
  } catch {
    assetDetailOpen.value = false
  } finally {
    assetDetailLoading.value = false
  }
}

const loadUsage = async () => {
  usageLoading.value = true
  try {
    const data = await getAdminSkillUsageSummary(usageDays.value)
    usageStats.value = data.stats
  } catch {
    usageStats.value = null
  } finally {
    usageLoading.value = false
  }
}

const onApprove = async (row: SkillAssetListItem | SkillAssetDetail) => {
  let diffText = ''
  try {
    const { diff } = await getAdminSkillAssetApproveDiff(row.id, approveMergeRegistry.value)
    const changed = (diff.fields_changed || []).join(', ') || '无字段差异'
    diffText = `\n\n预览：${diff.generated_pack_name}`
    if (diff.registry_pack_name) {
      diffText += ` ← 合并 ${diff.registry_pack_name}（${diff.registry_source || 'registry'}）`
    }
    diffText += `\n变更字段：${changed}`
  } catch {
  }
  try {
    await ElMessageBox.confirm(
      `通过后将写入 generated 技能包（${approveMergeRegistry.value ? '与注册表合并' : '独立覆盖'}）。${diffText}`,
      '审核通过',
      { type: 'warning', confirmButtonText: '通过并发布', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  approvingId.value = row.id
  try {
    const res = await approveAdminSkillAsset(row.id, { merge_with_registry: approveMergeRegistry.value })
    ElMessage.success(res.merged ? `已合并发布至 ${res.path}` : `已发布至 ${res.path || 'generated'}`)
    assetDetailOpen.value = false
    await Promise.all([loadReview(), loadReviewPendingCount(), loadRegistry(), loadSkillTree()])
  } catch {
    ElMessage.error('审核失败')
  } finally {
    approvingId.value = ''
  }
}

const onDeprecate = async (row: SkillAssetListItem | SkillAssetDetail) => {
  let reason = ''
  try {
    const { value } = await ElMessageBox.prompt('下架已发布资产（不删除磁盘技能包）', '下架', {
      confirmButtonText: '下架',
      cancelButtonText: '取消',
      inputPlaceholder: '原因（可选）'
    })
    reason = value || ''
  } catch {
    return
  }
  try {
    await deprecateAdminSkillAsset(row.id, { reason })
    ElMessage.success('已下架')
    assetDetailOpen.value = false
    await Promise.all([loadReview(), loadSkillTree()])
  } catch {
    ElMessage.error('下架失败')
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
    await Promise.all([loadReview(), loadReviewPendingCount(), loadSkillTree()])
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
  void loadSkillTree()
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

.tree-rev {
  font-size: 13px;
}

.asset-stat {
  font-size: 12px;
  color: var(--el-text-color-regular);
  white-space: nowrap;
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

.section-title {
  margin: 0 0 12px;
  font-size: 14px;
  font-weight: 600;
}

.bindings-title {
  margin-top: 20px;
}

:deep(.el-table__row) {
  cursor: pointer;
}
</style>
