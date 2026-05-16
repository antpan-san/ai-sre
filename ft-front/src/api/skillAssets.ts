import request from '../utils/request'

export interface SkillAssetListItem {
  id: string
  topic: string
  skill_key?: string
  problem_key?: string
  capability_key?: string
  category_path?: string
  name: string
  display_name: string
  status: string
  source: string
  created_by: string
  created_at: string
  approved_by?: string
  approved_at?: string
  current_version_id?: string
  version_label?: string
  observation_summary?: string
  risk_level?: string
  review_notes?: string
  rejected_reason?: string
  published_pack_path?: string
  published_at?: string
  deprecated_reason?: string
}

export interface SkillTreeNode {
  path: string
  parent_path?: string
  node_type: string
  title: string
  description?: string
  topic?: string
  skill_key?: string
  problem_key?: string
  capability_key?: string
  pack_key?: string
  feature_key?: string
  execution_mode?: string
  cli_visible: boolean
  status?: string
  sort_order: number
  asset_stats?: {
    total: number
    draft: number
    review: number
    approved: number
    deprecated: number
  }
}

export type SkillTreeResponse = {
  tree_rev: string
  tree_source?: string
  nodes: SkillTreeNode[]
}

export interface SkillAssetDetail extends SkillAssetListItem {
  version_status: string
  content: Record<string, unknown>
  checksum: string
  version_notes: string
}

export type SkillAssetsListResponse = {
  items: SkillAssetListItem[]
  total: number
  page: number
  page_size: number
}

export interface SkillApproveDiff {
  topic: string
  generated_pack_name: string
  registry_pack_name?: string
  registry_source?: string
  merge_preview: boolean
  generated_summary: Record<string, unknown>
  registry_summary?: Record<string, unknown>
  merged_summary?: Record<string, unknown>
  fields_changed?: string[]
}

export interface SkillAssetReviewRow {
  id: string
  skill_asset_id: string
  action: string
  actor_name?: string
  notes?: string
  publish_mode?: string
  merged_with_builtin: boolean
  published_pack_path?: string
  diff_summary?: Record<string, unknown>
  created_at: string
}

export interface SkillUsageRow {
  label: string
  count: number
  status?: string
  extra?: string
}

export interface SkillUsageSummary {
  diagnostic_plans: SkillUsageRow[]
  ai_executions: SkillUsageRow[]
  skill_assets: SkillUsageRow[]
  reviews: SkillUsageRow[]
}

export const listAdminSkillAssets = (params?: {
  status?: string
  topic?: string
  skill_key?: string
  problem_key?: string
  capability_key?: string
  category_path?: string
  created_by?: string
  page?: number
  page_size?: number
}): Promise<SkillAssetsListResponse> => {
  return request.get('/api/admin/skill-assets', { params })
}

export const getAdminSkillAsset = (id: string): Promise<{ asset: SkillAssetDetail }> => {
  return request.get(`/api/admin/skill-assets/${encodeURIComponent(id)}`)
}

export const getAdminSkillTree = (): Promise<SkillTreeResponse> => {
  return request.get('/api/admin/skill-tree')
}

export const approveAdminSkillAsset = (
  id: string,
  body?: { notes?: string; merge_with_registry?: boolean }
): Promise<{ asset_id: string; status: string; path: string; merged?: boolean }> => {
  return request.post(`/api/admin/skill-assets/${encodeURIComponent(id)}/approve`, body ?? {})
}

export const rejectAdminSkillAsset = (
  id: string,
  body?: { reason?: string }
): Promise<{ asset_id: string; status: string }> => {
  return request.post(`/api/admin/skill-assets/${encodeURIComponent(id)}/reject`, body ?? {})
}

export const getAdminSkillAssetApproveDiff = (
  id: string,
  mergeWithRegistry = true
): Promise<{ diff: SkillApproveDiff }> => {
  return request.get(`/api/admin/skill-assets/${encodeURIComponent(id)}/diff`, {
    params: { merge_with_registry: mergeWithRegistry }
  })
}

export const listAdminSkillAssetReviews = (
  id: string,
  limit = 50
): Promise<{ items: SkillAssetReviewRow[] }> => {
  return request.get(`/api/admin/skill-assets/${encodeURIComponent(id)}/reviews`, { params: { limit } })
}

export const deprecateAdminSkillAsset = (
  id: string,
  body?: { reason?: string }
): Promise<{ asset_id: string; status: string }> => {
  return request.post(`/api/admin/skill-assets/${encodeURIComponent(id)}/deprecate`, body ?? {})
}

export const getAdminSkillUsageSummary = (days = 30): Promise<{ since: string; days: number; stats: SkillUsageSummary }> => {
  return request.get('/api/admin/skill-usage/summary', { params: { days } })
}

export const exportAdminSkillUsageCSV = (days = 30): string => {
  const base = import.meta.env.VITE_API_BASE_URL || ''
  const q = new URLSearchParams({ days: String(days) })
  return `${base}/api/admin/skill-usage/export.csv?${q.toString()}`
}
