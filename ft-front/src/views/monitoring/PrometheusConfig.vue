<template>
  <div class="prometheus-config">
    <el-card>
      <template #header>
        <div class="card-header">
          <h2>Prometheus 配置管理</h2>
          <el-button type="primary" @click="handleAddConfig">
            <el-icon><Plus /></el-icon>
            添加配置
          </el-button>
        </div>
      </template>

      <!-- 配置列表 -->
      <el-table
        v-loading="loading"
        :data="prometheusConfigs"
        style="width: 100%"
        border
        stripe
      >
        <el-table-column prop="name" label="配置名称" min-width="150" />
        <el-table-column prop="description" label="描述" min-width="200" />
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
      width="70%"
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

        <el-form-item label="全局配置" prop="global">
          <el-card>
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="抓取间隔" prop="global.scrapeInterval">
                  <el-input
                    v-model="formData.global.scrapeInterval"
                    placeholder="例如: 15s"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="评估间隔" prop="global.evaluationInterval">
                  <el-input
                    v-model="formData.global.evaluationInterval"
                    placeholder="例如: 15s"
                    clearable
                  />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="抓取超时" prop="global.scrapeTimeout">
                  <el-input
                    v-model="formData.global.scrapeTimeout"
                    placeholder="例如: 10s"
                    clearable
                  />
                </el-form-item>
              </el-col>
            </el-row>
          </el-card>
        </el-form-item>

        <el-form-item label="告警规则文件">
          <el-input
            v-model="formData.ruleFilesText"
            type="textarea"
            placeholder="每行输入一个规则文件路径"
            :rows="3"
            @input="handleRuleFilesInput"
          />
        </el-form-item>
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
import type { PrometheusConfig } from '../../types/monitoring'

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
  global: {
    scrapeInterval: '15s',
    evaluationInterval: '15s',
    scrapeTimeout: '10s'
  },
  scrapeConfigs: [],
  alerting: {
    alertmanagers: []
  },
  ruleFiles: [] as string[],
  ruleFilesText: '' as string
})
const formRules = reactive({
  name: [
    { required: true, message: '请输入配置名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  'global.scrapeInterval': [
    { required: true, message: '请输入抓取间隔', trigger: 'blur' }
  ],
  'global.evaluationInterval': [
    { required: true, message: '请输入评估间隔', trigger: 'blur' }
  ],
  'global.scrapeTimeout': [
    { required: true, message: '请输入抓取超时', trigger: 'blur' }
  ]
})

// 当前编辑的配置ID
const currentConfigId = ref<string | null>(null)

// 计算属性
const formTitle = computed(() => {
  return currentConfigId.value ? '编辑 Prometheus 配置' : '新增 Prometheus 配置'
})

// 筛选 Prometheus 配置（全部）
const allPrometheusConfigs = computed(() => {
  return monitoringStore.configs.filter(config => config.type === 'prometheus') as PrometheusConfig[]
})

// 分页后的 Prometheus 配置
const prometheusConfigs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return allPrometheusConfigs.value.slice(start, end)
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
    total.value = allPrometheusConfigs.value.length
  } catch (error) {
    ElMessage.error('获取配置列表失败')
  } finally {
    loading.value = false
  }
}

// 处理规则文件输入
const handleRuleFilesInput = (value: string) => {
  formData.ruleFiles = value.split('\n').filter(line => line.trim())
}

// 处理状态变更
const handleStatusChange = async (row: PrometheusConfig) => {
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
const handleEditConfig = (row: PrometheusConfig) => {
  currentConfigId.value = row.id
  // 复制配置数据到表单
  Object.assign(formData, JSON.parse(JSON.stringify(row)))
  // 将规则文件数组转换为字符串
  formData.ruleFilesText = (row.ruleFiles || []).join('\n')
  dialogVisible.value = true
}

// 处理删除配置
const handleDeleteConfig = (row: PrometheusConfig) => {
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
      type: 'prometheus'
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
    global: {
      scrapeInterval: '15s',
      evaluationInterval: '15s',
      scrapeTimeout: '10s'
    },
    scrapeConfigs: [],
    alerting: {
      alertmanagers: []
    },
    ruleFiles: []
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
.prometheus-config {
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
