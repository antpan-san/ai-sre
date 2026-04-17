<template>
  <div class="dashboard">
    <div class="page-header">
      <h2>仪表盘</h2>
      <el-button 
        type="primary" 
        :icon="RefreshRight" 
        @click="handleRefresh"
        :loading="dashboardStore.loading"
      >
        刷新
      </el-button>
    </div>
    
    <!-- 资源使用情况卡片 -->
    <div class="resource-cards">
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>CPU使用率</span>
            <el-tag :type="getUsageType(dashboardStore.dashboardData?.resourceUsage?.cpu || 0)" size="small">
              {{ dashboardStore.dashboardData?.resourceUsage?.cpu || 0 }}%
            </el-tag>
          </div>
        </template>
        <div class="card-content">
          <el-progress 
            :percentage="dashboardStore.dashboardData?.resourceUsage?.cpu || 0" 
            :color="getUsageColor(dashboardStore.dashboardData?.resourceUsage?.cpu || 0)"
            :show-text="false"
            class="progress-bar"
          />
        </div>
      </el-card>
      
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>内存使用率</span>
            <el-tag :type="getUsageType(dashboardStore.dashboardData?.resourceUsage?.memory || 0)" size="small">
              {{ dashboardStore.dashboardData?.resourceUsage?.memory || 0 }}%
            </el-tag>
          </div>
        </template>
        <div class="card-content">
          <el-progress 
            :percentage="dashboardStore.dashboardData?.resourceUsage?.memory || 0" 
            :color="getUsageColor(dashboardStore.dashboardData?.resourceUsage?.memory || 0)"
            :show-text="false"
            class="progress-bar"
          />
        </div>
      </el-card>
      
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>磁盘使用率</span>
            <el-tag :type="getUsageType(dashboardStore.dashboardData?.resourceUsage?.disk || 0)" size="small">
              {{ dashboardStore.dashboardData?.resourceUsage?.disk || 0 }}%
            </el-tag>
          </div>
        </template>
        <div class="card-content">
          <el-progress 
            :percentage="dashboardStore.dashboardData?.resourceUsage?.disk || 0" 
            :color="getUsageColor(dashboardStore.dashboardData?.resourceUsage?.disk || 0)"
            :show-text="false"
            class="progress-bar"
          />
        </div>
      </el-card>
      
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="resource-card">
        <template #header>
          <div class="card-header">
            <span>网络流量</span>
            <el-tag type="info" size="small">
              入: {{ formatNetwork(dashboardStore.dashboardData?.resourceUsage?.network?.in || 0) }} / 
              出: {{ formatNetwork(dashboardStore.dashboardData?.resourceUsage?.network?.out || 0) }}
            </el-tag>
          </div>
        </template>
        <div class="card-content network-content">
          <div class="network-item">
            <span class="label">入流量</span>
            <span class="value">{{ formatNetwork(dashboardStore.dashboardData?.resourceUsage?.network?.in || 0) }}</span>
          </div>
          <div class="network-item">
            <span class="label">出流量</span>
            <span class="value">{{ formatNetwork(dashboardStore.dashboardData?.resourceUsage?.network?.out || 0) }}</span>
          </div>
        </div>
      </el-card>
    </div>
    
    <!-- 中间区域 -->
    <div class="middle-section">
      <!-- Kubernetes资源概览 -->
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="overview-card">
        <template #header>
          <div class="card-header">
            <span>Kubernetes资源概览</span>
          </div>
        </template>
        <div class="overview-content">
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Grid /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dashboardStore.dashboardData?.kubernetesOverview?.nodes || 0 }}</div>
              <div class="overview-label">节点</div>
            </div>
          </div>
          
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><List /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dashboardStore.dashboardData?.kubernetesOverview?.pods || 0 }}</div>
              <div class="overview-label">Pod总数</div>
            </div>
          </div>
          
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Check /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dashboardStore.dashboardData?.kubernetesOverview?.runningPods || 0 }}</div>
              <div class="overview-label">运行Pod</div>
            </div>
          </div>
          
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Link /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dashboardStore.dashboardData?.kubernetesOverview?.services || 0 }}</div>
              <div class="overview-label">服务</div>
            </div>
          </div>
          
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><Upload /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dashboardStore.dashboardData?.kubernetesOverview?.deployments || 0 }}</div>
              <div class="overview-label">部署</div>
            </div>
          </div>
          
          <div class="overview-item">
            <div class="overview-icon">
              <el-icon :size="32"><CopyDocument /></el-icon>
            </div>
            <div class="overview-info">
              <div class="overview-value">{{ dashboardStore.dashboardData?.kubernetesOverview?.replicasets || 0 }}</div>
              <div class="overview-label">ReplicaSet</div>
            </div>
          </div>
        </div>
      </el-card>
      
      <!-- 服务状态统计 -->
      <el-card v-loading="dashboardStore.loading" shadow="hover" class="stats-card">
        <template #header>
          <div class="card-header">
            <span>服务状态统计</span>
            <el-tag type="info" size="small">
              总计: {{ dashboardStore.dashboardData?.serviceStatusStats?.total || 0 }}
            </el-tag>
          </div>
        </template>
        <div class="stats-content">
          <div class="stats-item">
            <el-progress 
              type="circle" 
              :percentage="getStatusPercentage('running')" 
              :color="getStatusColor('running')"
              :format="(_percentage: number) => ''"
              :width="60"
            />
            <div class="stats-info">
              <div class="stats-value">{{ dashboardStore.dashboardData?.serviceStatusStats?.running || 0 }}</div>
              <div class="stats-label">运行中</div>
            </div>
          </div>
          
          <div class="stats-item">
            <el-progress 
              type="circle" 
              :percentage="getStatusPercentage('stopped')" 
              :color="getStatusColor('stopped')"
              :format="(_percentage: number) => ''"
              :width="60"
            />
            <div class="stats-info">
              <div class="stats-value">{{ dashboardStore.dashboardData?.serviceStatusStats?.stopped || 0 }}</div>
              <div class="stats-label">已停止</div>
            </div>
          </div>
          
          <div class="stats-item">
            <el-progress 
              type="circle" 
              :percentage="getStatusPercentage('error')" 
              :color="getStatusColor('error')"
              :format="() => ''"
              :width="60"
            />
            <div class="stats-info">
              <div class="stats-value">{{ dashboardStore.dashboardData?.serviceStatusStats?.error || 0 }}</div>
              <div class="stats-label">错误</div>
            </div>
          </div>
        </div>
      </el-card>
    </div>
    
    <!-- 最近部署的服务 -->
    <el-card v-loading="dashboardStore.loading" shadow="hover" class="recent-deployments-card">
      <template #header>
        <div class="card-header">
          <span>最近部署的服务</span>
          <el-link type="primary" href="#" :underline="false" @click.prevent="navigateToServiceList">
            查看全部
          </el-link>
        </div>
      </template>
      <div class="recent-deployments-content">
        <el-table :data="dashboardStore.dashboardData?.recentDeployments || []" stripe border size="small">
          <el-table-column prop="name" label="服务名称" min-width="150" />
          <el-table-column prop="image" label="镜像" min-width="180" />
          <el-table-column prop="replicas" label="副本数" width="80" align="center" />
          <el-table-column prop="status" label="状态" width="100" align="center">
            <template #default="scope">
              <el-tag :type="getStatusType(scope.row.status)" size="small">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="updateTime" label="更新时间" min-width="180" />
        </el-table>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshRight, Grid, List, Check, Link, Upload, CopyDocument } from '@element-plus/icons-vue'
import { useDashboardStore } from '../stores/dashboard'

const dashboardStore = useDashboardStore()
const router = useRouter()

// 页面加载时获取仪表盘数据
onMounted(() => {
  fetchDashboardData()
})

// 获取仪表盘数据
const fetchDashboardData = async () => {
  const data = await dashboardStore.fetchDashboardData()
  if (!data) {
    ElMessage.error('获取仪表盘数据失败')
  }
}

// 刷新数据
const handleRefresh = () => {
  fetchDashboardData()
}

// 获取使用率类型
const getUsageType = (percentage: number) => {
  if (percentage >= 80) return 'danger'
  if (percentage >= 60) return 'warning'
  return 'success'
}

// 获取使用率颜色
const getUsageColor = (percentage: number) => {
  if (percentage >= 80) return '#f56c6c'
  if (percentage >= 60) return '#e6a23c'
  return '#67c23a'
}

// 格式化网络流量
const formatNetwork = (value: number) => {
  if (value >= 1024) {
    return `${(value / 1024).toFixed(2)} GB`
  }
  return `${value.toFixed(2)} MB`
}

// 获取状态百分比
const getStatusPercentage = (status: 'running' | 'stopped' | 'error') => {
  const total = dashboardStore.dashboardData?.serviceStatusStats?.total || 0
  if (total === 0) return 0
  
  let count = 0
  switch (status) {
    case 'running':
      count = dashboardStore.dashboardData?.serviceStatusStats?.running || 0
      break
    case 'stopped':
      count = dashboardStore.dashboardData?.serviceStatusStats?.stopped || 0
      break
    case 'error':
      count = dashboardStore.dashboardData?.serviceStatusStats?.error || 0
      break
  }
  
  return Math.round((count / total) * 100)
}

// 获取状态颜色
const getStatusColor = (status: 'running' | 'stopped' | 'error') => {
  switch (status) {
    case 'running':
      return '#67c23a'
    case 'stopped':
      return '#909399'
    case 'error':
      return '#f56c6c'
    default:
      return '#409eff'
  }
}

// 获取状态类型
const getStatusType = (status: string) => {
  switch (status) {
    case 'running':
      return 'success'
    case 'stopped':
      return 'info'
    case 'error':
      return 'danger'
    default:
      return 'warning'
  }
}

// 获取状态文本
const getStatusText = (status: string) => {
  switch (status) {
    case 'running':
      return '运行中'
    case 'stopped':
      return '已停止'
    case 'error':
      return '错误'
    default:
      return status
  }
}

// 导航到服务列表
const navigateToServiceList = () => {
  router.push('/service/deploy')
}
</script>

<style scoped>
.dashboard {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
  color: #303133;
}

/* 资源卡片样式 */
.resource-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 20px;
}

.resource-card {
  height: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: bold;
}

.card-content {
  padding: 20px 0;
}

.progress-bar {
  height: 10px;
}

.network-content {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.network-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.network-item .label {
  color: #606266;
}

.network-item .value {
  font-weight: bold;
  color: #303133;
}

/* 中间区域样式 */
.middle-section {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 20px;
  margin-bottom: 20px;
}

/* 概览卡片样式 */
.overview-content {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 20px;
}

.overview-item {
  display: flex;
  align-items: center;
  gap: 10px;
}

.overview-icon {
  color: #409eff;
}

.overview-info {
  display: flex;
  flex-direction: column;
}

.overview-value {
  font-size: 20px;
  font-weight: bold;
  color: #303133;
}

.overview-label {
  color: #606266;
  font-size: 14px;
}

/* 统计卡片样式 */
.stats-content {
  display: flex;
  justify-content: space-around;
  align-items: center;
}

.stats-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.stats-info {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.stats-value {
  font-size: 18px;
  font-weight: bold;
  color: #303133;
}

.stats-label {
  color: #606266;
  font-size: 14px;
}

/* 最近部署样式 */
.recent-deployments-card {
  margin-bottom: 20px;
}

@media screen and (max-width: 1200px) {
  .middle-section {
    grid-template-columns: 1fr;
  }
}

@media screen and (max-width: 768px) {
  .resource-cards {
    grid-template-columns: 1fr;
  }
  
  .overview-content {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .stats-content {
    flex-direction: column;
    gap: 20px;
  }
}
</style>
