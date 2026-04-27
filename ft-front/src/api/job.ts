import request from '../utils/request'
import type { Machine } from '../types'

/**
 * 获取当前用户权限下的可用机器列表
 * @returns 机器列表
 */
export const getAvailableMachines = (): Promise<Machine[]> => {
  return request.get('/api/job/machines')
}

/**
 * 在选择的机器上执行命令
 * @param data 执行参数
 * @returns 执行结果
 */
export const executeCommand = (data: {
  machine_ids: string[]
  command: string
  timeout?: number
}): Promise<{ jobId: string }> => {
  return request.post('/api/job/execute', data)
}

/**
 * 获取命令执行结果
 * @param jobId 作业ID
 * @returns 执行结果
 */
export const getExecutionResult = (jobId: string): Promise<any> => {
  return request.get(`/api/job/result/${jobId}`)
}
