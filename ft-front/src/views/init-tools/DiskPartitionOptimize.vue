<template>
  <div class="disk-partition-optimize">
    <div class="page-header">
      <div class="page-header-bar">
        <el-button link type="primary" :icon="ArrowLeft" @click="backToHome">返回工具总览</el-button>
      </div>
      <h2>磁盘分区优化</h2>
      <p>优化磁盘分区配置、文件系统挂载选项与 swap，缓解 etcd fsync 抖动。请先选择目标节点与系统类型。</p>
    </div>

    <div class="content-container">
      <el-card class="content-card">
        <template #header>
          <div class="card-header">
            <h3>磁盘分区优化配置</h3>
          </div>
        </template>

        <NodeSystemSelector v-model="target" class="target-block" />

        <div class="disk-optimize-container">
          <div class="disk-info">
            <el-alert
              title="注意"
              type="warning"
              show-icon
              :closable="false"
              class="disk-alert"
            >
              磁盘分区优化可能会导致数据丢失，请确保已备份重要数据
            </el-alert>
          </div>

          <div class="disk-config">
            <h4>优化选项</h4>
            <el-checkbox-group v-model="diskOptions">
              <div class="disk-option-item">
                <el-checkbox label="enable_ssd_trim">启用SSD TRIM支持</el-checkbox>
                <el-tooltip content="启用SSD TRIM，提高SSD性能和寿命" placement="top">
                  <el-icon class="help-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
              </div>
              <div class="disk-option-item">
                <el-checkbox label="tune_filesystem">优化文件系统参数</el-checkbox>
                <el-tooltip content="优化EXT4/XFS等文件系统参数" placement="top">
                  <el-icon class="help-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
              </div>
              <div class="disk-option-item">
                <el-checkbox label="setup_swap">配置Swap分区</el-checkbox>
                <el-tooltip content="配置系统Swap分区大小" placement="top">
                  <el-icon class="help-icon"><QuestionFilled /></el-icon>
                </el-tooltip>
                <el-select
                  v-if="diskOptions.includes('setup_swap')"
                  v-model="swapSize"
                  style="width: 150px; margin-left: 10px"
                >
                  <el-option label="1GB" value="1G" />
                  <el-option label="2GB" value="2G" />
                  <el-option label="4GB" value="4G" />
                  <el-option label="8GB" value="8G" />
                  <el-option label="16GB" value="16G" />
                  <el-option label="自动(内存2倍)" value="auto" />
                </el-select>
              </div>
            </el-checkbox-group>
          </div>

          <div class="disk-selection">
            <h4>选择磁盘</h4>
            <el-select
              v-model="selectedDisks"
              multiple
              filterable
              collapse-tags
              placeholder="请选择要优化的磁盘"
              style="width: 100%"
            >
              <el-option
                v-for="disk in availableDisks"
                :key="disk.name"
                :label="`${disk.name} - ${disk.size}GB (${disk.fs})`"
                :value="disk.name"
              />
            </el-select>
            <div class="disk-count">已选择 {{ selectedDisks.length }} 个磁盘</div>
          </div>
        </div>

        <div class="card-actions">
          <el-button
            type="success"
            @click="optimizeDisks"
            :disabled="selectedDisks.length === 0 || diskOptions.length === 0 || !targetReady"
            :loading="optimizingDisks"
          >
            <el-icon><Calendar /></el-icon>
            优化磁盘
          </el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, QuestionFilled, ArrowLeft } from '@element-plus/icons-vue'
import NodeSystemSelector, { type NodeSystemValue } from '../../components/init-tools/NodeSystemSelector.vue'

const route = useRoute()
const router = useRouter()

// 磁盘分区优化
const diskOptions = ref<string[]>([])
const selectedDisks = ref<string[]>([])
const optimizingDisks = ref(false)
const swapSize = ref('auto')

const target = ref<NodeSystemValue>({ nodes: [], osType: '' })
const targetReady = computed(() => target.value.nodes.length > 0 && !!target.value.osType)

onMounted(() => {
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

const availableDisks = ref([
  { name: '/dev/sda', size: 100, fs: 'ext4' },
  { name: '/dev/sdb', size: 200, fs: 'xfs' },
  { name: '/dev/sdc', size: 500, fs: 'ext4' }
])

// 优化磁盘
const optimizeDisks = () => {
  if (!targetReady.value) {
    ElMessage.warning('请先选择目标节点与系统类型')
    return
  }
  ElMessageBox.confirm(
    `将向 ${target.value.nodes.length} 个节点（${target.value.osType}）执行磁盘优化，可能导致数据丢失，请确保已备份重要数据，是否继续？`,
    '危险操作',
    { type: 'error', confirmButtonText: '确认优化', cancelButtonText: '取消' }
  ).then(() => {
    optimizingDisks.value = true
    // 后端 API 待补齐，此处先做交互反馈
    setTimeout(() => {
      ElMessage.success('磁盘优化任务已下发')
      optimizingDisks.value = false
    }, 1500)
  }).catch(() => {
    // 取消操作
  })
}
</script>

<style scoped>
.disk-partition-optimize {
  padding: 20px;
  box-sizing: border-box;
}

.page-header {
  margin-bottom: 30px;
}

.page-header-bar {
  margin-bottom: 6px;
}

.page-header h2 {
  color: #1890ff;
  margin-bottom: 10px;
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

/* 磁盘分区优化 */
.disk-optimize-container {
  margin-bottom: 20px;
}

.disk-alert {
  margin-bottom: 20px;
}

.disk-option-item {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.disk-selection {
  margin-top: 20px;
}

.disk-selection h4 {
  margin: 0 0 15px 0;
  color: #374151;
  font-size: 14px;
  font-weight: 600;
}

.help-icon {
  margin-left: 5px;
  color: #9ca3af;
  cursor: help;
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