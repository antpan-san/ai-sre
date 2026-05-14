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

/**
 * 控制机一键安装 ai-sre CLI：须已登录控制台；curl 带 Bearer 拉取个性化脚本后写入本机令牌，后续服务端 AI 按账号计费与限额。
 * 令牌与会话 JWT 一致，过期后请在控制台重新复制本命令执行。
 */
export function getInstallAiSreShellCurlLine(): string {
  const base = getPublicApiBase()
  const token = getStoredAuthToken()
  if (!token) {
    return `# 请先在控制台登录后再复制「安装 ai-sre」命令（需携带当前账号访问令牌以绑定订阅与 AI 限额）。`
  }
  // JWT 通常不含单引号；若含单引号则无法安全嵌入单引号包裹的 shell 片段，退回双引号转义
  const url = `${base}/api/me/cli/install-ai-sre.sh`
  if (!token.includes("'")) {
    return `curl -fsSL -H 'Authorization: Bearer ${token}' '${url}' | sudo bash`
  }
  const esc = token.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/\$/g, '\\$').replace(/`/g, '\\`')
  return `curl -fsSL -H "Authorization: Bearer ${esc}" '${url.replace(/'/g, "'\\''")}' | sudo bash`
}
