import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import {
  getProxyConfigList,
  saveProxyConfig,
  deleteProxyConfig,
  applyProxyConfig
} from '../api/proxy'
import type {
  ProxyConfig,
  GetProxyConfigListParams,
  SaveProxyConfigParams,
} from '../types/proxy'

export const useProxyStore = defineStore('proxy', () => {
  // 状态
  const proxyConfigList = ref<ProxyConfig[]>([])
  const currentConfig = ref<ProxyConfig | null>(null)
  const total = ref<number>(0)
  const loading = ref<boolean>(false)
  const filters = reactive<GetProxyConfigListParams>({
    page: 1,
    pageSize: 10,
    name: '',
    status: undefined
  })

  // 获取代理配置列表
  const fetchProxyConfigList = async () => {
    loading.value = true
    try {
      const res = await getProxyConfigList(filters) as any
      proxyConfigList.value = res.list || []
      total.value = res.total || 0
      return res
    } catch (error) {
      console.error('获取代理配置列表失败:', error)
      return null
    } finally {
      loading.value = false
    }
  }

  // 保存新代理配置
  const saveNewProxyConfig = async (data: SaveProxyConfigParams) => {
    try {
      const res = await saveProxyConfig(data)
      await fetchProxyConfigList()
      return res
    } catch (error) {
      console.error('保存代理配置失败:', error)
      return null
    }
  }

  // 更新代理配置
  const updateExistingProxyConfig = async (id: string, data: SaveProxyConfigParams) => {
    try {
      const saveData = { ...data, id }
      const res = await saveProxyConfig(saveData)
      await fetchProxyConfigList()
      return res
    } catch (error) {
      console.error('更新代理配置失败:', error)
      return null
    }
  }

  // 删除代理配置
  const removeProxyConfig = async (id: string) => {
    try {
      await deleteProxyConfig({ id })
      await fetchProxyConfigList()
      return true
    } catch (error) {
      console.error('删除代理配置失败:', error)
      return false
    }
  }

  // 应用代理配置
  const applyExistingProxyConfig = async (id: string) => {
    try {
      const res = await applyProxyConfig({ id })
      return res
    } catch (error) {
      console.error('应用代理配置失败:', error)
      return null
    }
  }

  // 设置当前配置
  const setCurrentConfig = (config: ProxyConfig | null) => {
    currentConfig.value = config
  }

  // 重置过滤器
  const resetFilters = () => {
    filters.page = 1
    filters.pageSize = 10
    filters.name = ''
    filters.status = undefined
  }

  return {
    proxyConfigList,
    currentConfig,
    total,
    loading,
    filters,
    fetchProxyConfigList,
    saveNewProxyConfig,
    updateExistingProxyConfig,
    removeProxyConfig,
    applyExistingProxyConfig,
    setCurrentConfig,
    resetFilters
  }
})
