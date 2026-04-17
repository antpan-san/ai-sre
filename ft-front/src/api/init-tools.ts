import request from '../utils/request'

/**
 * 获取系统参数优化配置
 * @returns 系统参数列表
 */
export const getSystemParams = (): Promise<any> => {
  return request.get('/api/init-tools/system-params')
}

/**
 * 更新系统参数优化配置
 * @param data 系统参数更新数据
 * @returns 更新结果
 */
export const updateSystemParams = (data: {
  machineIds: number[]
  params: Record<string, string>
}): Promise<any> => {
  return request.post('/api/init-tools/system-params', data)
}

/**
 * 执行时间同步操作
 * @param data 时间同步参数
 * @returns 同步结果
 */
export const syncTime = (data: {
  machineIds: number[]
  ntpServer?: string
}): Promise<any> => {
  return request.post('/api/init-tools/time-sync', data)
}

/**
 * 执行系统安全加固
 * @param data 安全加固参数
 * @returns 加固结果
 */
export const hardenSecurity = (data: {
  machineIds: number[]
  options?: Record<string, boolean>
}): Promise<any> => {
  return request.post('/api/init-tools/security-harden', data)
}

/**
 * 执行磁盘分区优化
 * @param data 磁盘优化参数
 * @returns 优化结果
 */
export const optimizeDisk = (data: {
  machineIds: number[]
  options?: Record<string, any>
}): Promise<any> => {
  return request.post('/api/init-tools/disk-optimize', data)
}
