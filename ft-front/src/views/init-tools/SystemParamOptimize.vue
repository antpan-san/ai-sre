<template>
  <div class="system-param-optimize">
    <div class="page-header">
      <div class="page-header-bar">
        <el-button link type="primary" :icon="ArrowLeft" @click="backToHome">返回工具总览</el-button>
      </div>
      <h2>系统参数优化</h2>
      <p class="page-desc">
        调优 sysctl / ulimit / swap 等内核参数，提升 K8s 节点稳定性与性能。请先选择目标节点与系统类型，再勾选并应用。
      </p>
    </div>

    <div class="content-container">
      <el-card class="content-card">
        <template #header>
          <div class="card-header">
            <h3>系统参数配置</h3>
            <el-button
              type="primary"
              size="small"
              @click="refreshSystemParams"
              :loading="loadingSystemParams"
            >
              <el-icon><RefreshRight /></el-icon>
              刷新
            </el-button>
          </div>
        </template>

        <NodeSystemSelector v-model="target" class="target-block" />

        <div class="system-params-container">
          <el-table
            v-loading="loadingSystemParams"
            :data="systemParams"
            style="width: 100%"
            stripe
          >
            <el-table-column
              prop="key"
              label="参数名称"
              min-width="200"
            >
              <template #default="scope">
                <div class="param-name">
                  <span class="key">{{ scope.row.key }}</span>
                  <el-tag v-if="scope.row.required" type="danger" size="small">必填</el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column
              prop="value"
              label="参数值"
              min-width="250"
            >
              <template #default="scope">
                <el-input
                  v-model="scope.row.value"
                  placeholder="请输入参数值"
                  clearable
                />
              </template>
            </el-table-column>
            <el-table-column
              prop="description"
              label="描述"
              min-width="300"
            >
              <template #default="scope">
                <el-tooltip
                  :content="scope.row.description"
                  placement="top"
                >
                  <div class="param-description">{{ scope.row.description }}</div>
                </el-tooltip>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="card-actions">
          <el-button
            type="success"
            @click="applySystemParams"
            :loading="applyingSystemParams"
            :disabled="!targetReady"
          >
            <el-icon><Check /></el-icon>
            应用到所选节点
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { RefreshRight, Check, ArrowLeft } from '@element-plus/icons-vue'
import NodeSystemSelector, { type NodeSystemValue } from '../../components/init-tools/NodeSystemSelector.vue'

const route = useRoute()
const router = useRouter()

// 系统参数优化
const loadingSystemParams = ref(false)
const applyingSystemParams = ref(false)

const target = ref<NodeSystemValue>({ nodes: [], osType: '' })
const targetReady = computed(() => target.value.nodes.length > 0 && !!target.value.osType)

onMounted(() => {
  // 兼容从总览页跳转过来时透传的 nodes / osType
  const nodesQ = (route.query.nodes as string) || ''
  const osQ = (route.query.osType as string) || ''
  if (nodesQ) target.value.nodes = nodesQ.split(',').filter(Boolean)
  if (osQ) target.value.osType = osQ as NodeSystemValue['osType']
})

const backToHome = () => {
  const q = { ...route.query }
  delete q.nodes
  delete q.osType
  router.push({ path: '/init-tools', query: q })
}

interface SystemParam {
  key: string
  value: string
  description: string
  required: boolean
}

const systemParams = ref<SystemParam[]>([
  { key: 'vm.swappiness', value: '10', description: '内存交换策略，降低值减少交换', required: true },
  { key: 'net.core.somaxconn', value: '65535', description: '最大连接数', required: true },
  { key: 'net.ipv4.tcp_max_tw_buckets', value: '6000', description: 'TIME_WAIT状态的最大连接数', required: true },
  { key: 'net.ipv4.tcp_slow_start_after_idle', value: '0', description: '关闭TCP慢启动', required: true },
  { key: 'net.ipv4.tcp_syncookies', value: '1', description: '启用SYN cookies', required: true },
  { key: 'fs.file-max', value: '655350', description: '系统最大文件句柄数', required: true },
  { key: 'kernel.sysrq', value: '1', description: '启用SysRq功能', required: false },
  { key: 'kernel.panic', value: '60', description: '系统崩溃后自动重启时间', required: false }
])

// 刷新系统参数
const refreshSystemParams = () => {
  loadingSystemParams.value = true
  // 模拟API请求
  setTimeout(() => {
    ElMessage.success('系统参数已刷新')
    loadingSystemParams.value = false
  }, 1000)
}

// 应用系统参数
const applySystemParams = () => {
  if (!targetReady.value) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  applyingSystemParams.value = true
  // 检查必填参数
  const missingParams = systemParams.value.filter(p => p.required && !p.value.trim())
  if (missingParams.length > 0) {
    ElMessage.error(`缺少必填参数: ${missingParams.map(p => p.key).join(', ')}`)
    applyingSystemParams.value = false
    return
  }

  // 后端 API 待补齐，此处先做交互反馈；调用时会带上 target.nodes / target.osType
  setTimeout(() => {
    ElMessage.success(
      `系统参数已下发到 ${target.value.nodes.length} 个节点（${target.value.osType}），可在作业中心查看进度`
    )
    applyingSystemParams.value = false
  }, 1200)
}
</script>

<style scoped>
.system-param-optimize {
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  margin-bottom: 20px;
  padding-bottom: 10px;
  border-bottom: 1px solid #e5e7eb;
}

.page-header-bar {
  margin-bottom: 6px;
}

.page-header h2 {
  color: #1890ff;
  margin: 0;
  font-size: 20px;
  font-weight: 600;
}

.page-desc {
  margin: 6px 0 0;
  color: #6b7280;
  font-size: 13px;
}

.target-block {
  margin-bottom: 16px;
}

.content-container {
  height: calc(100% - 100px);
  overflow: auto;
}

.content-card {
  max-width: 100%;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
  color: #374151;
  font-size: 16px;
  font-weight: 600;
}

/* 系统参数优化 */
.system-params-container {
  margin-bottom: 20px;
}

.param-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.param-name .key {
  font-weight: 500;
}

/* 卡片操作 */
.card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #e5e7eb;
}
</style>