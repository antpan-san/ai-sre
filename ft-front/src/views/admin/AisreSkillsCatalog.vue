<template>
  <div class="aisre-skills page-shell page-shell--crud-wide">
    <div class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">ai-sre 技能包</h2>
        <p class="page-desc--muted">展示当前后端为客户端 ai-sre 加载的技能包（内置与生成版）。点击行可查看完整 YAML 等价结构。</p>
      </div>
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
      <el-table :data="filteredRows" stripe border size="small" empty-text="暂无技能包" @row-click="onRowClick">
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
            <el-button type="primary" link size="small" @click.stop="openDetail(row.name)">查看</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-drawer v-model="detailOpen" :title="detailTitle" size="min(720px, 92vw)" destroy-on-close>
      <div v-loading="detailLoading" class="drawer-body">
        <template v-if="detailSkill">
          <el-descriptions :column="1" border size="small" class="skill-meta">
            <el-descriptions-item label="名称">{{ detailSkill.pack.name }}</el-descriptions-item>
            <el-descriptions-item label="展示名">{{ detailSkill.pack.display_name }}</el-descriptions-item>
            <el-descriptions-item label="来源">{{ detailSkill.source }}</el-descriptions-item>
            <el-descriptions-item label="版本">{{ detailSkill.version }}</el-descriptions-item>
            <el-descriptions-item v-if="detailSkill.path" label="路径">{{ detailSkill.path }}</el-descriptions-item>
          </el-descriptions>
          <div class="json-block-wrap">
            <div class="json-block-head">
              <span>技能包内容（JSON）</span>
              <el-button size="small" @click="copyDetailJson">复制</el-button>
            </div>
            <pre class="json-pre">{{ detailJson }}</pre>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getAdminAiSkillDetail, getAdminAiSkills, type RegisteredSkill, type SkillSummary } from '../../api/aiSkills'
import { copyTextToClipboard } from '../../utils/clipboard'

const loading = ref(false)
const rows = ref<SkillSummary[]>([])
const dataDir = ref('')
const keyword = ref('')

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailSkill = ref<RegisteredSkill | null>(null)
const detailTitle = ref('技能包详情')

const filteredRows = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return rows.value
  return rows.value.filter((r) => {
    const blob = [r.name, r.display_name, ...(r.topics || [])].join(' ').toLowerCase()
    return blob.includes(q)
  })
})

const detailJson = computed(() => {
  if (!detailSkill.value) return ''
  try {
    return JSON.stringify(detailSkill.value.pack, null, 2)
  } catch {
    return ''
  }
})

const load = async () => {
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

const openDetail = async (name: string) => {
  detailTitle.value = name
  detailSkill.value = null
  detailOpen.value = true
  detailLoading.value = true
  try {
    const data = await getAdminAiSkillDetail(name)
    detailSkill.value = data.skill
    const dn = data.skill?.pack?.display_name
    if (dn) detailTitle.value = `${dn} (${name})`
  } catch {
    detailOpen.value = false
  } finally {
    detailLoading.value = false
  }
}

const onRowClick = (row: SkillSummary) => {
  void openDetail(row.name)
}

const copyDetailJson = async () => {
  if (!detailJson.value) return
  try {
    await copyTextToClipboard(detailJson.value)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

onMounted(() => {
  void load()
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

.skills-filter {
  width: min(320px, 100%);
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

:deep(.el-table__row) {
  cursor: pointer;
}
</style>
