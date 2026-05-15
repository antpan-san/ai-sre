import request from '../utils/request'

export interface RuntimeWatchSessionRow {
  id: string
  namespace: string
  pod: string
  container?: string
  interval_sec: number
  status: string
  created_at: string
  machine_note?: string
}

export interface CreateRuntimeWatchSessionBody {
  namespace: string
  pod: string
  container?: string
  interval_sec?: number
  machine_note?: string
}

export interface CreateRuntimeWatchSessionResult {
  id: string
  sample_write_token: string
  upload_path: string
  namespace: string
  pod: string
  container?: string
  interval_sec: number
}

export interface RuntimeWatchSampleRow {
  id: string
  observed_at: string
  payload: unknown
}

export interface RuntimeWatchSamplesResponse {
  session: RuntimeWatchSessionRow
  samples: RuntimeWatchSampleRow[]
}

export const listRuntimeWatchSessions = (): Promise<RuntimeWatchSessionRow[]> => {
  return request.get('/api/runtime-watch/sessions')
}

export const createRuntimeWatchSession = (
  body: CreateRuntimeWatchSessionBody
): Promise<CreateRuntimeWatchSessionResult> => {
  return request.post('/api/runtime-watch/sessions', body)
}

export const getRuntimeWatchSamples = (id: string, since?: string): Promise<RuntimeWatchSamplesResponse> => {
  return request.get(`/api/runtime-watch/sessions/${id}/samples`, {
    params: since ? { since } : undefined
  })
}

export const stopRuntimeWatchSession = (id: string): Promise<void> => {
  return request.post(`/api/runtime-watch/sessions/${id}/stop`, {})
}
