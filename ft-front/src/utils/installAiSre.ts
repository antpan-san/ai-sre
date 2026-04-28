/** 与控制台同源 API 前缀（与 K8s 部署页生成命令逻辑一致）。 */
export function getPublicApiBase(): string {
  return `${window.location.origin}${import.meta.env.VITE_BASE_API || '/ft-api'}`.replace(/\/$/, '')
}

/** 控制机一键安装 ai-sre CLI：curl 拉引导脚本后再拉二进制 */
export function getInstallAiSreShellCurlLine(): string {
  return `curl -fsSL '${getPublicApiBase()}/api/k8s/deploy/install-ai-sre.sh' | sudo bash`
}
