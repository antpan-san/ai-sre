import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  getMonitoringConfigList,
  getMonitoringConfig,
  createMonitoringConfig,
  updateMonitoringConfig,
  deleteMonitoringConfig,
  getAlertRules,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule
} from '../api/monitoring'
import type { ExporterConfig, AlertRuleGroup } from '../types/monitoring'

export const useMonitoringStore = defineStore('monitoring', () => {
  // 监控配置状态
  const configs = ref<ExporterConfig[]>([])
  const currentConfig = ref<ExporterConfig | null>(null)
  const configLoading = ref<boolean>(false)
  const configTotal = ref<number>(0)
  
  // 告警规则状态
  const alertRules = ref<AlertRuleGroup[]>([])
  const alertRuleLoading = ref<boolean>(false)
  
  // 抽屉状态
  const drawerVisible = ref<boolean>(false)
  const currentDrawerItem = ref<string>('')
  
  // 获取监控配置列表
  // 注意：响应拦截器已解包 code/data/msg，res 直接是 data 部分
  const fetchConfigs = async () => {
    configLoading.value = true
    try {
      const res = await getMonitoringConfigList() as any
      configs.value = res.list || []
      configTotal.value = res.total || 0
      return configs.value
    } catch (error) {
      return []
    } finally {
      configLoading.value = false
    }
  }
  
  // 获取单个监控配置
  const fetchConfig = async (id: string) => {
    configLoading.value = true
    try {
      const res = await getMonitoringConfig(id) as any
      currentConfig.value = res
      return res
    } catch (error) {
      return null
    } finally {
      configLoading.value = false
    }
  }
  
  // 创建监控配置
  const addConfig = async (data: any) => {
    configLoading.value = true
    try {
      const res = await createMonitoringConfig(data) as any
      configs.value.push(res)
      configTotal.value++
      return res
    } catch (error) {
      return null
    } finally {
      configLoading.value = false
    }
  }
  
  // 更新监控配置
  const updateConfig = async (id: string, data: any) => {
    configLoading.value = true
    try {
      const res = await updateMonitoringConfig(id, data) as any
      const index = configs.value.findIndex(item => item.id === id)
      if (index !== -1) {
        configs.value[index] = res
      }
      if (currentConfig.value?.id === id) {
        currentConfig.value = res
      }
      return res
    } catch (error) {
      return null
    } finally {
      configLoading.value = false
    }
  }
  
  // 删除监控配置
  const removeConfig = async (id: string) => {
    configLoading.value = true
    try {
      await deleteMonitoringConfig(id)
      const index = configs.value.findIndex(item => item.id === id)
      if (index !== -1) {
        configs.value.splice(index, 1)
        configTotal.value--
      }
      if (currentConfig.value?.id === id) {
        currentConfig.value = null
      }
      return true
    } catch (error) {
      return false
    } finally {
      configLoading.value = false
    }
  }
  
  // 获取告警规则列表
  const fetchAlertRules = async () => {
    alertRuleLoading.value = true
    try {
      const res = await getAlertRules() as any
      alertRules.value = Array.isArray(res) ? res : []
      return alertRules.value
    } catch (error) {
      return []
    } finally {
      alertRuleLoading.value = false
    }
  }
  
  // 创建告警规则
  const addAlertRule = async (data: any) => {
    alertRuleLoading.value = true
    try {
      const res = await createAlertRule(data) as any
      // 找到对应的规则组并添加新规则
      const ruleGroup = alertRules.value.find(group => group.name === data.ruleGroupName)
      if (ruleGroup) {
        ruleGroup.rules.push(res)
      }
      return res
    } catch (error) {
      return null
    } finally {
      alertRuleLoading.value = false
    }
  }
  
  // 更新告警规则
  const updateAlertRuleById = async (id: string, data: any) => {
    alertRuleLoading.value = true
    try {
      const res = await updateAlertRule(id, data) as any
      // 找到对应的规则并更新
      for (const group of alertRules.value) {
        const index = group.rules.findIndex(rule => rule.id === id)
        if (index !== -1) {
          group.rules[index] = res
          break
        }
      }
      return res
    } catch (error) {
      return null
    } finally {
      alertRuleLoading.value = false
    }
  }
  
  // 删除告警规则
  const removeAlertRule = async (id: string) => {
    alertRuleLoading.value = true
    try {
      await deleteAlertRule(id)
      // 找到对应的规则并删除
      for (const group of alertRules.value) {
        const index = group.rules.findIndex(rule => rule.id === id)
        if (index !== -1) {
          group.rules.splice(index, 1)
          break
        }
      }
      return true
    } catch (error) {
      return false
    } finally {
      alertRuleLoading.value = false
    }
  }
  
  // 设置抽屉可见性
  const setDrawerVisible = (visible: boolean) => {
    drawerVisible.value = visible
  }
  
  // 设置当前抽屉项
  const setCurrentDrawerItem = (item: string) => {
    currentDrawerItem.value = item
    drawerVisible.value = true
  }
  
  // 重置当前配置
  const resetCurrentConfig = () => {
    currentConfig.value = null
  }
  
  // 重置所有状态
  const resetAll = () => {
    configs.value = []
    currentConfig.value = null
    configLoading.value = false
    configTotal.value = 0
    alertRules.value = []
    alertRuleLoading.value = false
    drawerVisible.value = false
    currentDrawerItem.value = ''
  }
  
  return {
    // 监控配置状态
    configs,
    currentConfig,
    configLoading,
    configTotal,
    
    // 告警规则状态
    alertRules,
    alertRuleLoading,
    
    // 抽屉状态
    drawerVisible,
    currentDrawerItem,
    
    // 方法
    fetchConfigs,
    fetchConfig,
    addConfig,
    updateConfig,
    removeConfig,
    fetchAlertRules,
    addAlertRule,
    updateAlertRuleById,
    removeAlertRule,
    setDrawerVisible,
    setCurrentDrawerItem,
    resetCurrentConfig,
    resetAll
  }
})
