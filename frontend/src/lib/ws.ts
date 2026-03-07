export type WSMessage = {
  type: string
  payload?: unknown
}

type Handler = (msg: WSMessage) => void

class DashboardWS {
  private ws: WebSocket | null = null
  private handlers: Handler[] = []
  private reconnectDelay = 1000
  connect() {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:"
    const url = `${protocol}//${window.location.host}/ws/dashboard`
    this.ws = new WebSocket(url)
    this.ws.onmessage = (e) => {
      try {
        const msg: WSMessage = JSON.parse(e.data)
        this.handlers.forEach((h) => h(msg))
      } catch {
        // Ignore malformed message payloads and keep the socket alive.
        return
      }
    }
    this.ws.onclose = () => {
      setTimeout(() => {
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, 60000)
        this.connect()
      }, this.reconnectDelay)
    }
    this.ws.onopen = () => {
      this.reconnectDelay = 1000
    }
  }

  on(handler: Handler) {
    this.handlers.push(handler)
  }

  off(handler: Handler) {
    this.handlers = this.handlers.filter((h) => h !== handler)
  }
}

let instance: DashboardWS | null = null

export function getDashboardWS(): DashboardWS {
  if (!instance) {
    instance = new DashboardWS()
    instance.connect()
  }
  return instance
}
