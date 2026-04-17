import request from '../utils/request'
import type {
  GetProxyConfigListParams,
  GetProxyConfigListResponse,
  SaveProxyConfigParams,
  SaveProxyConfigResponse,
  GetProxyConfigDetailParams,
  GetProxyConfigDetailResponse,
  DeleteProxyConfigParams,
  DeleteProxyConfigResponse,
  ApplyProxyConfigParams,
  ApplyProxyConfigResponse
} from '../types/proxy'

/**
 * 获取代理配置列表
 * @param params 查询参数
 * @returns 代理配置列表
 */
export const getProxyConfigList = (params?: GetProxyConfigListParams): Promise<GetProxyConfigListResponse> => {
  return request.get('/api/proxy/config/list', { params })
}

/**
 * 获取代理配置详情
 * @param params 查询参数
 * @returns 代理配置详情
 */
export const getProxyConfigDetail = (params: GetProxyConfigDetailParams): Promise<GetProxyConfigDetailResponse> => {
  return request.get('/api/proxy/config/detail', { params })
}

/**
 * 保存代理配置
 * @param data 配置数据
 * @returns 保存结果
 */
export const saveProxyConfig = (data: SaveProxyConfigParams): Promise<SaveProxyConfigResponse> => {
  return request.post('/api/proxy/config/save', data)
}

/**
 * 删除代理配置
 * @param params 查询参数
 * @returns 删除结果
 */
export const deleteProxyConfig = (params: DeleteProxyConfigParams): Promise<DeleteProxyConfigResponse> => {
  return request.delete('/api/proxy/config/delete', { params })
}

/**
 * 应用代理配置
 * @param data 配置数据
 * @returns 应用结果
 */
export const applyProxyConfig = (data: ApplyProxyConfigParams): Promise<ApplyProxyConfigResponse> => {
  return request.post('/api/proxy/config/apply', data)
}
