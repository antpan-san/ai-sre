<template>
  <div class="k8s-deploy-progress">
    <div class="page-header">
      <h2>Kubernetes 集群部署进度</h2>
      <p class="page-desc">实时查看集群部署进度和日志信息</p>
    </div>

    <!-- 部署状态卡片 -->
    <div class="progress-card">
      <div class="progress-card-header">
        <div class="progress-card-indicator">
          <el-icon :size="20">
            <Loading v-if="deployProgress.status === 'running' || deployProgress.status === 'pending'" />
            <CircleCheckFilled v-else-if="deployProgress.status === 'success'" />
            <CircleCloseFilled v-else />
          </el-icon>
        </div>
        <div class="progress-card-meta">
          <h3 class="progress-card-title">部署状态</h3>
          <p class="progress-card-desc">{{ deployProgress.currentStep || '准备中...' }}</p>
        </div>
        <div class="progress-card-actions">
          <el-button
            v-if="isRunning"
            type="danger"
            size="small"
            :loading="cancelling"
            @click="handleTerminateDeploy"
          >
            终止
          </el-button>
          <el-button
            v-if="deployProgress.status === 'success'"
            type="primary"
            size="small"
            @click="goToClusterList"
          >
            查看集群
          </el-button>
          <el-button
            v-if="!isRunning"
            size="small"
            @click="refreshProgressOnce"
          >
            刷新
          </el-button>
        </div>
      </div>

      <div class="progress-card-body">
        <!-- 总体进度 -->
        <div class="progress-section">
          <div class="progress-info">
            <span class="progress-title">总体进度</span>
            <span class="progress-value">{{ deployProgress.progress }}%</span>
          </div>
          <el-progress
            :percentage="deployProgress.progress"
            :status="progressBarStatus"
            :stroke-width="18"
          />
        </div>

        <!-- 状态信息 -->
        <div class="status-grid">
          <div class="status-cell">
            <span class="status-label">当前状态</span>
            <el-tag :type="statusTagType" size="default" effect="dark">
              {{ statusText }}
            </el-tag>
          </div>
          <div class="status-cell">
            <span class="status-label">当前步骤</span>
            <span class="status-value">{{ deployProgress.currentStep || '准备中' }}</span>
          </div>
          <div class="status-cell status-cell-wide">
            <span class="status-label">步骤进度</span>
            <el-progress
              :percentage="deployProgress.stepProgress"
              :stroke-width="10"
              :status="deployProgress.stepProgress === 100 ? 'success' : undefined"
              style="flex: 1"
            />
          </div>
        </div>

        <!-- 时间信息 -->
        <div class="time-grid">
          <div class="time-cell">
            <span class="time-label">开始时间</span>
            <span class="time-value">{{ deployProgress.startTime ? formatTime(deployProgress.startTime) : '未开始' }}</span>
          </div>
          <div class="time-cell">
            <span class="time-label">结束时间</span>
            <span class="time-value">{{ deployProgress.endTime ? formatTime(deployProgress.endTime) : '进行中' }}</span>
          </div>
          <div class="time-cell">
            <span class="time-label">耗时</span>
            <span class="time-value time-duration">{{ duration }}</span>
          </div>
        </div>

        <!-- 失败提示 -->
        <el-alert
          v-if="deployProgress.status === 'failed'"
          title="部署失败"
          :description="deployProgress.error || '未知错误'"
          type="error"
          show-icon
          class="result-alert"
        />

        <!-- 成功提示 -->
        <template v-if="deployProgress.status === 'success'">
          <el-alert title="部署成功" type="success" show-icon class="result-alert">
            Kubernetes 集群已成功部署。
          </el-alert>
          <div class="cluster-info-box">
            <div class="cluster-info-row">
              <span class="cluster-info-label">Kubeconfig 文件</span>
              <el-button type="primary" size="small" @click="downloadKubeconfig">下载</el-button>
            </div>
            <div class="cluster-info-row">
              <span class="cluster-info-label">API Server 地址</span>
              <span class="cluster-info-value">{{ clusterApiServerAddress || '--' }}</span>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- 部署日志卡片 -->
    <div class="log-card">
      <div class="log-card-header">
        <div class="log-card-indicator">
          <el-icon :size="18"><Document /></el-icon>
        </div>
        <h3 class="log-card-title">部署日志</h3>
        <div class="log-card-actions">
          <el-button size="small" :loading="loadingLogs" @click="loadLogs(true)">刷新日志</el-button>
          <el-button size="small" @click="clearLogs">清空</el-button>
        </div>
      </div>

      <div class="log-body" ref="logContainerRef">
        <div
          v-for="(log, index) in deployLogs"
          :key="index"
          :class="['log-line', `log-line--${log.level}`]"
        >
          <span class="log-time">{{ formatTime(log.timestamp) }}</span>
          <span :class="['log-badge', `log-badge--${log.level}`]">{{ logLevelText(log.level) }}</span>
          <span v-if="log.step" class="log-step">[{{ log.step }}]</span>
          <span class="log-msg">{{ log.message }}</span>
        </div>
        <div v-if="loadingLogs" class="log-loading">
          <el-skeleton :rows="3" animated />
        </div>
        <div v-if="!loadingLogs && deployLogs.length === 0" class="log-empty">
          暂无日志信息
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Loading,
  CircleCheckFilled,
  CircleCloseFilled,
  Document
} from '@element-plus/icons-vue'
import {
  getDeployProgress,
  getDeployLogs,
  terminateDeploy
} from '../../../api/k8s-deploy'
import type { DeployProgress, DeployLog } from '../../../types/k8s-deploy'
import { wsService } from '../../../utils/websocket'

const router = useRouter()
const route = useRoute()

// ---------- 状态 ----------
const loadingLogs = ref(false)
const cancelling = ref(false)
const deployId = ref<string | null>(null)
const deployProgress = reactive<DeployProgress>({
  progress: 0,
  status: 'pending',
  currentStep: '',
  stepProgress: 0
})
const deployLogs = ref<DeployLog[]>([])
const logOffset = ref(0)
const logContainerRef = ref<HTMLElement | null>(null)
const clusterApiServerAddress = ref('')
let progressWsHandler: (msg: any) => void = () => {}

// ---------- 计算属性 ----------
const isRunning = computed(() =>
  deployProgress.status === 'running' || deployProgress.status === 'pending'
)

const progressBarStatus = computed(() => {
  if (deployProgress.status === 'success') return 'success'
  if (deployProgress.status === 'failed' || deployProgress.status === 'cancelled') return 'exception'
  return undefined
})

const statusTagType = computed<'success' | 'danger' | 'warning' | 'info'>(() => {
  const m: Record<string, 'success' | 'danger' | 'warning' | 'info'> = {
    success: 'success',
    failed: 'danger',
    cancelled: 'warning',
    running: 'info',
    pending: 'info'
  }
  return m[deployProgress.status] || 'info'
})

const statusText = computed(() => {
  const m: Record<string, string> = {
    pending: '待开始',
    running: '部署中',
    success: '部署成功',
    failed: '部署失败',
    cancelled: '已取消'
  }
  return m[deployProgress.status] || '未知状态'
})

const duration = computed(() => {
  if (!deployProgress.startTime) return '00:00:00'
  const start = new Date(deployProgress.startTime).getTime()
  const end = deployProgress.endTime
    ? new Date(deployProgress.endTime).getTime()
    : Date.now()
  const d = end - start
  const h = Math.floor(d / 3600000)
  const m = Math.floor((d % 3600000) / 60000)
  const s = Math.floor((d % 60000) / 1000)
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
})

// ---------- 生命周期 ----------
onMounted(() => {
  const id = route.query.deployId as string
  if (!id) {
    ElMessage.error('部署 ID 不存在')
    router.push('/service/k8s-deploy')
    return
  }
  deployId.value = id
  refreshProgressOnce()
  registerProgressWs()
  loadLogs()
})

onUnmounted(() => {
  unregisterProgressWs()
})

// ---------- 进度：首次 HTTP 拉取 + 后续 WebSocket 推送（无轮询） ----------
const refreshProgressOnce = async () => {
  if (!deployId.value) return
  try {
    const res = await getDeployProgress({ deployId: deployId.value })
    Object.assign(deployProgress, res)
  } catch (e: any) {
    ElMessage.error('获取部署进度失败: ' + (e.msg || e.message))
  }
}

function registerProgressWs() {
  const id = deployId.value
  if (!id) return
  progressWsHandler = (msg: any) => {
    const data = msg?.data
    if (data && data.deployId === id) {
      Object.assign(deployProgress, {
        progress: data.progress ?? deployProgress.progress,
        status: data.status ?? deployProgress.status,
        currentStep: data.currentStep ?? deployProgress.currentStep,
        stepProgress: data.stepProgress ?? deployProgress.stepProgress,
        startTime: data.startTime ?? deployProgress.startTime,
        endTime: data.endTime ?? deployProgress.endTime,
        error: data.error ?? deployProgress.error,
        totalCount: data.totalCount ?? deployProgress.totalCount,
        successCount: data.successCount ?? deployProgress.successCount,
        failedCount: data.failedCount ?? deployProgress.failedCount
      })
    }
  }
  wsService.on('k8s_deploy_progress', progressWsHandler)
}

function unregisterProgressWs() {
  wsService.off('k8s_deploy_progress', progressWsHandler)
}

const loadLogs = async (reset = false) => {
  if (!deployId.value || loadingLogs.value) return
  if (reset) {
    logOffset.value = 0
    deployLogs.value = []
  }
  loadingLogs.value = true
  try {
    const res = await getDeployLogs({ deployId: deployId.value, offset: logOffset.value, limit: 100 })
    const newLogs = (res as any).logs || []
    if (newLogs.length > 0) {
      deployLogs.value = reset ? newLogs : [...deployLogs.value, ...newLogs]
      logOffset.value += newLogs.length
      scrollToBottom()
    }
  } catch (e: any) {
    ElMessage.error('获取日志失败: ' + (e.msg || e.message))
  } finally {
    loadingLogs.value = false
  }
}

const scrollToBottom = () => {
  if (logContainerRef.value) {
    requestAnimationFrame(() => {
      logContainerRef.value!.scrollTop = logContainerRef.value!.scrollHeight
    })
  }
}

// ---------- 操作 ----------
const handleTerminateDeploy = () => {
  ElMessageBox.confirm(
    '终止后将下发清理任务，client 端会清理已部署内容并严格恢复到部署前状态。确定继续？',
    '终止部署',
    {
      confirmButtonText: '确定终止',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(async () => {
    if (!deployId.value) return
    cancelling.value = true
    try {
      await terminateDeploy(deployId.value)
      ElMessage.success('已终止部署，清理任务已下发')
      refreshProgressOnce()
    } catch (e: any) {
      ElMessage.error('终止失败: ' + (e.msg || e.message))
    } finally {
      cancelling.value = false
    }
  }).catch(() => {})
}

const clearLogs = () => {
  deployLogs.value = []
  logOffset.value = 0
}

const goToClusterList = () => {
  router.push('/service/k8s-deploy')
}

const downloadKubeconfig = () => {
  ElMessage.info('kubeconfig 下载功能开发中')
}

// ---------- 辅助 ----------
const formatTime = (t: string) => {
  if (!t) return ''
  try {
    return new Date(t).toLocaleString('zh-CN')
  } catch {
    return t
  }
}

const logLevelText = (l: string) => {
  const m: Record<string, string> = { info: 'INFO', warning: 'WARN', error: 'ERROR' }
  return m[l] || 'INFO'
}
</script>

<style scoped>
/* ==================== 页面布局 ==================== */
.k8s-deploy-progress {
  width: 100%;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
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

/* ==================== 进度卡片（与 DeployForm 统一） ==================== */
.progress-card,
.log-card {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.progress-card-header {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 28px;
  border-bottom: 1px solid #f0f0f0;
  background: linear-gradient(135deg, var(--mi-surface-warm-a) 0%, var(--mi-surface-warm-b) 100%);
}

.progress-card-indicator {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-dark-2));
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.progress-card-meta {
  flex: 1;
}

.progress-card-title {
  margin: 0 0 2px 0;
  font-size: 17px;
  font-weight: 600;
  color: #1f2937;
}

.progress-card-desc {
  margin: 0;
  font-size: 13px;
  color: #9ca3af;
}

.progress-card-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.progress-card-body {
  padding: 28px;
}

/* 总体进度 */
.progress-section {
  margin-bottom: 28px;
}

.progress-info {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
}

.progress-title {
  font-size: 15px;
  font-weight: 600;
  color: #374151;
}

.progress-value {
  font-size: 22px;
  font-weight: 700;
  color: var(--el-color-primary);
}

/* 状态网格 */
.status-grid {
  display: grid;
  grid-template-columns: auto auto 1fr;
  gap: 20px 32px;
  align-items: center;
  margin-bottom: 24px;
  padding: 16px 20px;
  background: #f9fafb;
  border-radius: 10px;
  border: 1px solid #f0f0f0;
}

.status-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-cell-wide {
  gap: 12px;
}

.status-label {
  font-weight: 500;
  color: #6b7280;
  font-size: 13px;
  white-space: nowrap;
}

.status-value {
  color: #374151;
  font-size: 13px;
}

/* 时间网格 */
.time-grid {
  display: flex;
  gap: 32px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.time-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.time-label {
  font-weight: 500;
  color: #6b7280;
  font-size: 13px;
}

.time-value {
  color: #374151;
  font-size: 13px;
}

.time-duration {
  font-family: 'JetBrains Mono', 'Consolas', monospace;
  font-weight: 600;
  color: var(--el-color-primary);
}

/* 结果提示 */
.result-alert {
  margin-bottom: 16px;
}

.cluster-info-box {
  padding: 16px 20px;
  background: #f0f9eb;
  border: 1px solid #b7eb8f;
  border-radius: 10px;
}

.cluster-info-row {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.cluster-info-row:last-child {
  margin-bottom: 0;
}

.cluster-info-label {
  font-weight: 500;
  min-width: 140px;
  color: #6b7280;
  font-size: 13px;
}

.cluster-info-value {
  color: #374151;
  font-size: 13px;
}

/* ==================== 日志卡片 ==================== */
.log-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 28px;
  border-bottom: 1px solid #f0f0f0;
  background: linear-gradient(135deg, var(--mi-surface-warm-a) 0%, var(--mi-surface-warm-b) 100%);
}

.log-card-indicator {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-primary-dark-2));
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.log-card-title {
  flex: 1;
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.log-card-actions {
  flex-shrink: 0;
}

.log-body {
  max-height: 500px;
  overflow-y: auto;
  padding: 16px 20px;
  background: #1e1e2e;
  border-radius: 0 0 12px 12px;
  font-family: 'JetBrains Mono', 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-line {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 4px;
  padding: 2px 0;
}

.log-time {
  color: #6c7086;
  min-width: 170px;
  flex-shrink: 0;
}

.log-badge {
  font-weight: 700;
  min-width: 48px;
  text-align: center;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 11px;
  flex-shrink: 0;
}

.log-badge--info {
  color: #89b4fa;
  background: rgba(137, 180, 250, 0.15);
}

.log-badge--warning {
  color: #f9e2af;
  background: rgba(249, 226, 175, 0.15);
}

.log-badge--error {
  color: #f38ba8;
  background: rgba(243, 139, 168, 0.15);
}

.log-step {
  color: #89dceb;
  font-weight: 600;
  flex-shrink: 0;
}

.log-msg {
  color: #cdd6f4;
  word-break: break-all;
  flex: 1;
}

.log-line--error .log-msg {
  color: #f38ba8;
}

.log-line--warning .log-msg {
  color: #f9e2af;
}

.log-loading {
  padding: 12px 0;
}

.log-empty {
  text-align: center;
  color: #6c7086;
  padding: 32px;
}
</style>
