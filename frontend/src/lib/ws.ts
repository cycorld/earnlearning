import { isTokenExpired, removeToken, setToken } from './auth'

type EventCallback = (data: unknown) => void

class WebSocketClient {
  private ws: WebSocket | null = null
  private listeners = new Map<string, Set<EventCallback>>()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectDelay = 1000
  private maxReconnectDelay = 30000
  private token: string | null = null
  private intentionalClose = false

  connect(token: string): void {
    this.token = token
    this.intentionalClose = false
    this.reconnectDelay = 1000
    this.createConnection()
  }

  disconnect(): void {
    this.intentionalClose = true
    this.token = null
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  on(event: string, callback: EventCallback): () => void {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set())
    }
    this.listeners.get(event)!.add(callback)

    return () => {
      const callbacks = this.listeners.get(event)
      if (callbacks) {
        callbacks.delete(callback)
        if (callbacks.size === 0) {
          this.listeners.delete(event)
        }
      }
    }
  }

  private createConnection(): void {
    if (!this.token) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/ws?token=${encodeURIComponent(this.token)}`

    this.ws = new WebSocket(wsUrl)

    this.ws.onopen = () => {
      this.reconnectDelay = 1000
    }

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as {
          event: string
          data: unknown
        }
        const callbacks = this.listeners.get(message.event)
        if (callbacks) {
          callbacks.forEach((cb) => cb(message.data))
        }
      } catch {
        // ignore malformed messages
      }
    }

    this.ws.onclose = () => {
      this.ws = null
      if (!this.intentionalClose && this.token) {
        if (isTokenExpired(this.token)) {
          // Token expired — try refresh via API, then reconnect
          fetch('/api/auth/refresh', {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${this.token}` },
          })
            .then(res => res.ok ? res.json() : Promise.reject())
            .then(data => {
              const newToken = data.data.token
              this.token = newToken
              setToken(newToken)
              this.createConnection()
            })
            .catch(() => {
              removeToken()
              window.location.href = '/login'
            })
          return
        }
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer) return
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.createConnection()
      this.reconnectDelay = Math.min(
        this.reconnectDelay * 2,
        this.maxReconnectDelay,
      )
    }, this.reconnectDelay)
  }
}

export const wsClient = new WebSocketClient()
