import request from '../utils/request'

export interface CLIInstallSession {
  command: string
  expires_at: string
}

export const createCLIInstallSession = (): Promise<CLIInstallSession> => {
  return request.post('/api/me/cli/install-session')
}
