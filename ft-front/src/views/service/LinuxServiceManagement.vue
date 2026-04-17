<template>
  <div class="linux-service-management">
    <div class="page-header">
      <h2>Linux服务管理</h2>
    </div>
    
    <el-card class="filter-card">
      <el-form
        :model="searchForm"
        label-position="left"
        label-width="80px"
        inline
      >
        <el-form-item label="服务名称">
          <el-input
            v-model="searchForm.name"
            placeholder="请输入服务名称"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        
        <el-form-item label="状态">
          <el-select
            v-model="searchForm.status"
            placeholder="请选择服务状态"
            clearable
          >
            <el-option label="运行中" value="active" />
            <el-option label="已停止" value="inactive" />
            <el-option label="失败" value="failed" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="机器">
          <el-input
            v-model="searchForm.machineId"
            placeholder="请输入机器ID"
            clearable
          />
        </el-form-item>
        
        <el-form-item>
          <el-button
            type="primary"
            @click="handleSearch"
          >
            <el-icon><Search /></el-icon>
            查询
          </el-button>
          
          <el-button @click="handleReset">
            <el-icon><RefreshRight /></el-icon>
            重置
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>
    
    <el-card class="table-card">
      <div class="table-container">
        <el-table
          v-loading="serviceStore.linuxLoading"
          :data="serviceStore.linuxServiceList"
          stripe
          style="width: 100%"
          @selection-change="handleSelectionChange"
        >
          <el-table-column
            type="selection"
            width="55"
          />
          
          <el-table-column
            prop="name"
            label="服务名称"
            min-width="150"
          >
            <template #default="scope">
              <div class="service-name">{{ scope.row.name }}</div>
            </template>
          </el-table-column>
          
          <el-table-column
            prop="status"
            label="状态"
            width="100"
          >
            <template #default="scope">
              <el-tag
                :type="getStatusTagType(scope.row.status)"
                size="small"
              >
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          
          <el-table-column
            prop="description"
            label="描述"
            min-width="200"
          >
            <template #default="scope">
              <el-tooltip
                :content="scope.row.description"
                placement="top"
                effect="dark"
              >
                <div class="description-text">{{ scope.row.description }}</div>
              </el-tooltip>
            </template>
          </el-table-column>
          
          <el-table-column
            prop="machineName"
            label="所属机器"
            width="120"
          />
          
          <el-table-column
            prop="pid"
            label="PID"
            width="100"
          >
            <template #default="scope">
              {{ scope.row.pid || '-' }}
            </template>
          </el-table-column>
          
          <el-table-column
            prop="startCmd"
            label="启动命令"
            min-width="250"
          >
            <template #default="scope">
              <el-tooltip
                :content="scope.row.startCmd"
                placement="top"
                effect="dark"
              >
                <div class="start-cmd">{{ scope.row.startCmd }}</div>
              </el-tooltip>
            </template>
          </el-table-column>
          
          <el-table-column
            prop="createTime"
            label="创建时间"
            width="180"
          >
            <template #default="scope">
              <span>{{ formatDate(scope.row.createTime) }}</span>
            </template>
          </el-table-column>
          
          <el-table-column
            label="操作"
            width="180"
            fixed="right"
          >
            <template #default="scope">
              <el-button
                v-if="scope.row.status !== 'active'"
                type="success"
                size="small"
                @click="handleServiceOperation(scope.row, 'start')"
              >
                <el-icon><VideoPlay /></el-icon>
                启动
              </el-button>
              
              <el-button
                v-if="scope.row.status === 'active'"
                type="warning"
                size="small"
                @click="handleServiceOperation(scope.row, 'stop')"
              >
                <el-icon><CircleClose /></el-icon>
                停止
              </el-button>
              
              <el-button
                type="primary"
                size="small"
                @click="handleServiceOperation(scope.row, 'restart')"
              >
                <el-icon><RefreshRight /></el-icon>
                重启
              </el-button>
              
              <el-dropdown @command="(command: 'enable' | 'disable') => handleDropdownCommand(scope.row, command)">
                <el-button size="small">
                  更多 <el-icon class="el-icon--right"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="enable">
                      <el-icon><Check /></el-icon> 开机自启
                    </el-dropdown-item>
                    <el-dropdown-item command="disable">
                      <el-icon><Close /></el-icon> 取消自启
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </template>
          </el-table-column>
        </el-table>
      </div>
      
      <div class="pagination-container">
        <el-pagination
          v-model:current-page="serviceStore.linuxFilters.page"
          v-model:page-size="serviceStore.linuxFilters.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          :total="serviceStore.linuxTotal"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
    
    <!-- 操作确认对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="400px"
      :close-on-click-modal="false"
    >
      <div class="dialog-content">
        <p>{{ dialogContent }}</p>
      </div>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="dialogLoading"
            @click="handleConfirm"
          >
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Search,
  RefreshRight,
  VideoPlay,
  CircleClose,
  ArrowDown,
  Check,
  Close
} from '@element-plus/icons-vue'
import { useServiceStore } from '../../stores/service'
import type { LinuxServiceInfo, LinuxServiceOperationParams } from '../../types/service'

// 服务管理Store
const serviceStore = useServiceStore()

// 搜索表单
const searchForm = reactive({
  name: '',
  status: '' as 'active' | 'inactive' | 'failed' | '',
  machineId: ''
})

// 分页由Store管理

// 选中的服务
const selectedServices = ref<LinuxServiceInfo[]>([])

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('')
const dialogContent = ref('')
const dialogLoading = ref(false)
const currentOperation = ref<{ service: LinuxServiceInfo; action: string } | null>(null)

// 状态文本映射
const statusTextMap = {
  active: '运行中',
  inactive: '已停止',
  failed: '失败'
}

// 状态标签类型映射
const statusTagTypeMap = {
  active: 'success',
  inactive: 'info',
  failed: 'danger'
}

// 获取状态文本
const getStatusText = (status: string): string => {
  return statusTextMap[status as keyof typeof statusTextMap] || status
}

// 获取状态标签类型
const getStatusTagType = (status: string): string => {
  return statusTagTypeMap[status as keyof typeof statusTagTypeMap] || 'info'
}

// 格式化日期
const formatDate = (dateString: string): string => {
  if (!dateString) return ''
  const date = new Date(dateString)
  return date.toLocaleString()
}

// 搜索
const handleSearch = () => {
  serviceStore.setLinuxFilters({
    ...searchForm,
    status: searchForm.status || undefined,
    page: 1
  })
  serviceStore.fetchLinuxServiceList()
}

// 重置
const handleReset = () => {
  Object.assign(searchForm, {
    name: '',
    status: '',
    machineId: ''
  })
  serviceStore.resetLinuxFilters()
  serviceStore.fetchLinuxServiceList()
}

// 分页大小变化
const handleSizeChange = (newSize: number) => {
  serviceStore.setLinuxFilters({ pageSize: newSize })
  serviceStore.fetchLinuxServiceList()
}

// 当前页变化
const handleCurrentChange = (newPage: number) => {
  serviceStore.setLinuxFilters({ page: newPage })
  serviceStore.fetchLinuxServiceList()
}

// 选择变化
const handleSelectionChange = (selection: LinuxServiceInfo[]) => {
  selectedServices.value = selection
}

// 服务操作
const handleServiceOperation = (service: LinuxServiceInfo, action: string) => {
  const actionTextMap = {
    start: '启动',
    stop: '停止',
    restart: '重启'
  }
  
  dialogTitle.value = `${actionTextMap[action as keyof typeof actionTextMap]}服务`
  dialogContent.value = `确定要${actionTextMap[action as keyof typeof actionTextMap]}服务"${service.name}"吗？`
  currentOperation.value = { service, action }
  dialogVisible.value = true
}

// 下拉菜单操作
const handleDropdownCommand = (service: LinuxServiceInfo, command: string) => {
  const commandTextMap = {
    enable: '设置为开机自启',
    disable: '取消开机自启'
  }
  
  dialogTitle.value = commandTextMap[command as keyof typeof commandTextMap]
  dialogContent.value = `确定要${commandTextMap[command as keyof typeof commandTextMap]}服务"${service.name}"吗？`
  currentOperation.value = { service, action: command }
  dialogVisible.value = true
}

// 确认操作
const handleConfirm = async () => {
  if (!currentOperation.value) return
  
  try {
    dialogLoading.value = true
    
    const { service, action } = currentOperation.value
    const params: LinuxServiceOperationParams = {
      serviceId: service.id,
      operation: action as 'start' | 'stop' | 'restart' | 'enable' | 'disable'
    }
    
    const res = await serviceStore.handleLinuxServiceOperation(params)
    if (res) {
      ElMessage.success('操作成功')
    } else {
      ElMessage.error('操作失败')
    }
    
    dialogVisible.value = false
  } catch (error) {
    console.error('操作失败:', error)
    ElMessage.error('操作失败，请重试')
  } finally {
    dialogLoading.value = false
    currentOperation.value = null
  }
}

// 组件挂载时获取服务列表
onMounted(() => {
  serviceStore.fetchLinuxServiceList()
})
</script>

<style scoped>
.linux-service-management {
  padding: 20px;
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 {
  margin: 0 0 20px 0;
  color: #303133;
}

.filter-card {
  margin-bottom: 20px;
}

.table-container {
  margin-bottom: 20px;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
}

.description-text,
.start-cmd {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.service-name {
  font-weight: 500;
}
</style>