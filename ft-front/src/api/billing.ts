import request from '../utils/request'

export interface FeatureBillingRow {
  feature_key: string
  billing_enabled: boolean
  description: string
  updated_at?: string
}

export type FeatureBillingUpdateItem = {
  feature_key: string
  billing_enabled?: boolean
  description?: string
}

export const getAdminFeatureBilling = (): Promise<FeatureBillingRow[]> => {
  return request.get('/api/admin/billing/features')
}

export const putAdminFeatureBilling = (items: FeatureBillingUpdateItem[]): Promise<FeatureBillingRow[]> => {
  return request.put('/api/admin/billing/features', { items })
}

export interface BillingMe {
  billing_exempt: boolean
  subscription: Record<string, unknown> | null
  entitlements: unknown[]
  feature_flags: Record<string, boolean>
}

export const getBillingMe = (): Promise<BillingMe> => {
  return request.get('/api/billing/me')
}

export const createCheckoutSession = (): Promise<{ url: string }> => {
  return request.post('/api/billing/checkout-session', {})
}

export const grantUserEntitlement = (
  userId: string,
  body: { feature_key: string; valid_until?: string | null }
): Promise<unknown> => {
  return request.post(`/api/admin/users/${userId}/entitlement`, body)
}
