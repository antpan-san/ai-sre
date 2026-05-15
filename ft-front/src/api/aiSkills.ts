import request from '../utils/request'

export interface SkillSummary {
  name: string
  display_name: string
  topics: string[]
  source: string
  version: string
  path?: string
}

export interface SkillErrorCode {
  code: string
  summary: string
  root_cause: string
  typical_evidence?: string[]
  recovery_one_liner?: string
  platform_followup?: string
  related_codes?: string[]
}

export interface SkillPack {
  name: string
  display_name: string
  topics: string[]
  match_keywords?: string[]
  input?: string[]
  analysis_steps?: string[]
  output_format?: string[]
  extra_guidance?: string
  prompt_template?: string
  error_codes?: SkillErrorCode[]
}

export interface RegisteredSkill {
  pack: SkillPack
  source: string
  version: string
  path?: string
}

export type AdminSkillsListResponse = {
  skills: SkillSummary[]
  data_dir?: string
}

/** 超级管理员：与 CLI 读取的注册表一致（内置 + generated） */
export const getAdminAiSkills = (): Promise<AdminSkillsListResponse> => {
  return request.get('/api/admin/ai/skills')
}

export const getAdminAiSkillDetail = (name: string): Promise<{ skill: RegisteredSkill }> => {
  return request.get(`/api/admin/ai/skills/${encodeURIComponent(name)}`)
}
