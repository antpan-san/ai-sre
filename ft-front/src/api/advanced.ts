import request from '../utils/request'
import type { Backup, BackupParams, BackupListResponse, PerformanceData, PerformanceParams, PerformanceDataResponse, PerformanceReport } from '../types'

// 获取备份列表
export const getBackups = (params: BackupParams): Promise<BackupListResponse> => {
  return request.get('/api/advanced/backups', { params })
}

// 获取备份详情
export const getBackupDetail = (id: number): Promise<Backup> => {
  return request.get(`/api/advanced/backups/${id}`)
}

// 创建备份
export const createBackup = (data: { name: string; description?: string }): Promise<Backup> => {
  return request.post('/api/advanced/backups', data)
}

// 恢复备份
export const restoreBackup = (id: number): Promise<void> => {
  return request.post(`/api/advanced/backups/${id}/restore`)
}

// 删除备份
export const deleteBackup = (id: number): Promise<void> => {
  return request.delete(`/api/advanced/backups/${id}`)
}

// 批量删除备份
export const batchDeleteBackups = (ids: number[]): Promise<void> => {
  return request.delete('/api/advanced/backups/batch', { data: { ids } })
}

// 获取备份进度
export const getBackupProgress = (id: number): Promise<{ progress: number; status: string }> => {
  return request.get(`/api/advanced/backups/${id}/progress`)
}

// 获取性能数据
export const getPerformanceData = (params: PerformanceParams): Promise<PerformanceDataResponse> => {
  return request.get('/api/advanced/performance', { params })
}

// 获取性能报告
export const getPerformanceReport = (id: number): Promise<PerformanceReport> => {
  return request.get(`/api/advanced/performance/report/${id}`)
}

// 生成性能报告
export const generatePerformanceReport = (params: PerformanceParams): Promise<PerformanceReport> => {
  return request.post('/api/advanced/performance/report/generate', params)
}

// 导出性能报告
export const exportPerformanceReport = (id: number): Promise<Blob> => {
  return request.get(`/api/advanced/performance/report/${id}/export`, { responseType: 'blob' })
}

// 获取系统性能指标
export const getSystemPerformanceMetrics = (): Promise<PerformanceData> => {
  return request.get('/api/advanced/performance/metrics')
}
