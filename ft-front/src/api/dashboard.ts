import request from '../utils/request'
import type { GetDashboardDataResponse } from '../types/dashboard'

/**
 * 获取仪表盘数据
 * @returns 仪表盘数据
 */
export const getDashboardData = (): Promise<GetDashboardDataResponse> => {
  return request.get('/api/dashboard/data')
}
