import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { DeployConfig } from '../types/k8s-deploy'

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
      if (!config.clusterBasicInfo.cpuArch) {
        config.clusterBasicInfo.cpuArch = 'arm64'
      }
    }
    if (savedActiveStep.value != null && savedActiveStep.value >= 0) {
      activeStepRef.value = savedActiveStep.value
    }
  }

  function clearState() {
    savedDeployConfig.value = null
    savedActiveStep.value = 0
  }

  return {
    savedDeployConfig,
    savedActiveStep,
    saveState,
    restoreInto,
    clearState
  }
})
