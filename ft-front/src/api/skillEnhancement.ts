import request from '../utils/request'

export interface SkillEnhancementReview {
  time?: string
  request_id?: string
  review_key?: string
  topic: string
  command_kind?: string
  skill_name?: string
  pack_key?: string
  problem_key?: string
  style?: string
  needs_enhancement: boolean
  priority: string
  savings_score: number
  recommendations?: string[]
  suggested_actions?: string[]
  similar_recent_count?: number
  enhancement_status?: string
}

export interface EnhancementTopicScore {
  topic: string
  open_count: number
  savings_score: number
}

export interface SkillEnhancementSummary {
  open_count: number
  high_priority: number
  medium_priority: number
  total_savings_score: number
  by_topic: Record<string, number>
  top_topics: EnhancementTopicScore[]
  recent?: SkillEnhancementReview[]
}

export function getAdminSkillEnhancementSummary(recent = 20) {
  return request.get<SkillEnhancementSummary>('/api/admin/skill-enhancement-reviews/summary', {
    params: { recent }
  })
}

export function listAdminSkillEnhancementReviews(limit = 50, openOnly = true) {
  return request.get<{ reviews: SkillEnhancementReview[] }>('/api/admin/skill-enhancement-reviews', {
    params: { limit, open_only: openOnly }
  })
}

export function updateSkillEnhancementStatus(body: {
  request_id?: string
  review_key?: string
  topic: string
  status: 'refined' | 'dismissed'
  note?: string
}) {
  return request.post<{ updated: boolean; status: string }>('/api/admin/skill-enhancement-reviews/status', body)
}

export function lookupExecutionByRequestID(requestId: string) {
  return request.get<{ execution_id?: string; ai_child_execution_id?: string }>(
    `/api/admin/executions/by-request/${encodeURIComponent(requestId)}`
  )
}

export function adminRefineSkill(body: {
  topic: string
  user_hint?: string
  dry_run?: boolean
  max_samples?: number
  max_feedback?: number
  timeout_sec?: number
}) {
  return request.post<{
    draft_yaml?: string
    dry_run?: boolean
    persisted_path?: string
    samples_used?: number
    feedback_used?: number
  }>('/api/admin/skills/refine', body)
}
