import request from '../utils/request'

export interface ClientExecutionListItem {
  id: string
  time: string
  command: string
  normalized_command?: string
  target?: string
  topic?: string
  skill_pack?: string
  pack_key?: string
  status: string
  severity?: string
  summary?: string
  root_cause?: string
  evidence_completeness?: string
  ai_source?: string
  used_ai: boolean
  user?: string
  machine?: string
  client_version?: string
  duration_ms?: number
  legacy_kind?: string
  has_auto_iteration?: boolean
}

export interface ClientExecutionStats {
  total_24h: number
  success_24h: number
  failed_24h: number
  ai_calls_24h: number
  auto_iteration_24h: number
  incomplete_evidence_24h: number
}

export interface ClientExecutionDetail {
  record: Record<string, unknown>
  legacy_kind?: string
  children?: Record<string, unknown>[]
  events?: Record<string, unknown>[]
  timeline?: { phase: string; message: string; time?: string; level?: string }[]
  runtime_report?: {
    session_id: string
    target_display?: string
    root_cause?: string
    sample_count?: number
    diagnosis_source?: string
  }
  enhancement_review?: Record<string, unknown>
  auto_iteration_id?: string
}

export function listAISreExecutions(params: Record<string, string | number | boolean | undefined>) {
  return request.get<{ list: ClientExecutionListItem[]; total: number }>('/api/ai-sre/executions', { params })
}

export function getAISreExecutionStats(hours = 24) {
  return request.get<{ stats: ClientExecutionStats }>('/api/ai-sre/executions/stats', { params: { hours } })
}

export function getAISreExecutionDetail(id: string) {
  return request.get<ClientExecutionDetail>(`/api/ai-sre/executions/${encodeURIComponent(id)}`)
}
