import request from '../utils/request'
import type { SkillEnhancementReview, SkillEnhancementSummary } from './skillEnhancement'

export interface DiagnoseSample {
  time?: string
  topic: string
  target?: string
  command?: string
  cli_version?: string
  used_ai?: boolean
  local_rule_hit?: boolean
  rule_hit?: boolean
  evidence_completeness?: string
  root_cause_digest?: string
  sample_source?: string
  execution_id?: string
  skill_name?: string
  command_kind?: string
  answer_head?: string
  enhancement_review?: SkillEnhancementReview
}

export interface DiagnoseSampleSummary {
  total_samples: number
  cli_check_count: number
  rule_hit_count: number
  used_ai_count: number
  since_hours: number
  by_topic: Record<string, number>
  top_topics: { topic: string; count: number }[]
}

export function getAdminDiagnoseSampleSummary(hours = 24) {
  return request.get<DiagnoseSampleSummary>('/api/admin/diagnose-samples/summary', { params: { hours } })
}

export function listAdminDiagnoseSamples(params: { topic?: string; limit?: number; hours?: number }) {
  return request.get<{ samples: DiagnoseSample[] }>('/api/admin/diagnose-samples', { params })
}

export interface AutoIterationFeedbackItem {
  id: string
  created_at?: string
  topic: string
  classification?: string
  need_iteration?: boolean
  user_message?: string
  auto_iteration_id?: string
  request_id?: string
  execution_id?: string
  command?: string
  summary?: string
  helpful?: boolean
}

export function listAdminAutoIterationFeedbacks(limit = 50) {
  return request.get<{ feedbacks: AutoIterationFeedbackItem[] }>('/api/admin/auto-iteration-feedbacks', { params: { limit } })
}

export type { SkillEnhancementSummary, SkillEnhancementReview }

export { getAdminSkillEnhancementSummary, listAdminSkillEnhancementReviews, updateSkillEnhancementStatus, lookupExecutionByRequestID, adminRefineSkill } from './skillEnhancement'
