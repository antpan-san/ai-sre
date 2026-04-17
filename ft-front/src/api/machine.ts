import request from '../utils/request'
import type { Machine, MachineListParams, MachineListResponse, MachineForm, RegisterWorkerNode, RegisterWorkersResponse } from '../types'

// 获取机器列表（后端返回扁平列表，含 node_role / cluster_id / master_machine_id）
export const getMachineList = (params: MachineListParams): Promise<MachineListResponse> => {
  return request.get('/api/machine', { params })
}

// 获取机器详情
export const getMachineDetail = (id: string): Promise<Machine> => {
  return request.get(`/api/machine/${id}`)
}

// 添加机器
export const addMachine = (data: MachineForm): Promise<Machine> => {
  return request.post('/api/machine', data)
}

// 更新机器
export const updateMachine = (id: string, data: MachineForm): Promise<Machine> => {
  return request.put(`/api/machine/${id}`, data)
}

// 删除机器
export const deleteMachine = (id: string): Promise<void> => {
  return request.delete(`/api/machine/${id}`)
}

// 批量删除机器
export const batchDeleteMachine = (ids: string[]): Promise<void> => {
  return request.delete('/api/machine/batch', { data: { ids } })
}

// 更新机器状态
export const updateMachineStatus = (id: string, status: string): Promise<Machine> => {
  return request.patch(`/api/machine/${id}/status`, { status })
}

// 注册受控节点 (为 master 添加管理的 worker 节点)
export const registerWorkerNodes = (masterId: string, workers: RegisterWorkerNode[]): Promise<RegisterWorkersResponse> => {
  return request.post(`/api/machine/${masterId}/register-workers`, { workers })
}

// 查询任务详情（用于注册节点后轮询执行状态）
export const getTaskDetail = (taskId: string): Promise<any> => {
  return request.get(`/api/task/${taskId}`)
}
