<template>
  <div class="backup-restore">
    <div class="page-header">
      <h2>备份与恢复</h2>
    </div>
    <el-alert
      class="billing-alert"
      type="warning"
      show-icon
      :closable="false"
      :title="billingAlertTitle"
    />
    
    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-input
        v-model="filters.name"
        placeholder="搜索备份名称"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
      <el-select
        v-model="filters.status"
        placeholder="选择状态"
        clearable
        @change="handleSearch"
        class="filter-select"
      >
        <el-option label="已完成" value="completed" />
        <el-option label="运行中" value="running" />
        <el-option label="失败" value="failed" />
      </el-select>
      
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        format="YYYY-MM-DD"
        value-format="YYYY-MM-DD"
        @change="handleDateChange"
        class="date-picker"
      />
      
      <el-button type="primary" @click="handleSearch">
        <el-icon><Search /></el-icon>
        搜索
      </el-button>
      
      <el-button @click="handleReset">
        <el-icon><RefreshRight /></el-icon>
        重置
      </el-button>
      
      <el-button v-if="canManageBackups" type="success" :disabled="!canRunBackupActions" @click="handleCreateBackup">
        <el-icon><Plus /></el-icon>
        创建备份
      </el-button>

      <el-button v-if="canManageBackups && needsBackupSubscription" type="warning" @click="goSubscribe">
        订阅解锁
      </el-button>
    </div>
    
    <!-- 备份列表表格 -->
    <div class="backup-table">
      <el-table
        v-loading="loading"
        :data="backupsList"
        stripe
        border
        @selection-change="handleSelectionChange"
        style="width: 100%"
        row-key="id"
      >
        <el-table-column v-if="canRunBackupActions" type="selection" min-width="40" />
        
        <el-table-column prop="id" label="ID" min-width="60" align="center" />
        
        <el-table-column prop="name" label="备份名称" min-width="150" align="left" />
        
        <el-table-column prop="description" label="描述" min-width="200" align="left">
          <template #default="scope">
            <el-tooltip effect="dark" :content="scope.row.description" placement="top">
              <span class="description" style="display: inline-block; width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                {{ scope.row.description || '无描述' }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <el-table-column prop="size" label="大小" min-width="120" align="center">
          <template #default="scope">
            {{ formatFileSize(scope.row.size) }}
          </template>
        </el-table-column>
        
        <el-table-column prop="status" label="状态" min-width="100" align="center">
          <template #default="scope">
            <el-tag
              :type="getStatusType(scope.row.status)"
              size="small"
            >
              {{ getStatusText(scope.row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="backupTime" label="备份时间" min-width="180" align="center" />
        
        <el-table-column prop="createTime" label="创建时间" min-width="180" align="center" />
        
        <el-table-column label="操作" min-width="180" align="center">
          <template #default="scope">
            <el-button
              v-if="canManageBackups"
              size="small"
              type="primary"
              @click="handleRestoreBackup(scope.row.id)"
              :icon="RefreshRight"
              title="恢复"
              :disabled="!canRunBackupActions || scope.row.status !== 'completed'"
            >
            </el-button>
            
            <el-button
              v-if="canManageBackups"
              type="danger"
              size="small"
              @click="handleDeleteBackup(scope.row.id)"
              :icon="Delete"
              title="删除"
              :disabled="!canRunBackupActions"
            >
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    
    <!-- 批量操作 -->
    <div class="batch-actions" v-if="canRunBackupActions && selectedIds.length > 0">
      <el-button type="danger" @click="handleBatchDelete">
        <el-icon><Delete /></el-icon>
        批量删除 ({{ selectedIds.length }})
      </el-button>
    </div>
    
    <!-- 分页 -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
    
    <!-- 创建备份对话框 -->
    <el-dialog
      v-if="canRunBackupActions"
      v-model="createDialogVisible"
      title="创建备份"
      width="400px"
    >
      <el-form
        ref="createBackupFormRef"
        :model="createBackupForm"
        :rules="createBackupRules"
        label-position="top"
      >
        <el-form-item label="备份名称" prop="name">
          <el-input
            v-model="createBackupForm.name"
            placeholder="请输入备份名称"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="createBackupForm.description"
            placeholder="请输入备份描述"
            type="textarea"
            :rows="3"
            clearable
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="createDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="createDialogLoading"
            @click="handleCreateDialogSubmit"
          >
            确认创建
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, RefreshRight, Plus, Delete } from '@element-plus/icons-vue'
import { getBackups, createBackup, restoreBackup, deleteBackup, batchDeleteBackups } from '../../api/advanced'
import { createCheckoutSession, getBillingCapabilities, type BillingCapabilityFeature } from '../../api/billing'
import type { Backup, BackupParams, BackupListResponse } from '../../types'

// 响应式状态
const loading = ref(false)
const createDialogVisible = ref(false)
const createDialogLoading = ref(false)
const createBackupFormRef = ref()
const selectedIds = ref<number[]>([])
const currentRole = computed(() => {
  try {
    return (JSON.parse(localStorage.getItem('userInfo') || '{}') as { role?: string }).role || ''
  } catch {
    return ''
  }
})
const canManageBackups = computed(() => currentRole.value === 'admin' || currentRole.value === 'super_admin')
const backupCapability = ref<BillingCapabilityFeature | null>(null)
const backupPackKey = computed(() => backupCapability.value?.pack_key || 'pack.backup_performance')
const canUseBackupData = computed(() => backupCapability.value?.can_execute ?? true)
const canRunBackupActions = computed(() => canManageBackups.value && canUseBackupData.value)
const needsBackupSubscription = computed(() => {
  const state = backupCapability.value?.execute_state as Record<string, unknown> | undefined
  return !canUseBackupData.value && String(state?.biz || '').startsWith('PAYWALL_')
})
const billingAlertTitle = computed(() => {
  if (!backupCapability.value) {
    return '订阅功能包：pack.backup_performance。未订阅用户可查看页面与配置说明，真实备份数据与执行动作会在后端校验订阅。'
  }
  if (canUseBackupData.value) {
    return `当前账号可使用 ${backupPackKey.value}；创建、恢复、删除仍仅限管理员角色。`
  }
  const state = backupCapability.value.execute_state as Record<string, unknown> | undefined
  return String(state?.msg || `可预览，真实备份数据与执行动作需订阅 ${backupPackKey.value}。`)
})

// 筛选条件
const filters = reactive<BackupParams>({
  page: 1,
  pageSize: 20,
  name: '',
  status: '',
  startDate: '',
  endDate: ''
})

const dateRange = ref<[string, string] | null>(null)

// 备份列表数据
const backupsList = ref<Backup[]>([])
const total = ref(0)

// 创建备份表单
const createBackupForm = reactive({
  name: '',
  description: ''
})

// 创建备份表单验证规则
const createBackupRules = reactive({
  name: [
    { required: true, message: '请输入备份名称', trigger: 'blur' },
    { min: 2, max: 50, message: '备份名称长度在 2 到 50 个字符', trigger: 'blur' }
  ]
})

// 加载备份列表
onMounted(async () => {
  await loadBackupCapability()
  if (canUseBackupData.value) {
    fetchBackups()
  }
})

const loadBackupCapability = async () => {
  try {
    const data = await getBillingCapabilities()
    backupCapability.value = (data.features || []).find((item) => item.feature_key === 'feature.backup_performance') || null
  } catch {
    backupCapability.value = null
  }
}

// 获取备份列表
const fetchBackups = async () => {
  if (!canUseBackupData.value) {
    backupsList.value = []
    total.value = 0
    return
  }
  loading.value = true
  try {
    const response: BackupListResponse = await getBackups(filters)
    backupsList.value = response.list
    total.value = response.total
  } catch (error) {
    console.error('获取备份列表失败:', error)
    ElMessage.error('获取备份列表失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  if (!canUseBackupData.value) {
    ElMessage.warning('订阅后可查看真实备份数据')
    return
  }
  filters.page = 1
  fetchBackups()
}

// 处理日期范围变化
const handleDateChange = (val: [string, string] | null) => {
  if (val) {
    filters.startDate = val[0]
    filters.endDate = val[1]
  } else {
    filters.startDate = ''
    filters.endDate = ''
  }
  filters.page = 1
  fetchBackups()
}

// 处理重置
const handleReset = () => {
  Object.assign(filters, {
    page: 1,
    pageSize: 20,
    name: '',
    status: '',
    startDate: '',
    endDate: ''
  })
  dateRange.value = null
  fetchBackups()
}

// 处理分页大小变化
const handleSizeChange = (size: number) => {
  filters.pageSize = size
  filters.page = 1
  fetchBackups()
}

// 处理当前页变化
const handleCurrentChange = (current: number) => {
  filters.page = current
  fetchBackups()
}

// 处理选择变化
const handleSelectionChange = (selection: Backup[]) => {
  selectedIds.value = selection.map(item => item.id)
}

// 处理创建备份
const handleCreateBackup = () => {
  if (!canRunBackupActions.value) {
    void goSubscribe()
    return
  }
  resetCreateBackupForm()
  createDialogVisible.value = true
}

const goSubscribe = async () => {
  try {
    const resp = await createCheckoutSession({ pack_key: backupPackKey.value })
    if (resp.url) {
      window.location.href = resp.url
    } else {
      ElMessage.info('当前账号无需订阅')
    }
  } catch {
    /* 拦截器已提示 */
  }
}

// 处理恢复备份
const handleRestoreBackup = (id: number) => {
  ElMessageBox.confirm('确定要恢复该备份吗？此操作可能会覆盖当前数据。', '警告', {
    confirmButtonText: '确定恢复',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await restoreBackup(id)
      ElMessage.success('恢复备份成功')
      fetchBackups()
    } catch (error) {
      console.error('恢复备份失败:', error)
      ElMessage.error('恢复备份失败')
    }
  }).catch(() => {
    // 取消恢复
  })
}

// 处理删除备份
const handleDeleteBackup = (id: number) => {
  ElMessageBox.confirm('确定要删除该备份吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await deleteBackup(id)
      ElMessage.success('删除成功')
      selectedIds.value = selectedIds.value.filter(item => item !== id)
      fetchBackups()
    } catch (error) {
      console.error('删除备份失败:', error)
      ElMessage.error('删除备份失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理批量删除
const handleBatchDelete = () => {
  ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 个备份吗？`, '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await batchDeleteBackups(selectedIds.value)
      ElMessage.success('批量删除成功')
      selectedIds.value = []
      fetchBackups()
    } catch (error) {
      console.error('批量删除备份失败:', error)
      ElMessage.error('批量删除备份失败')
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理创建备份对话框提交
const handleCreateDialogSubmit = async () => {
  if (!createBackupFormRef.value) return
  
  try {
    await createBackupFormRef.value.validate()
    createDialogLoading.value = true
    
    await createBackup(createBackupForm)
    ElMessage.success('备份创建成功')
    createDialogVisible.value = false
    resetCreateBackupForm()
    fetchBackups()
  } catch (error) {
    console.error('创建备份失败:', error)
    ElMessage.error('创建备份失败')
  } finally {
    createDialogLoading.value = false
  }
}

// 重置创建备份表单
const resetCreateBackupForm = () => {
  createBackupForm.name = ''
  createBackupForm.description = ''
  
  if (createBackupFormRef.value) {
    createBackupFormRef.value.resetFields()
  }
}

// 获取状态类型
const getStatusType = (status: string) => {
  switch (status) {
    case 'completed':
      return 'success'
    case 'running':
      return 'primary'
    case 'failed':
      return 'danger'
    default:
      return 'info'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'completed':
      return '已完成'
    case 'running':
      return '运行中'
    case 'failed':
      return '失败'
    default:
      return status
  }
}

// 格式化文件大小
const formatFileSize = (size: number): string => {
  if (size === 0) return '0 Bytes'
  
  const k = 1024
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(size) / Math.log(k))
  
  return parseFloat((size / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}
</script>

<style scoped>
.backup-restore {
  padding: 0 20px 20px 20px;
}

.page-header h2 {
  margin: 0 0 20px 0;
  color: var(--el-color-primary);
  font-size: 30px;
  font-weight: 600;
}

.billing-alert {
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
  gap: 15px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.search-input {
  width: 250px;
}

.filter-select {
  width: 120px;
}

.date-picker {
  width: 320px;
}

.backup-table {
  margin-bottom: 20px;
  width: 100%;
}

.batch-actions {
  margin-top: 20px;
  padding: 15px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.pagination {
  text-align: center;
  margin-bottom: 20px;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
