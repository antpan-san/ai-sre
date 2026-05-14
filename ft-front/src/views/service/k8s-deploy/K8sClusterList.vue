<template>
  <div class="k8s-cluster-list">
    <div class="page-header">
      <h2>集群</h2>
      <el-button type="primary" size="small" text @click="goToDeploy">新建</el-button>
    </div>

    <div class="cluster-sheet">
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
          description="暂无集群。使用右上角「新建」开始。"
          class="empty-state"
        >
          <el-button type="primary" @click="goToDeploy">新建</el-button>
        </el-empty>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
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
  padding: var(--page-padding-y) var(--page-padding-x);
  display: flex;
  flex-direction: column;
  gap: 14px;
  box-sizing: border-box;
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.page-header h2 {
  margin: 0;
  font-size: var(--page-header-title-max);
  font-weight: 600;
  color: var(--apple-ink);
}

.cluster-sheet {
  min-width: 0;
}

.cluster-card-body {
  padding: 0;
  min-height: 0;
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
