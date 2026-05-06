import request from '../utils/request'
import type {
  DeployServiceParams,
  DeployServiceResponse,
  GetServiceListParams,
  GetServiceListResponse,
  GetServiceDetailParams,
  GetServiceDetailResponse,
  ServiceOperationParams,
  ServiceOperationResponse,
  BatchDeleteServicesParams,
  BatchDeleteServicesResponse,
  GetLinuxServiceListParams,
  GetLinuxServiceListResponse,
  LinuxServiceOperationParams,
  LinuxServiceOperationResponse,
  CreateServiceDeploymentParams,
  CreateServiceDeploymentResponse
} from '../types/service'

/**
 * 部署服务
 * @param data 部署参数
 * @returns 部署结果
 */
export const deployService = (data: DeployServiceParams): Promise<DeployServiceResponse> => {
  return request.post('/api/service/deploy', data)
}

export const createServiceDeployment = (data: CreateServiceDeploymentParams): Promise<CreateServiceDeploymentResponse> => {
  return request.post('/api/service-deploy/deployments', data)
}

/**
 * 获取服务列表
 * @param params 查询参数
 * @returns 服务列表
 */
export const getServiceList = (params?: GetServiceListParams): Promise<GetServiceListResponse> => {
  return request.get('/api/service/list', { params })
}

/**
 * 获取服务详情
 * @param params 查询参数
 * @returns 服务详情
 */
export const getServiceDetail = (params: GetServiceDetailParams): Promise<GetServiceDetailResponse> => {
  return request.get('/api/service/detail', { params })
}

/**
 * 启动服务
 * @param params 服务ID
 * @returns 操作结果
 */
export const startService = (params: ServiceOperationParams): Promise<ServiceOperationResponse> => {
  return request.post('/api/service/start', params)
}

/**
 * 停止服务
 * @param params 服务ID
 * @returns 操作结果
 */
export const stopService = (params: ServiceOperationParams): Promise<ServiceOperationResponse> => {
  return request.post('/api/service/stop', params)
}

/**
 * 重启服务
 * @param params 服务ID
 * @returns 操作结果
 */
export const restartService = (params: ServiceOperationParams): Promise<ServiceOperationResponse> => {
  return request.post('/api/service/restart', params)
}

/**
 * 删除服务
 * @param params 服务ID
 * @returns 操作结果
 */
export const deleteService = (params: ServiceOperationParams): Promise<ServiceOperationResponse> => {
  return request.delete('/api/service/delete', { params })
}

/**
 * 批量删除服务
 * @param data 服务ID列表
 * @returns 操作结果
 */
export const batchDeleteServices = (data: BatchDeleteServicesParams): Promise<BatchDeleteServicesResponse> => {
  return request.post('/api/service/batch-delete', data)
}

/**
 * 获取Linux服务列表
 * @param params 查询参数
 * @returns Linux服务列表
 */
export const getLinuxServiceList = (params?: GetLinuxServiceListParams): Promise<GetLinuxServiceListResponse> => {
  return request.get('/api/service/linux/list', { params })
}

/**
 * Linux服务操作（启动/停止/重启/启用/禁用）
 * @param data 操作参数
 * @returns 操作结果
 */
export const operateLinuxService = (data: LinuxServiceOperationParams): Promise<LinuxServiceOperationResponse> => {
  return request.post('/api/service/linux/operate', data)
}