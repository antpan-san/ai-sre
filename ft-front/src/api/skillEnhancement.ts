import request from '../utils/request'

export interface SkillEnhancementReview {
  time?: string
  request_id?: string
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
