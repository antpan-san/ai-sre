/**
 * SSE over fetch with Authorization header (EventSource cannot set headers).
 */
export type SSEHandlers = {
  onEvent?: (eventName: string, data: string) => void
  onError?: (err: unknown) => void
  /** Stream ended (server closed or connection dropped). */
  onClose?: () => void
}

function apiBase(): string {
  const base = import.meta.env.VITE_BASE_API || '/ft-api'
  return base.replace(/\/$/, '')
}

export function connectSSE(path: string, handlers: SSEHandlers): AbortController {
  const ac = new AbortController()
  const token = localStorage.getItem('token')
  const url = `${apiBase()}${path.startsWith('/') ? path : `/${path}`}`

  void (async () => {
    try {
      const res = await fetch(url, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
        signal: ac.signal
      })
      if (!res.ok || !res.body) {
        handlers.onError?.(new Error(`SSE HTTP ${res.status}`))
        return
      }
      const reader = res.body.getReader()
      const dec = new TextDecoder()
      let buf = ''
      let eventName = 'message'
      let dataLines: string[] = []

      const flush = () => {
        if (dataLines.length) {
          handlers.onEvent?.(eventName, dataLines.join('\n'))
        }
        eventName = 'message'
        dataLines = []
      }

      for (;;) {
        const { done, value } = await reader.read()
        if (done) break
        buf += dec.decode(value, { stream: true })
        const lines = buf.split('\n')
        buf = lines.pop() ?? ''
        for (const line of lines) {
          if (line.startsWith(':')) continue
          if (line === '') {
            flush()
            continue
          }
          if (line.startsWith('event:')) {
            eventName = line.slice(6).trim()
          } else if (line.startsWith('data:')) {
            dataLines.push(line.slice(5).trimStart())
          }
        }
      }
      flush()
      handlers.onClose?.()
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        handlers.onError?.(err)
        handlers.onClose?.()
      }
    }
  })()

  return ac
}
