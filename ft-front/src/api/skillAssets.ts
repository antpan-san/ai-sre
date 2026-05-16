import request from '../utils/request'

export interface SkillAssetListItem {
  id: string
  topic: string
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
  page?: number
  page_size?: number
}): Promise<SkillAssetsListResponse> => {
  return request.get('/api/admin/skill-assets', { params })
}

export const getAdminSkillAsset = (id: string): Promise<{ asset: SkillAssetDetail }> => {
  return request.get(`/api/admin/skill-assets/${encodeURIComponent(id)}`)
}

export const approveAdminSkillAsset = (
  id: string,
  body?: { notes?: string }
): Promise<{ asset_id: string; status: string; path: string }> => {
  return request.post(`/api/admin/skill-assets/${encodeURIComponent(id)}/approve`, body ?? {})
}

export const rejectAdminSkillAsset = (
  id: string,
  body?: { reason?: string }
): Promise<{ asset_id: string; status: string }> => {
  return request.post(`/api/admin/skill-assets/${encodeURIComponent(id)}/reject`, body ?? {})
}
