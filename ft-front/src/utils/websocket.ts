/**
 * WebSocket service for real-time machine status updates.
 * Auto-reconnects with exponential backoff.
 * Dispatches events by message `type` to registered handlers.
 */

type MessageHandler = (data: any) => void

class WebSocketService {
  private ws: WebSocket | null = null
  private url = ''
  private handlers: Map<string, MessageHandler[]> = new Map()
  private reconnectAttempts = 0
  private maxReconnectAttempts = 20
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private isManualClose = false

  /**
   * Connect to the WebSocket server.
   * @param userId - The user ID for the WebSocket connection path.
   */
  connect(userId: string): void {
    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return // already connected or connecting
    }

    this.isManualClose = false

    // Build WS URL: dev → backend port (VITE_API_PORT); prod → same host/port as page (e.g. nginx :80)
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.hostname
    if (import.meta.env.DEV) {
      const port = import.meta.env.VITE_API_PORT || '8080'
      this.url = `${protocol}//${host}:${port}/ws/${userId}`
    } else {
      const p = window.location.port
      const portSuffix =
        p && p !== '80' && p !== '443' ? `:${p}` : ''
      this.url = `${protocol}//${host}${portSuffix}/ws/${userId}`
    }

    this.doConnect()
  }

  private doConnect(): void {
    try {
      this.ws = new WebSocket(this.url)

      this.ws.onopen = () => {
        console.log('[WS] Connected:', this.url)
        this.reconnectAttempts = 0
      }

      this.ws.onmessage = (event: MessageEvent) => {
        try {
          const msg = JSON.parse(event.data)
          const type = msg.type as string
          if (type && this.handlers.has(type)) {
            this.handlers.get(type)!.forEach(handler => handler(msg))
          }
          // Also dispatch to wildcard handlers
          if (this.handlers.has('*')) {
            this.handlers.get('*')!.forEach(handler => handler(msg))
          }
        } catch (err) {
          console.error('[WS] Failed to parse message:', err)
        }
      }

      this.ws.onclose = (event) => {
        console.log('[WS] Disconnected:', event.code, event.reason)
        if (!this.isManualClose) {
          this.scheduleReconnect()
        }
      }

      this.ws.onerror = (error) => {
        console.error('[WS] Error:', error)
      }
    } catch (err) {
      console.error('[WS] Connection failed:', err)
      this.scheduleReconnect()
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('[WS] Max reconnect attempts reached, giving up')
      return
    }

    // Exponential backoff: 1s, 2s, 4s, 8s, ... max 30s
    const delay = Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000)
    this.reconnectAttempts++

    console.log(`[WS] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`)

    this.reconnectTimer = setTimeout(() => {
      this.doConnect()
    }, delay)
  }

  /**
   * Register a handler for a specific message type.
   * Use '*' to handle all message types.
   */
  on(type: string, handler: MessageHandler): void {
    if (!this.handlers.has(type)) {
      this.handlers.set(type, [])
    }
    this.handlers.get(type)!.push(handler)
  }

  /**
   * Remove a handler for a specific message type.
   */
  off(type: string, handler: MessageHandler): void {
    const handlers = this.handlers.get(type)
    if (handlers) {
      const idx = handlers.indexOf(handler)
      if (idx >= 0) handlers.splice(idx, 1)
    }
  }

  /**
   * Disconnect the WebSocket.
   */
  disconnect(): void {
    this.isManualClose = true
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    console.log('[WS] Manually disconnected')
  }

  /**
   * Check if the WebSocket is currently connected.
   */
  get isConnected(): boolean {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN
  }
}

// Singleton instance
export const wsService = new WebSocketService()
