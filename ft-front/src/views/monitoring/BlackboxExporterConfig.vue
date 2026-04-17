<template>
  <div class="blackbox-exporter-config">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>Blackbox Exporter 配置管理</h2>
          <el-button type="primary" @click="handleAddConfig">
            <el-icon><Plus /></el-icon>
            添加配置
          </el-button>
        </div>
      </template>

      <!-- 配置列表 -->
      <el-table
        v-loading="loading"
        :data="blackboxExporterConfigs"
        style="width: 100%"
        border
        stripe
      >
        <el-table-column prop="name" label="配置名称" min-width="150" />
        <el-table-column prop="description" label="描述" min-width="200" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column prop="configFile" label="配置文件" min-width="200" />
        <el-table-column prop="enabled" label="状态" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.enabled"
              @change="handleStatusChange(scope.row)"
              :disabled="loading"
            />
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="180" />
        <el-table-column prop="updateTime" label="更新时间" width="180" />
        <el-table-column label="操作" width="150" fixed="right">
          <template #default="scope">
            <el-button
              type="primary"
              size="small"
              @click="handleEditConfig(scope.row)"
              :disabled="loading"
            >
              编辑
            </el-button>
            <el-button
              type="danger"
              size="small"
              @click="handleDeleteConfig(scope.row)"
              :disabled="loading"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 配置表单对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="formTitle"
      width="60%"
      :before-close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-position="top"
        label-width="100px"
        size="large"
      >
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="配置名称" prop="name">
              <el-input
                v-model="formData.name"
                placeholder="请输入配置名称"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="描述">
              <el-input
                v-model="formData.description"
                placeholder="请输入配置描述"
                clearable
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="端口" prop="port">
              <el-input-number
                v-model="formData.port"
                :min="1"
                :max="65535"
                :step="1"
                placeholder="请输入端口号"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="配置文件" prop="configFile">
              <el-input
                v-model="formData.configFile"
                placeholder="请输入配置文件路径"
                clearable
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="handleDialogClose">取消</el-button>
          <el-button type="primary" @click="handleFormSubmit" :loading="formLoading">
            提交
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { useMonitoringStore } from '../../stores/monitoring'
import type { BlackboxExporterConfig } from '../../types/monitoring'

// 状态
const loading = ref(false)
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(0)

// 对话框状态
const dialogVisible = ref(false)
const formRef = ref()
const formLoading = ref(false)
const formData = reactive({
  name: '',
  description: '',
  enabled: true,
  port: 9115,
  configFile: '/etc/blackbox-exporter/config.yml'
})
const formRules = reactive({
  name: [
    { required: true, message: '请输入配置名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  port: [
    { required: true, message: '请输入端口号', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: '端口号必须在 1-65535 之间', trigger: 'blur' }
  ],
  configFile: [
    { required: true, message: '请输入配置文件路径', trigger: 'blur' }
  ]
})

// 当前编辑的配置ID
const currentConfigId = ref<string | null>(null)

// 计算属性
const formTitle = computed(() => {
  return currentConfigId.value ? '编辑 Blackbox Exporter 配置' : '新增 Blackbox Exporter 配置'
})

// 筛选 Blackbox Exporter 配置（全部）
const allBlackboxExporterConfigs = computed(() => {
  return monitoringStore.configs.filter(config => config.type === 'blackbox-exporter') as BlackboxExporterConfig[]
})

// 分页后的配置
const blackboxExporterConfigs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return allBlackboxExporterConfigs.value.slice(start, end)
})

// Store
const monitoringStore = useMonitoringStore()

// 生命周期钩子
onMounted(() => {
  fetchConfigs()
})

// 获取配置列表
const fetchConfigs = async () => {
  loading.value = true
  try {
    await monitoringStore.fetchConfigs()
    total.value = allBlackboxExporterConfigs.value.length
  } catch (error) {
    ElMessage.error('获取配置列表失败')
  } finally {
    loading.value = false
  }
}

// 处理状态变更
const handleStatusChange = async (row: BlackboxExporterConfig) => {
  loading.value = true
  try {
    await monitoringStore.updateConfig(row.id, {
      enabled: row.enabled
    })
    ElMessage.success('状态更新成功')
  } catch (error) {
    ElMessage.error('状态更新失败')
    // 恢复原状态
    row.enabled = !row.enabled
  } finally {
    loading.value = false
  }
}

// 处理添加配置
const handleAddConfig = () => {
  resetForm()
  currentConfigId.value = null
  dialogVisible.value = true
}

// 处理编辑配置
const handleEditConfig = (row: BlackboxExporterConfig) => {
  currentConfigId.value = row.id
  // 复制配置数据到表单
  Object.assign(formData, JSON.parse(JSON.stringify(row)))
  dialogVisible.value = true
}

// 处理删除配置
const handleDeleteConfig = (row: BlackboxExporterConfig) => {
  ElMessageBox.confirm('确定要删除此配置吗？', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    loading.value = true
    try {
      await monitoringStore.removeConfig(row.id)
      ElMessage.success('删除成功')
    } catch (error) {
      ElMessage.error('删除失败')
    } finally {
      loading.value = false
    }
  }).catch(() => {
    // 取消删除
  })
}

// 处理表单提交
const handleFormSubmit = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    formLoading.value = true
    
    const configData = {
      ...formData,
      type: 'blackbox-exporter'
    }
    
    if (currentConfigId.value) {
      // 更新配置
      await monitoringStore.updateConfig(currentConfigId.value, configData)
      ElMessage.success('配置更新成功')
    } else {
      // 创建配置
      await monitoringStore.addConfig(configData)
      ElMessage.success('配置创建成功')
    }
    
    dialogVisible.value = false
    fetchConfigs()
  } catch (error) {
    // 表单验证失败或提交失败
  } finally {
    formLoading.value = false
  }
}

// 处理对话框关闭
const handleDialogClose = () => {
  resetForm()
  dialogVisible.value = false
}

// 重置表单
const resetForm = () => {
  if (formRef.value) {
    formRef.value.resetFields()
  }
  
  // 重置表单数据
  Object.assign(formData, {
    name: '',
    description: '',
    enabled: true,
    port: 9115,
    configFile: '/etc/blackbox-exporter/config.yml'
  })
  
  currentConfigId.value = null
}

// 分页处理
const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
  fetchConfigs()
}

const handleCurrentChange = (current: number) => {
  currentPage.value = current
  fetchConfigs()
}
</script>

<style scoped>
.blackbox-exporter-config {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h2 {
  margin: 0;
  font-size: 20px;
}

.pagination-container {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
