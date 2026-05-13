import request from '../utils/request'

export interface OpsfleetErrorCode {
  code: string
  summary: string
  root_cause: string
  typical_evidence?: string[]
  recovery_one_liner?: string
  platform_followup?: string
  related_codes?: string[]
}

export interface ErrorCodesListData {
  codes: OpsfleetErrorCode[]
  count: number
}

export const getErrorCodesCatalog = (): Promise<ErrorCodesListData> => {
  return request.get('/api/ai/error-codes')
}
