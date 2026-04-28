<template>
  <div class="k8s-cluster-list">
    <div class="page-header">
      <h2>Kubernetes 集群列表</h2>
      <p class="page-desc">查看和管理已部署的 Kubernetes 集群</p>
    </div>

    <div class="action-bar">
      <el-button type="primary" @click="goToDeploy">
        <el-icon><Plus /></el-icon>
        新建部署
      </el-button>
    </div>

    <div class="cluster-card">
      <div class="cluster-card-header">
        <div class="cluster-card-indicator">
          <el-icon :size="20"><Grid /></el-icon>
        </div>
        <h3 class="cluster-card-title">集群列表</h3>
      </div>

      <div class="cluster-card-body">
        <el-table
          v-loading="loading"
          :data="clusterList"
          stripe
          style="width: 100%"
          @row-click="handleRowClick"
        >
          <el-table-column prop="cluster_name" label="集群名称" min-width="180">
            <template #default="scope">
              <span class="cluster-name">{{ scope.row.cluster_name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="version" label="K8s 版本" width="120" />
          <el-table-column prop="status" label="状态" width="120">
            <template #default="scope">
              <el-tag :type="getStatusType(scope.row.status)" size="small">
                {{ getStatusText(scope.row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" width="180">
            <template #default="scope">
              {{ scope.row.created_at ? new Date(scope.row.created_at).toLocaleString('zh-CN') : '--' }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" fixed="right">
            <template #default="scope">
              <el-button type="primary" link size="small" @click.stop="handleViewCluster(scope.row)">
                查看
              </el-button>
              <el-button type="primary" link size="small" @click.stop="handleDownloadKubeconfig(scope.row)">
                下载 kubeconfig
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <el-empty
          v-if="!loading && clusterList.length === 0"
          description="暂无集群，点击「新建部署」开始部署 Kubernetes 集群"
          class="empty-state"
        >
          <el-button type="primary" @click="goToDeploy">新建部署</el-button>
        </el-empty>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, Grid } from '@element-plus/icons-vue'
import { getClusterList } from '../../../api/k8s-deploy'

const router = useRouter()

interface ClusterItem {
  id: string
  cluster_name: string  // 与后端 K8sCluster.cluster_name 对齐
  version: string
  status: string
  created_at: string
}

const loading = ref(false)
const clusterList = ref<ClusterItem[]>([])

const goToDeploy = () => {
  router.push('/service/k8s-deploy')
}

const loadClusters = async () => {
  loading.value = true
  try {
    const res = await getClusterList() as any
    // 后端返回 PageResult { list, total }
    clusterList.value = (res?.list ?? res ?? []) as ClusterItem[]
  } catch (e: any) {
    clusterList.value = []
    ElMessage.warning('获取集群列表失败: ' + (e.msg || e.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

const handleRowClick = (row: ClusterItem) => {
  handleViewCluster(row)
}

const handleViewCluster = (row: ClusterItem) => {
  ElMessage.info(`集群「${row.cluster_name}」详情页开发中`)
}

const handleDownloadKubeconfig = (row: ClusterItem) => {
  ElMessage.info(`kubeconfig 下载功能开发中 (集群: ${row.cluster_name})`)
}

const getStatusType = (status: string) => {
  const m: Record<string, 'success' | 'danger' | 'warning' | 'info'> = {
    running: 'success',
    failed: 'danger',
    deploying: 'warning',
    pending: 'info',
    success: 'success',
    cancelled: 'info',
  }
  return m[status] || 'info'
}

const getStatusText = (status: string) => {
  const m: Record<string, string> = {
    running: '运行中',
    failed: '失败',
    deploying: '部署中',
    pending: '待部署',
    success: '已部署',
    cancelled: '已取消',
  }
  return m[status] || status
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.k8s-cluster-list {
  width: 100%;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.page-header {
  text-align: center;
}

.page-header h2 {
  color: var(--el-color-primary);
  margin: 0 0 6px 0;
  font-size: 26px;
  font-weight: 600;
}

.page-desc {
  color: #6b7280;
  font-size: 14px;
  margin: 0;
}

.action-bar {
  display: flex;
  justify-content: flex-end;
}

.cluster-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.cluster-card-header {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 28px;
  border-bottom: 1px solid #f0f0f0;
  background: linear-gradient(135deg, var(--mi-surface-warm-a) 0%, var(--mi-surface-warm-b) 100%);
}

.cluster-card-indicator {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-dark-2));
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.cluster-card-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: #1f2937;
}

.cluster-card-body {
  padding: 20px 28px;
}

.cluster-name {
  font-weight: 500;
  color: var(--el-color-primary);
  cursor: pointer;
}

.cluster-name:hover {
  text-decoration: underline;
}

.empty-state {
  padding: 48px 0;
}
</style>
