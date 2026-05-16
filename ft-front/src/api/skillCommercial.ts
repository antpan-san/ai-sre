import request from '../utils/request'

export interface CommercialProduct {
  id: string
  product_key: string
  title: string
  description?: string
  product_type: string
  status: string
  price_hint?: string
  sort_order: number
}

export interface ProductNodeBinding {
  id: string
  product_key: string
  node_path: string
  skill_key?: string
  capability_key?: string
  pack_key?: string
  grant_scope: string
}

export const listCommercialProducts = (): Promise<{ products: CommercialProduct[]; policy_rev: string }> => {
  return request.get('/api/admin/skill-commercial/products')
}

export const listCommercialBindings = (productKey?: string): Promise<{ bindings: ProductNodeBinding[] }> => {
  return request.get('/api/admin/skill-commercial/bindings', { params: productKey ? { product_key: productKey } : {} })
}

export const createCommercialBinding = (body: {
  product_key: string
  node_path: string
  grant_scope?: string
  pack_key?: string
}): Promise<{ created: boolean }> => {
  return request.post('/api/admin/skill-commercial/bindings', body)
}

export const deleteCommercialBinding = (id: string): Promise<{ deleted: boolean }> => {
  return request.delete(`/api/admin/skill-commercial/bindings/${encodeURIComponent(id)}`)
}

export const nodeCommercialProducts = (nodePath: string): Promise<{ node_path: string; product_keys: string[] }> => {
  return request.get('/api/admin/skill-commercial/node-products', { params: { node_path: nodePath } })
}
