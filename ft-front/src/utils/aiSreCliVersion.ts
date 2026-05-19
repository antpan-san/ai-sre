import { getPublicApiBase } from './installAiSre'

export interface AiSreCLIVersionInfo {
  name?: string
  version: string
  ok?: boolean
  message?: string
  binary_path?: string
  env_version?: string
}

/** 公开接口，响应为裸 JSON（非 { code, data }），须用 fetch。 */
export async function fetchAiSreCLIVersion(): Promise<AiSreCLIVersionInfo | null> {
  const base = getPublicApiBase()
  const url = `${base}/api/k8s/deploy/cli/ai-sre/version`
  try {
    const resp = await fetch(url, {
      method: 'GET',
      headers: { Accept: 'application/json' },
      cache: 'no-store'
    })
    if (!resp.ok) {
      return null
    }
    const data = (await resp.json()) as AiSreCLIVersionInfo
    const version = String(data?.version ?? '').trim()
    if (!version || version === 'unknown') {
      return null
    }
    return { ...data, version }
  } catch {
    return null
  }
}
