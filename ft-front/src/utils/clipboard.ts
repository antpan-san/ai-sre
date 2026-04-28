/** 非 HTTPS / 部分浏览器下 Clipboard API 不可用，降级到 execCommand */
export async function copyTextToClipboard(text: string): Promise<void> {
  const fallback = (): void => {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.left = '-9999px'
    document.body.appendChild(ta)
    ta.select()
    try {
      if (!document.execCommand('copy')) {
        throw new Error('execCommand copy failed')
      }
    } finally {
      document.body.removeChild(ta)
    }
  }

  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // 权限或策略失败时再试降级
    }
  }
  fallback()
}
