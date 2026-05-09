/**
 * 非 HTTPS（内网控制台常见）或未授予剪贴板权限时，Clipboard API 会失败；
 * 在用户点击事件中用 textarea + execCommand 通常仍可用。
 */
export async function copyTextToClipboard(text: string): Promise<void> {
  const fallback = (): void => {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', 'readonly')
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    ta.style.left = '-9999px'
    ta.style.top = '0'
    document.body.appendChild(ta)
    ta.focus({ preventScroll: true })
    ta.select()
    ta.setSelectionRange(0, text.length)
    try {
      if (!document.execCommand('copy')) {
        throw new Error('execCommand copy failed')
      }
    } finally {
      document.body.removeChild(ta)
    }
  }

  if (typeof navigator !== 'undefined' && navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return
    } catch {
      // SecurityError / 权限拒绝等，走降级
    }
  }
  fallback()
}
