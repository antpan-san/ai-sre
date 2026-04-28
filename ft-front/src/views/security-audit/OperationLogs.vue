<template>
  <div class="operation-logs">
    <div class="page-header">
      <h2>操作日志</h2>
    </div>
    
    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-input
        v-model="filters.username"
        placeholder="搜索用户名"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
      <el-input
        v-model="filters.operation"
        placeholder="搜索操作类型"
        :prefix-icon="Search"
        clearable
        @clear="handleSearch"
        @keyup.enter="handleSearch"
        class="search-input"
      />
      
      <el-input
        v-model="filters.resource"
        placeholder="搜索资源类型"
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
        <el-option label="成功" value="success" />
        <el-option label="失败" value="fail" />
      </el-select>
      
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        format="YYYY-MM-DD HH:mm"
        value-format="YYYY-MM-DD HH:mm"
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
      
      <el-button type="success" @click="handleExport">
        <el-icon><Download /></el-icon>
        导出日志
      </el-button>
    </div>
    
    <!-- 操作日志表格 -->
    <div class="logs-table">
      <el-table
        v-loading="loading"
        :data="logsList"
        stripe
        border
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" min-width="80" align="center" />
        
        <el-table-column prop="username" label="用户名" min-width="120" align="center" />
        
        <el-table-column prop="operation" label="操作" min-width="150" align="center">
          <template #default="scope">
            <el-tag size="small">{{ scope.row.operation }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="resource" label="资源" min-width="120" align="center">
          <template #default="scope">
            <el-tag size="small" type="info">{{ scope.row.resource }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="resourceId" label="资源ID" min-width="100" align="center" />
        
        <el-table-column prop="ip" label="IP地址" min-width="130" align="center" />
        
        <el-table-column prop="userAgent" label="用户代理" min-width="200" align="left">
          <template #default="scope">
            <el-tooltip effect="dark" :content="scope.row.userAgent" placement="top">
              <span class="user-agent" style="display: inline-block; width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                {{ scope.row.userAgent }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <el-table-column prop="status" label="状态" min-width="80" align="center">
          <template #default="scope">
            <el-tag
              :type="scope.row.status === 'success' ? 'success' : 'danger'"
              size="small"
            >
              {{ scope.row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="errorMessage" label="错误信息" min-width="150" align="left">
          <template #default="scope">
            <el-tooltip v-if="scope.row.errorMessage" effect="dark" :content="scope.row.errorMessage" placement="top">
              <span class="error-message" style="color: #f56c6c; display: inline-block; width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                {{ scope.row.errorMessage }}
              </span>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <el-table-column prop="createTime" label="创建时间" min-width="180" align="center" />
      </el-table>
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, RefreshRight, Download } from '@element-plus/icons-vue'
import { getOperationLogs, exportOperationLogs } from '../../api/security-audit'
import type { OperationLog, OperationLogParams, OperationLogListResponse } from '../../types'

// 响应式状态
const loading = ref(false)
const filters = reactive<OperationLogParams>({
  page: 1,
  pageSize: 20,
  username: '',
  operation: '',
  resource: '',
  status: '',
  startDate: '',
  endDate: ''
})
const dateRange = ref<[string, string] | null>(null)
const logsList = ref<OperationLog[]>([])
const total = ref(0)

// 加载操作日志
onMounted(() => {
  fetchOperationLogs()
})

// 获取操作日志列表
const fetchOperationLogs = async () => {
  loading.value = true
  try {
    const response: OperationLogListResponse = await getOperationLogs(filters)
    logsList.value = response.list
    total.value = response.total
  } catch (error) {
    console.error('获取操作日志失败:', error)
    ElMessage.error('获取操作日志失败')
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  filters.page = 1
  fetchOperationLogs()
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
  fetchOperationLogs()
}

// 处理重置
const handleReset = () => {
  Object.assign(filters, {
    page: 1,
    pageSize: 20,
    username: '',
    operation: '',
    resource: '',
    status: '',
    startDate: '',
    endDate: ''
  })
  dateRange.value = null
  fetchOperationLogs()
}

// 处理分页大小变化
const handleSizeChange = (size: number) => {
  filters.pageSize = size
  filters.page = 1
  fetchOperationLogs()
}

// 处理当前页变化
const handleCurrentChange = (current: number) => {
  filters.page = current
  fetchOperationLogs()
}

// 处理导出日志
const handleExport = async () => {
  loading.value = true
  try {
    const blob = await exportOperationLogs(filters)
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `operation-logs-${new Date().getTime()}.csv`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    ElMessage.success('日志导出成功')
  } catch (error) {
    console.error('导出日志失败:', error)
    ElMessage.error('导出日志失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.operation-logs {
  padding: 0 20px 20px 20px;
}

.page-header h2 {
  margin: 0 0 20px 0;
  color: var(--el-color-primary);
  font-size: 30px;
  font-weight: 600;
}

.search-filters {
  display: flex;
  gap: 15px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.search-input {
  width: 200px;
}

.filter-select {
  width: 120px;
}

.date-picker {
  width: 320px;
}

.logs-table {
  margin-bottom: 20px;
  width: 100%;
}

.pagination {
  text-align: center;
  margin-bottom: 20px;
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
}
</style>