import request from '../utils/request'
import type { OperationLog, OperationLogParams, OperationLogListResponse, Permission, PermissionListParams, PermissionListResponse } from '../types'

// 获取操作日志列表
export const getOperationLogs = (params: OperationLogParams): Promise<OperationLogListResponse> => {
  return request.get('/api/security-audit/operation-logs', { params })
}

// 获取操作日志详情
export const getOperationLogDetail = (id: number): Promise<OperationLog> => {
  return request.get(`/api/security-audit/operation-logs/${id}`)
}

// 导出操作日志
export const exportOperationLogs = (params: OperationLogParams): Promise<Blob> => {
  return request.get('/api/security-audit/operation-logs/export', { params, responseType: 'blob' })
}

// 获取权限列表
export const getPermissions = (params: PermissionListParams): Promise<PermissionListResponse> => {
  return request.get('/api/security-audit/permissions', { params })
}

// 获取权限详情
export const getPermissionDetail = (id: number): Promise<Permission> => {
  return request.get(`/api/security-audit/permissions/${id}`)
}

// 添加权限
export const addPermission = (data: Permission): Promise<Permission> => {
  return request.post('/api/security-audit/permissions', data)
}

// 更新权限
export const updatePermission = (id: number, data: Permission): Promise<Permission> => {
  return request.put(`/api/security-audit/permissions/${id}`, data)
}

// 删除权限
export const deletePermission = (id: number): Promise<void> => {
  return request.delete(`/api/security-audit/permissions/${id}`)
}

// 批量删除权限
export const batchDeletePermissions = (ids: number[]): Promise<void> => {
  return request.delete('/api/security-audit/permissions/batch', { data: { ids } })
}

// 获取角色权限列表
export const getRolePermissions = (role: string): Promise<PermissionListResponse> => {
  return request.get(`/api/security-audit/roles/${role}/permissions`)
}

// 分配角色权限
export const assignRolePermissions = (role: string, permissionIds: number[]): Promise<void> => {
  return request.post(`/api/security-audit/roles/${role}/permissions`, { permissionIds })
}
