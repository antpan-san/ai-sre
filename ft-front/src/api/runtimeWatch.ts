import request from '../utils/request'

export interface RuntimeDiagnosisRow {
  id: string
  namespace: string
  pod: string
  container?: string
  interval_sec: number
  status: string
  created_at: string
  machine_note?: string
  target_display?: string
  resource_kind?: string
  resource_name?: string
  work_pod?: string
  diagnosis_level?: string
  root_cause?: string
  evidence?: string
  diagnosis_source?: string
  sample_count?: number
  last_diagnosed_at?: string
  has_diagnosis: boolean
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
  session: RuntimeDiagnosisRow
  samples: RuntimeWatchSampleRow[]
}

export const listRuntimeDiagnoses = (): Promise<RuntimeDiagnosisRow[]> => {
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

export const deleteRuntimeWatchSession = (id: string): Promise<void> => {
  return request.delete(`/api/runtime-watch/sessions/${encodeURIComponent(id)}`)
}

/** @deprecated use listRuntimeDiagnoses */
export const listRuntimeWatchSessions = listRuntimeDiagnoses

export type RuntimeWatchSessionRow = RuntimeDiagnosisRow
