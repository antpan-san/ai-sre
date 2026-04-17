import request from '../utils/request'
import type {
  GetMonitoringConfigListResponse,
  GetMonitoringConfigResponse,
  CreateMonitoringConfigRequest,
  CreateMonitoringConfigResponse,
  DeleteMonitoringConfigResponse,
  GetAlertRulesResponse,
  SaveAlertRuleResponse,
  DeleteAlertRuleResponse
} from '../types/monitoring'

/**
 * 获取监控配置列表
 * @returns 监控配置列表
 */
export const getMonitoringConfigList = (): Promise<GetMonitoringConfigListResponse> => {
  return request.get('/api/monitoring/configs')
}

/**
 * 获取单个监控配置
 * @param id 配置ID
 * @returns 监控配置
 */
export const getMonitoringConfig = (id: string): Promise<GetMonitoringConfigResponse> => {
  return request.get(`/api/monitoring/configs/${id}`)
}

/**
 * 创建监控配置
 * @param data 配置数据
 * @returns 创建的配置
 */
export const createMonitoringConfig = (data: CreateMonitoringConfigRequest): Promise<CreateMonitoringConfigResponse> => {
  return request.post('/api/monitoring/configs', data)
}

/**
 * 更新监控配置
 * @param id 配置ID
 * @param data 配置数据
 * @returns 更新后的配置
 */
export const updateMonitoringConfig = (id: string, data: CreateMonitoringConfigRequest): Promise<CreateMonitoringConfigResponse> => {
  return request.put(`/api/monitoring/configs/${id}`, data)
}

/**
 * 删除监控配置
 * @param id 配置ID
 * @returns 删除结果
 */
export const deleteMonitoringConfig = (id: string): Promise<DeleteMonitoringConfigResponse> => {
  return request.delete(`/api/monitoring/configs/${id}`)
}

/**
 * 获取告警规则列表
 * @returns 告警规则列表
 */
export const getAlertRules = (): Promise<GetAlertRulesResponse> => {
  return request.get('/api/monitoring/alert-rules')
}

/**
 * 创建告警规则
 * @param data 规则数据
 * @returns 创建的规则
 */
export const createAlertRule = (data: any): Promise<SaveAlertRuleResponse> => {
  return request.post('/api/monitoring/alert-rules', data)
}

/**
 * 更新告警规则
 * @param id 规则ID
 * @param data 规则数据
 * @returns 更新后的规则
 */
export const updateAlertRule = (id: string, data: any): Promise<SaveAlertRuleResponse> => {
  return request.put(`/api/monitoring/alert-rules/${id}`, data)
}

/**
 * 删除告警规则
 * @param id 规则ID
 * @returns 删除结果
 */
export const deleteAlertRule = (id: string): Promise<DeleteAlertRuleResponse> => {
  return request.delete(`/api/monitoring/alert-rules/${id}`)
}
