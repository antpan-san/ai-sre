import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getDashboardData } from '../api/dashboard'
import type { DashboardData } from '../types/dashboard'

export const useDashboardStore = defineStore('dashboard', () => {
  // 状态
  const dashboardData = ref<DashboardData | null>(null)
  const loading = ref<boolean>(false)

  // 获取仪表盘数据
  const fetchDashboardData = async () => {
    loading.value = true
    try {
      const res = await getDashboardData()
      // 响应拦截器已解包 data，res 即 { code, data, msg } 中的 data
      dashboardData.value = res as unknown as DashboardData
      return dashboardData.value
    } catch (error) {
      console.error('获取仪表盘数据失败:', error)
      return null
    } finally {
      loading.value = false
    }
  }

  // 重置状态
  const resetAll = () => {
    dashboardData.value = null
    loading.value = false
  }

  return {
    dashboardData,
    loading,
    fetchDashboardData,
    resetAll
  }
})
