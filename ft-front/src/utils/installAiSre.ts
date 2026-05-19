/** 与控制台同源 API 前缀（与 K8s 部署页生成命令逻辑一致）。 */
export function getPublicApiBase(): string {
  return `${window.location.origin}${import.meta.env.VITE_BASE_API || '/ft-api'}`.replace(/\/$/, '')
}

/** 当前登录 JWT（无则 null） */
export function getStoredAuthToken(): string | null {
  try {
    const t = localStorage.getItem('token')
    return t?.trim() ? t.trim() : null
  } catch {
    return null
  }
}

export const INSTALL_AI_SRE_PLACEHOLDER = '# 点击「生成复制」'
