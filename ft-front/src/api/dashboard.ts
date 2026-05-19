import request from '../utils/request'
import type { DashboardHostResources, GetDashboardDataResponse } from '../types/dashboard'

/**
 * 获取仪表盘数据
 * @returns 仪表盘数据
 */
export const getDashboardData = (): Promise<GetDashboardDataResponse> => {
  return request.get('/api/dashboard/data')
}

/** 导航栏资源圆环：仅 CPU/内存/磁盘，轻量轮询 */
export const getDashboardHostResources = (): Promise<DashboardHostResources> => {
  return request.get('/api/dashboard/host-resources', { silent: true })
}
