import request from '../utils/request'

export interface ExecutionRecordParams {
  page?: number
  pageSize?: number
  source?: string
  status?: string
  rollbackCapability?: string
  target?: string
  keyword?: string
  startDate?: string
  endDate?: string
}

export interface PrepareExecutionRecordRequest {
  source: string
  category: string
  name: string
  command: string
  target_host?: string
  target_ips?: string[]
  resource_type?: string
  resource_id?: string
  resource_name?: string
  rollback_capability?: string
  rollback_plan?: Record<string, any>
  rollback_advice?: string
  metadata?: Record<string, any>
}

export const getExecutionRecords = (params: ExecutionRecordParams): Promise<any> => {
  return request.get('/api/execution-records', { params })
}

export const getExecutionRecordDetail = (id: string): Promise<any> => {
  return request.get(`/api/execution-records/${id}`)
}

export const prepareExecutionRecord = (data: PrepareExecutionRecordRequest): Promise<{
  id: string
  correlationId: string
  reportToken: string
}> => {
  return request.post('/api/execution-records/prepare', data)
}

export const previewExecutionRollback = (id: string): Promise<any> => {
  return request.post(`/api/execution-records/${id}/rollback-preview`)
}

export const rollbackExecutionRecord = (id: string, confirmed: boolean): Promise<any> => {
  return request.post(`/api/execution-records/${id}/rollback`, { confirmed })
}
