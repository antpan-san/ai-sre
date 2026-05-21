import request from '../utils/request'

export interface AutoIterationSettings {
  enabled: boolean
  max_concurrent: number
  high_risk_requires_approval: boolean
  auto_dispatch_enabled: boolean
  low_risk_auto_deploy_enabled: boolean
  github_sync_enabled: boolean
  dingtalk_notify_enabled: boolean
  github_repo?: string
  has_dingtalk_webhook: boolean
  notes?: string
  updated_at?: string
  updated_by?: string
}

export interface AutoIteration {
  id: string
  title: string
  description?: string
  status: string
  source: string
  risk_level: string
  requires_super_admin_approval: boolean
  topic?: string
  command?: string
  summary?: string
  last_error?: string
  created_by?: string
  approved_by?: string
  approved_at?: string
  assigned_agent_id?: string
  created_at: string
}

export interface AutoIterationEvent {
  id: string
  event_type: string
  actor_type: string
  actor_name?: string
  message: string
  created_at: string
}

export const getAutoIterationSamples = (id: string) =>
  request.get<{
    topic?: string
    execution_id?: string
    root_cause_digest?: string
    similar_recent_count?: number
    sample_classification?: string
    trigger_sample?: Record<string, unknown>
    similar_samples?: Record<string, unknown>[]
  }>(`/api/admin/auto-iterations/${encodeURIComponent(id)}/samples`)

export const listAutoIterations = (params?: {
  status?: string
  topic?: string
  source?: string
  keyword?: string
  page?: number
  page_size?: number
}): Promise<{ list: AutoIteration[]; total: number }> => {
  return request.get('/api/admin/auto-iterations', { params }).then((r: { list: AutoIteration[]; total: number }) => r)
}

export const getAutoIteration = (
  id: string
): Promise<{ iteration: AutoIteration; events: AutoIterationEvent[] }> => {
  return request.get(`/api/admin/auto-iterations/${encodeURIComponent(id)}`)
}

export const getAutoIterationSettings = (): Promise<{ settings: AutoIterationSettings }> => {
  return request.get('/api/admin/auto-iterations/settings')
}

export const updateAutoIterationSettings = (body: Partial<AutoIterationSettings>): Promise<{ settings: AutoIterationSettings }> => {
  return request.put('/api/admin/auto-iterations/settings', body)
}

export const createManualAutoIteration = (body: {
  title?: string
  description?: string
  command?: string
  topic?: string
  auto_start?: boolean
}): Promise<{ iteration: AutoIteration }> => {
  return request.post('/api/admin/auto-iterations/manual', body)
}

const postAction = (id: string, action: string, body?: Record<string, string | boolean>) =>
  request.post(`/api/admin/auto-iterations/${encodeURIComponent(id)}/${action}`, body ?? {})

export const startAutoIteration = (id: string) => postAction(id, 'start')
export const pauseAutoIteration = (id: string) => postAction(id, 'pause')
export const resumeAutoIteration = (id: string) => postAction(id, 'resume')
export const cancelAutoIteration = (id: string) => postAction(id, 'cancel')
export const approveAutoIteration = (id: string, notes?: string, force?: boolean) =>
  postAction(id, 'approve', { notes: notes ?? '', force: force === true })
export const rejectAutoIteration = (id: string, reason?: string) => postAction(id, 'reject', { reason: reason ?? '' })
export const rollbackAutoIteration = (id: string, reason?: string) => postAction(id, 'rollback', { reason: reason ?? '' })
export const runAutoIterationTests = (id: string) => postAction(id, 'run-tests')
export const syncAutoIterationGitHub = (id: string) => postAction(id, 'sync-github')
export const resendAutoIterationNotification = (id: string) => postAction(id, 'resend-notification')
