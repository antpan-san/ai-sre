<template>
  <div class="k8s-cluster-panel">
    <div v-if="showToolbar" class="k8s-cluster-panel__toolbar">
      <p class="k8s-cluster-panel__hint page-desc--muted">
        {{ hintText }}
      </p>
      <el-button type="primary" size="small" @click="goToDeploy">新建集群</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="displayList"
      stripe
      border
      size="small"
      class="k8s-cluster-panel__table"
      @row-click="handleRowClick"
    >
      <el-table-column prop="cluster_name" label="集群名称" min-width="160" show-overflow-tooltip>
        <template #default="scope">
          <span class="cluster-name">{{ scope.row.cluster_name }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="version" label="K8s 版本" width="110" show-overflow-tooltip />
      <el-table-column prop="status" label="状态" width="100" align="center">
        <template #default="scope">
          <el-tag :type="getStatusType(scope.row.status)" size="small">
            {{ getStatusText(scope.row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="创建时间" width="168">
        <template #default="scope">
          {{ scope.row.created_at ? new Date(scope.row.created_at).toLocaleString('zh-CN') : '—' }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="168" fixed="right" align="right">
        <template #default="scope">
          <el-button type="primary" link size="small" @click.stop="handleViewCluster(scope.row)">查看</el-button>
          <el-button type="primary" link size="small" @click.stop="handleDownloadKubeconfig(scope.row)">
            下载 kubeconfig
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-empty
      v-if="!loading && clusterList.length === 0"
      description="暂无集群。点击右上角「新建集群」开始。"
      class="k8s-cluster-panel__empty"
    >
      <el-button type="primary" size="small" @click="goToDeploy">新建集群</el-button>
    </el-empty>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getClusterList } from '../../api/k8s-deploy'

const props = withDefaults(
  defineProps<{
    deployPath?: string
    progressPath?: string
    maxRows?: number
    hint?: string
    showToolbar?: boolean
  }>(),
  {
    deployPath: '/admin/service/k8s-deploy',
    progressPath: '/admin/service/k8s-deploy/progress',
    maxRows: 0,
    hint: '',
    showToolbar: true,
  }
)

const router = useRouter()

interface ClusterItem {
  id: string
  cluster_name: string
  version: string
  status: string
  created_at: string
}

const loading = ref(false)
const clusterList = ref<ClusterItem[]>([])

const hintText = computed(
  () =>
    props.hint ||
    '租户内已登记的 Kubernetes 集群；新建部署请从「部署配置 → Kubernetes」进入。'
)

const displayList = computed(() => {
  if (!props.maxRows || props.maxRows <= 0) return clusterList.value
  return clusterList.value.slice(0, props.maxRows)
})

const goToDeploy = () => {
  router.push(props.deployPath)
}

const loadClusters = async () => {
  loading.value = true
  try {
    const res = (await getClusterList()) as { list?: ClusterItem[] } | ClusterItem[]
    clusterList.value = Array.isArray(res) ? (res as ClusterItem[]) : (res?.list ?? [])
  } catch (e: unknown) {
    clusterList.value = []
    const msg = e && typeof e === 'object' && 'msg' in e ? String((e as { msg?: string }).msg) : ''
    ElMessage.warning('获取集群列表失败: ' + (msg || (e instanceof Error ? e.message : '未知错误')))
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
  ElMessage.info(`kubeconfig 下载功能开发中（集群: ${row.cluster_name}）`)
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
  const map: Record<string, string> = {
    running: '运行中',
    failed: '失败',
    deploying: '部署中',
    pending: '待部署',
    success: '已部署',
    cancelled: '已取消',
  }
  return map[status] || status
}

defineExpose({ loadClusters })

onMounted(() => {
  void loadClusters()
})
</script>

<style scoped>
.k8s-cluster-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
}

.k8s-cluster-panel__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.k8s-cluster-panel__hint {
  margin: 0;
  flex: 1;
  min-width: 200px;
  line-height: 1.45;
}

.k8s-cluster-panel__table {
  width: 100%;
}

.k8s-cluster-panel__empty {
  padding: 24px 0;
}

.cluster-name {
  font-weight: 500;
  color: var(--el-color-primary);
  cursor: pointer;
}

.cluster-name:hover {
  text-decoration: underline;
}
</style>
