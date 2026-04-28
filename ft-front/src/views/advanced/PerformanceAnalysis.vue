<template>
  <div class="performance-analysis">
    <div class="page-header">
      <h2>性能分析</h2>
    </div>
    
    <!-- 搜索和筛选区域 -->
    <div class="search-filters">
      <el-select
        v-model="filters.machineId"
        placeholder="选择机器"
        clearable
        @change="handleSearch"
        class="filter-select"
      >
        <el-option
          v-for="machine in machines"
          :key="machine.id"
          :label="machine.name"
          :value="machine.id"
        />
      </el-select>
      
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始时间"
        end-placeholder="结束时间"
        format="YYYY-MM-DD HH:mm"
        value-format="YYYY-MM-DD HH:mm"
        @change="handleDateChange"
        class="date-picker"
      />
      
      <el-select
        v-model="filters.interval"
        placeholder="选择时间间隔"
        @change="handleSearch"
        class="filter-select"
      >
        <el-option label="1分钟" value="1m" />
        <el-option label="5分钟" value="5m" />
        <el-option label="15分钟" value="15m" />
        <el-option label="1小时" value="1h" />
      </el-select>
      
      <el-select
        v-model="selectedMetrics"
        placeholder="选择指标"
        multiple
        clearable
        @change="handleMetricsChange"
        class="filter-select"
      >
        <el-option label="CPU使用率" value="cpu" />
        <el-option label="内存使用率" value="memory" />
        <el-option label="磁盘使用率" value="disk" />
        <el-option label="网络流量" value="network" />
      </el-select>
      
      <el-button type="primary" @click="handleSearch">
        <el-icon><Search /></el-icon>
        查询数据
      </el-button>
      
      <el-button @click="handleReset">
        <el-icon><RefreshRight /></el-icon>
        重置
      </el-button>
      
      <el-button type="success" @click="handleGenerateReport">
        <el-icon><Document /></el-icon>
        生成报告
      </el-button>
      
      <el-button type="info" @click="handleExportData">
        <el-icon><Download /></el-icon>
        导出数据
      </el-button>
    </div>
    
    <!-- 统计卡片区域 -->
    <div class="stats-cards">
      <el-card class="stat-card" shadow="hover">
        <div class="card-content">
          <div class="card-title">CPU使用率</div>
          <div class="card-value">{{ stats.cpu.average.toFixed(2) }}%</div>
          <div class="card-desc">平均: {{ stats.cpu.average.toFixed(2) }}% | 最高: {{ stats.cpu.max.toFixed(2) }}% | 最低: {{ stats.cpu.min.toFixed(2) }}%</div>
        </div>
      </el-card>
      
      <el-card class="stat-card" shadow="hover">
        <div class="card-content">
          <div class="card-title">内存使用率</div>
          <div class="card-value">{{ stats.memory.average.toFixed(2) }}%</div>
          <div class="card-desc">平均: {{ stats.memory.average.toFixed(2) }}% | 最高: {{ stats.memory.max.toFixed(2) }}% | 最低: {{ stats.memory.min.toFixed(2) }}%</div>
        </div>
      </el-card>
      
      <el-card class="stat-card" shadow="hover">
        <div class="card-content">
          <div class="card-title">磁盘使用率</div>
          <div class="card-value">{{ stats.disk.average.toFixed(2) }}%</div>
          <div class="card-desc">平均: {{ stats.disk.average.toFixed(2) }}% | 最高: {{ stats.disk.max.toFixed(2) }}% | 最低: {{ stats.disk.min.toFixed(2) }}%</div>
        </div>
      </el-card>
      
      <el-card class="stat-card" shadow="hover">
        <div class="card-content">
          <div class="card-title">网络流量</div>
          <div class="card-value">{{ formatNetworkSpeed(stats.network.in.average + stats.network.out.average) }}</div>
          <div class="card-desc">入站: {{ formatNetworkSpeed(stats.network.in.average) }} | 出站: {{ formatNetworkSpeed(stats.network.out.average) }}</div>
        </div>
      </el-card>
    </div>
    
    <!-- 图表展示区域 -->
    <div class="charts-container">
      <el-card class="chart-card">
        <template #header>
          <div class="card-header">
            <span>性能趋势图</span>
          </div>
        </template>
        <div class="chart-content">
          <!-- 这里使用Element Plus的图表组件，实际项目中可能需要引入ECharts或其他图表库 -->
          <div class="chart-placeholder">
            <el-icon class="chart-icon"><PieChart /></el-icon>
            <div class="chart-text">性能数据趋势图</div>
            <div class="chart-subtext">选择时间范围和指标查看详细数据</div>
          </div>
        </div>
      </el-card>
    </div>
    
    <!-- 数据表格区域 -->
    <div class="data-table-container">
      <el-card class="data-table-card">
        <template #header>
          <div class="card-header">
            <span>性能数据详情</span>
          </div>
        </template>
        <el-table
          v-loading="loading"
          :data="currentPageData"
          stripe
          border
          style="width: 100%"
        >
          <el-table-column prop="timestamp" label="时间" min-width="180" align="center" />
          <el-table-column prop="cpuUsage" label="CPU使用率 (%)" min-width="120" align="center" />
          <el-table-column prop="memoryUsage" label="内存使用率 (%)" min-width="120" align="center" />
          <el-table-column prop="diskUsage" label="磁盘使用率 (%)" min-width="120" align="center" />
          <el-table-column prop="networkIn" label="入站流量 (KB/s)" min-width="120" align="center" />
          <el-table-column prop="networkOut" label="出站流量 (KB/s)" min-width="120" align="center" />
        </el-table>
      </el-card>
    </div>
    
    <!-- 分页 -->
    <div class="pagination">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        :total="performanceData.length"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
    
    <!-- 报告生成对话框 -->
    <el-dialog
      v-model="reportDialogVisible"
      title="生成性能报告"
      width="500px"
    >
      <el-form
        ref="reportFormRef"
        :model="reportForm"
        :rules="reportRules"
        label-position="top"
      >
        <el-form-item label="报告名称" prop="name">
          <el-input
            v-model="reportForm.name"
            placeholder="请输入报告名称"
            clearable
          />
        </el-form-item>
        
        <el-form-item label="报告描述" prop="description">
          <el-input
            v-model="reportForm.description"
            placeholder="请输入报告描述"
            type="textarea"
            :rows="3"
            clearable
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="reportDialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :loading="reportDialogLoading"
            @click="handleReportDialogSubmit"
          >
            生成报告
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Search, RefreshRight, Document, Download, PieChart } from '@element-plus/icons-vue'
import { getPerformanceData } from '../../api/advanced'
import type { PerformanceData, PerformanceParams, PerformanceDataResponse } from '../../types'

// 响应式状态
const loading = ref(false)
const reportDialogVisible = ref(false)
const reportDialogLoading = ref(false)
const reportFormRef = ref()
const currentPage = ref(1)
const pageSize = ref(20)

// 筛选条件
const filters = reactive<PerformanceParams>({
  machineId: undefined,
  startTime: '',
  endTime: '',
  interval: '5m',
  metrics: ['cpu', 'memory', 'disk', 'network']
})

const dateRange = ref<[string, string] | null>(null)
const selectedMetrics = ref<Array<'cpu' | 'memory' | 'disk' | 'network'>>(['cpu', 'memory', 'disk', 'network'])

// 机器列表
const machines = ref<Array<{ id: number; name: string }>>([
  { id: 1, name: '服务器1' },
  { id: 2, name: '服务器2' },
  { id: 3, name: '服务器3' },
  { id: 4, name: '服务器4' }
])

// 性能数据
const performanceData = ref<PerformanceData[]>([])

// 统计数据
const stats = reactive({
  cpu: { average: 0, max: 0, min: 0 },
  memory: { average: 0, max: 0, min: 0 },
  disk: { average: 0, max: 0, min: 0 },
  network: { in: { average: 0 }, out: { average: 0 } }
})

// 报告表单
const reportForm = reactive({
  name: '',
  description: ''
})

// 报告表单验证规则
const reportRules = reactive({
  name: [
    { required: true, message: '请输入报告名称', trigger: 'blur' },
    { min: 2, max: 50, message: '报告名称长度在 2 到 50 个字符', trigger: 'blur' }
  ]
})

// 计算当前页的数据
const currentPageData = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return performanceData.value.slice(start, end)
})

// 加载性能数据
onMounted(() => {
  fetchPerformanceData()
})

// 获取性能数据
const fetchPerformanceData = async () => {
  loading.value = true
  try {
    const response: PerformanceDataResponse = await getPerformanceData(filters)
    performanceData.value = response.data
    machines.value = response.machines
    updateStats()
  } catch (error) {
    console.error('获取性能数据失败:', error)
    ElMessage.error('获取性能数据失败')
  } finally {
    loading.value = false
  }
}

// 更新统计数据
const updateStats = () => {
  if (performanceData.value.length === 0) {
    Object.assign(stats, {
      cpu: { average: 0, max: 0, min: 0 },
      memory: { average: 0, max: 0, min: 0 },
      disk: { average: 0, max: 0, min: 0 },
      network: { in: { average: 0 }, out: { average: 0 } }
    })
    return
  }

  // 计算CPU统计
  const cpuValues = performanceData.value.map(item => item.cpuUsage)
  stats.cpu.average = cpuValues.reduce((sum, val) => sum + val, 0) / cpuValues.length
  stats.cpu.max = Math.max(...cpuValues)
  stats.cpu.min = Math.min(...cpuValues)

  // 计算内存统计
  const memoryValues = performanceData.value.map(item => item.memoryUsage)
  stats.memory.average = memoryValues.reduce((sum, val) => sum + val, 0) / memoryValues.length
  stats.memory.max = Math.max(...memoryValues)
  stats.memory.min = Math.min(...memoryValues)

  // 计算磁盘统计
  const diskValues = performanceData.value.map(item => item.diskUsage)
  stats.disk.average = diskValues.reduce((sum, val) => sum + val, 0) / diskValues.length
  stats.disk.max = Math.max(...diskValues)
  stats.disk.min = Math.min(...diskValues)

  // 计算网络统计
  const networkInValues = performanceData.value.map(item => item.networkIn)
  const networkOutValues = performanceData.value.map(item => item.networkOut)
  stats.network.in.average = networkInValues.reduce((sum, val) => sum + val, 0) / networkInValues.length
  stats.network.out.average = networkOutValues.reduce((sum, val) => sum + val, 0) / networkOutValues.length
}

// 处理搜索
const handleSearch = () => {
  fetchPerformanceData()
}

// 处理日期范围变化
const handleDateChange = (val: [string, string] | null) => {
  if (val) {
    filters.startTime = val[0]
    filters.endTime = val[1]
  } else {
    filters.startTime = ''
    filters.endTime = ''
  }
  fetchPerformanceData()
}

// 处理指标变化
const handleMetricsChange = () => {
  filters.metrics = selectedMetrics.value
  fetchPerformanceData()
}

// 处理重置
const handleReset = () => {
  filters.machineId = undefined
  filters.startTime = ''
  filters.endTime = ''
  filters.interval = '5m'
  filters.metrics = ['cpu', 'memory', 'disk', 'network']
  dateRange.value = null
  selectedMetrics.value = ['cpu', 'memory', 'disk', 'network']
  fetchPerformanceData()
}

// 处理分页大小变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  currentPage.value = 1
}

// 处理当前页变化
const handleCurrentChange = (current: number) => {
  currentPage.value = current
}

// 处理生成报告
const handleGenerateReport = () => {
  reportDialogVisible.value = true
}

// 处理导出数据
const handleExportData = async () => {
  loading.value = true
  try {
    // 这里可以实现数据导出功能
    ElMessage.success('数据导出成功')
  } catch (error) {
    console.error('导出数据失败:', error)
    ElMessage.error('导出数据失败')
  } finally {
    loading.value = false
  }
}

// 处理报告生成对话框提交
const handleReportDialogSubmit = async () => {
  if (!reportFormRef.value) return
  
  try {
    await reportFormRef.value.validate()
    reportDialogLoading.value = true
    
    // 这里应该调用API生成报告
    // await generatePerformanceReport(filters)
    
    ElMessage.success('报告生成成功')
    reportDialogVisible.value = false
    resetReportForm()
  } catch (error) {
    console.error('生成报告失败:', error)
    ElMessage.error('生成报告失败')
  } finally {
    reportDialogLoading.value = false
  }
}

// 重置报告表单
const resetReportForm = () => {
  reportForm.name = ''
  reportForm.description = ''
  
  if (reportFormRef.value) {
    reportFormRef.value.resetFields()
  }
}

// 格式化网络速度
const formatNetworkSpeed = (speed: number): string => {
  if (speed < 1024) {
    return `${speed.toFixed(2)} KB/s`
  } else {
    return `${(speed / 1024).toFixed(2)} MB/s`
  }
}
</script>

<style scoped>
.performance-analysis {
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

.filter-select {
  width: 150px;
}

.date-picker {
  width: 320px;
}

.stats-cards {
  display: flex;
  gap: 20px;
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.stat-card {
  flex: 1;
  min-width: 250px;
}

.card-content {
  text-align: center;
}

.card-title {
  font-size: 14px;
  color: #666;
  margin-bottom: 8px;
}

.card-value {
  font-size: 32px;
  font-weight: 600;
  color: var(--el-color-primary);
  margin-bottom: 8px;
}

.card-desc {
  font-size: 12px;
  color: #999;
}

.charts-container {
  margin-bottom: 20px;
}

.chart-card {
  height: 400px;
}

.chart-content {
  height: 350px;
  display: flex;
  justify-content: center;
  align-items: center;
}

.chart-placeholder {
  text-align: center;
  color: #999;
}

.chart-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.chart-text {
  font-size: 18px;
  margin-bottom: 8px;
}

.chart-subtext {
  font-size: 14px;
}

.data-table-container {
  margin-bottom: 20px;
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