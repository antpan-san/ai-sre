import request from '../utils/request'
export { fetchAiSreCLIVersion, type AiSreCLIVersionInfo } from '../utils/aiSreCliVersion'

export interface CLIInstallSession {
  command: string
  expires_at: string
}

export const createCLIInstallSession = (): Promise<CLIInstallSession> => {
  return request.post('/api/me/cli/install-session')
}
