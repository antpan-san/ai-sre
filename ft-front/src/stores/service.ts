import { defineStore } from 'pinia'
import { ref, reactive } from 'vue'
import {
  deployService,
  getServiceList,
  getLinuxServiceList,
  operateLinuxService
} from '../api/service'
import type {
  DeployServiceParams,
  ServiceInfo,
  LinuxServiceInfo,
  GetLinuxServiceListParams,
  LinuxServiceOperationParams
} from '../types/service'

export const useServiceStore = defineStore('service', () => {
  // 服务部署相关状态
  const serviceList = ref<ServiceInfo[]>([])
  const serviceTotal = ref<number>(0)
  const serviceLoading = ref<boolean>(false)

  // Linux 服务相关状态
  const linuxServiceList = ref<LinuxServiceInfo[]>([])
  const linuxTotal = ref<number>(0)
  const linuxLoading = ref<boolean>(false)
  const linuxFilters = reactive<GetLinuxServiceListParams>({
    page: 1,
    pageSize: 10,
    name: '',
    status: undefined,
    machineId: undefined
  })

  // 部署新服务
  const deployNewService = async (data: DeployServiceParams) => {
    serviceLoading.value = true
    try {
      const res = await deployService(data)
      return res
    } catch (error) {
      console.error('部署服务失败:', error)
      return null
    } finally {
      serviceLoading.value = false
    }
  }

  // 获取服务列表
  const fetchServiceList = async (params?: Record<string, any>) => {
    serviceLoading.value = true
    try {
      const res = await getServiceList(params) as any
      serviceList.value = res.list || []
      serviceTotal.value = res.total || 0
      return res
    } catch (error) {
      console.error('获取服务列表失败:', error)
      return null
    } finally {
      serviceLoading.value = false
    }
  }

  // 获取 Linux 服务列表
  const fetchLinuxServiceList = async () => {
    linuxLoading.value = true
    try {
      const res = await getLinuxServiceList(linuxFilters) as any
      linuxServiceList.value = res.list || []
      linuxTotal.value = res.total || 0
      return res
    } catch (error) {
      console.error('获取Linux服务列表失败:', error)
      return null
    } finally {
      linuxLoading.value = false
    }
  }

  // Linux 服务操作
  const handleLinuxServiceOperation = async (params: LinuxServiceOperationParams) => {
    try {
      const res = await operateLinuxService(params)
      return res
    } catch (error) {
      console.error('Linux服务操作失败:', error)
      return null
    }
  }

  // 设置 Linux 服务过滤器
  const setLinuxFilters = (newFilters: Partial<GetLinuxServiceListParams>) => {
    Object.assign(linuxFilters, newFilters)
  }

  // 重置 Linux 服务过滤器
  const resetLinuxFilters = () => {
    linuxFilters.page = 1
    linuxFilters.pageSize = 10
    linuxFilters.name = ''
    linuxFilters.status = undefined
    linuxFilters.machineId = undefined
  }

  return {
    // 服务部署
    serviceList,
    serviceTotal,
    serviceLoading,
    deployNewService,
    fetchServiceList,

    // Linux 服务
    linuxServiceList,
    linuxTotal,
    linuxLoading,
    linuxFilters,
    fetchLinuxServiceList,
    handleLinuxServiceOperation,
    setLinuxFilters,
    resetLinuxFilters
  }
})
