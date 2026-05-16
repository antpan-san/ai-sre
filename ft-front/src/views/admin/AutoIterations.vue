<template>
  <div class="auto-iterations page-shell page-shell--crud-wide">
    <div class="page-head">
      <div class="page-head-copy">
        <h2 class="page-title">自动迭代</h2>
        <p class="page-desc--muted">仅超级管理员可管理平台级自动迭代任务；CLI 用户仅可提交反馈。</p>
      </div>
      <div class="page-head-actions">
        <el-button :loading="loading" @click="loadAll">刷新</el-button>
        <el-button type="primary" @click="openManual">手动创建</el-button>
      </div>
    </div>

    <el-card shadow="never" class="settings-card" v-loading="settingsLoading">
      <template #header>全局开关</template>
      <el-form label-width="140px" size="small">
        <el-form-item label="启用自动迭代">
          <el-switch v-model="settings.enabled" @change="saveSettings" />
        </el-form-item>
        <el-form-item label="最大并发">
          <el-input-number v-model="settings.max_concurrent" :min="1" :max="20" @change="saveSettings" />
        </el-form-item>
        <el-form-item label="高风险需审批">
          <el-switch v-model="settings.high_risk_requires_approval" @change="saveSettings" />
        </el-form-item>
        <el-form-item v-if="settings.has_dingtalk_webhook" label="钉钉">
          <el-tag type="success" size="small">已配置</el-tag>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" v-loading="loading">
      <el-table :data="rows" stripe border size="small" empty-text="暂无任务" @row-click="goDetail">
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column prop="topic" label="Topic" width="100" />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column prop="risk_level" label="风险" width="80" />
        <el-table-column prop="source" label="来源" width="100" />
        <el-table-column prop="created_by" label="创建人" width="100" />
        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click.stop="goDetail(row)">详情</el-button>
          </template>
        </el-table-column>
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

    <el-dialog v-model="manualOpen" title="手动创建迭代" width="480px">
      <el-form label-width="80px">
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
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  createManualAutoIteration,
  getAutoIterationSettings,
  listAutoIterations,
  updateAutoIterationSettings,
  type AutoIteration,
  type AutoIterationSettings
} from '../../api/autoIterations'

const router = useRouter()
const loading = ref(false)
const settingsLoading = ref(false)
const creating = ref(false)
const rows = ref<AutoIteration[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const settings = reactive<AutoIterationSettings>({
  enabled: false,
  max_concurrent: 2,
  high_risk_requires_approval: true,
  has_dingtalk_webhook: false
})
const manualOpen = ref(false)
const manual = reactive({ title: '', topic: '', description: '' })

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
    ElMessage.success('设置已保存')
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
  } catch {
    rows.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const loadAll = () => Promise.all([loadSettings(), loadList()])

const goDetail = (row: AutoIteration | { id: string }) => {
  const id = typeof row === 'object' && 'id' in row ? row.id : ''
  if (id) router.push(`/admin/auto-iterations/${id}`)
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
    await createManualAutoIteration({ ...manual })
    manualOpen.value = false
    ElMessage.success('已创建')
    await loadList()
  } catch {
    ElMessage.error('创建失败')
  } finally {
    creating.value = false
  }
}

onMounted(() => void loadAll())
</script>

<style scoped>
.settings-card {
  margin-bottom: 16px;
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
</style>
