import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

describe("dashboard websocket transport", () => {
  beforeEach(() => {
    vi.resetModules()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("connects without leaking a token query parameter", async () => {
    const urls: string[] = []

    class MockWebSocket {
      onmessage: ((event: MessageEvent) => void) | null = null
      onclose: (() => void) | null = null
      onopen: (() => void) | null = null

      constructor(url: string) {
        urls.push(url)
      }
    }

    vi.stubGlobal("WebSocket", MockWebSocket as unknown as typeof WebSocket)

    const { getDashboardWS } = await import("./ws")
    getDashboardWS()

    expect(urls).toHaveLength(1)
    expect(urls[0]).toContain("/ws/dashboard")
    expect(urls[0]).not.toContain("token=")
  })
})
