import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DeployConfig, DeployRecord, DeployProgress } from '../types/k8s-deploy'
import { getDeployRecords } from '../api/k8s-deploy'
import { wsService } from '../utils/websocket'

/**
 * 深度合并 source 到 target（用于恢复表单到 reactive 对象）
 * 数组整体替换；对象递归合并；其余按 key 覆盖
 */
function deepMergeInto(target: any, source: any): void {
  if (source == null) return
  for (const key of Object.keys(source)) {
    const s = source[key]
    if (Array.isArray(s)) {
      target[key] = [...s]
    } else if (s !== null && typeof s === 'object' && typeof target[key] === 'object' && target[key] !== null && !Array.isArray(target[key])) {
      deepMergeInto(target[key], s)
    } else {
      target[key] = s
    }
  }
}

export const useK8sDeployStore = defineStore('k8s-deploy', () => {
  const savedDeployConfig = ref<DeployConfig | null>(null)
  const savedActiveStep = ref<number>(0)

  function saveState(config: DeployConfig, step: number) {
    savedDeployConfig.value = JSON.parse(JSON.stringify(config))
    savedActiveStep.value = step
  }

  function restoreInto(config: DeployConfig, activeStepRef: { value: number }) {
    if (savedDeployConfig.value) {
      deepMergeInto(config, savedDeployConfig.value)
    }
    if (savedActiveStep.value != null && savedActiveStep.value >= 0) {
      activeStepRef.value = savedActiveStep.value
    }
  }

  function clearState() {
    savedDeployConfig.value = null
    savedActiveStep.value = 0
  }

  // ---------- 部署记录与 WebSocket 实时进度 ----------
  const deployRecords = ref<DeployRecord[]>([])
  const loadingRecords = ref(false)
  let wsHandlerRegistered = false
  let progressWsHandler: (msg: any) => void = () => {}

  function applyProgressToRecord(progress: DeployProgress): void {
    const id = progress.deployId
    if (!id) return
    const list = deployRecords.value
    const idx = list.findIndex(r => r.deployId === id)
    const patch: Partial<DeployRecord> = {
      status: progress.status,
      progress: progress.progress,
      currentStep: progress.currentStep,
      stepProgress: progress.stepProgress,
      startTime: progress.startTime,
      endTime: progress.endTime,
      error: progress.error
    }
    if (idx >= 0) {
      deployRecords.value = [
        ...list.slice(0, idx),
        { ...list[idx], ...patch },
        ...list.slice(idx + 1)
      ]
    } else {
      deployRecords.value = [
        { deployId: id, clusterName: id.slice(0, 8) + '…', createdAt: new Date().toISOString(), ...patch } as DeployRecord,
        ...list
      ]
    }
  }

  function registerProgressWs() {
    if (wsHandlerRegistered) return
    wsHandlerRegistered = true
    progressWsHandler = (msg: any) => {
      const data = msg?.data
      if (data && data.deployId) applyProgressToRecord(data as DeployProgress)
    }
    wsService.on('k8s_deploy_progress', progressWsHandler)
  }

  function unregisterProgressWs() {
    if (!wsHandlerRegistered) return
    wsHandlerRegistered = false
    wsService.off('k8s_deploy_progress', progressWsHandler)
  }

  async function fetchDeployRecords() {
    loadingRecords.value = true
    registerProgressWs()
    try {
      const list = await getDeployRecords()
      deployRecords.value = Array.isArray(list) ? list : []
    } catch (e) {
      console.error('获取部署记录失败:', e)
      deployRecords.value = []
    } finally {
      loadingRecords.value = false
    }
  }

  /** 当前正在进行的部署（用于第一步「正在部署」展示） */
  function getRunningDeploy(): DeployRecord | null {
    return deployRecords.value.find(
      r => r.status === 'running' || r.status === 'pending'
    ) ?? null
  }

  return {
    savedDeployConfig,
    savedActiveStep,
    saveState,
    restoreInto,
    clearState,
    deployRecords,
    loadingRecords,
    fetchDeployRecords,
    getRunningDeploy,
    applyProgressToRecord,
    registerProgressWs,
    unregisterProgressWs
  }
})
