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

export const listAdminSkillAssets = (params?: {
  status?: string
  topic?: string
  skill_key?: string
  problem_key?: string
  capability_key?: string
  category_path?: string
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
