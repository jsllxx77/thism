import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const fetchMock = vi.fn()

describe("api transport", () => {
  beforeEach(() => {
    fetchMock.mockReset()
    vi.stubGlobal("fetch", fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("does not attach an authorization header to browser session requests", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ role: "guest" }),
    })

    const { api } = await import("./api")
    await api.session()

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    const headers = options.headers as Headers

    expect(headers.get("Authorization")).toBeNull()
    expect(headers.get("Accept-Language")).toBeTruthy()
  })

  it("attaches the csrf token from the csrf cookie on state-changing requests", async () => {
    document.cookie = "thism_csrf=csrf-token-123; path=/"
    fetchMock.mockResolvedValue({
      ok: true,
      json: async () => ({ ok: true }),
    })

    const { api } = await import("./api")
    await api.logout()

    const [, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    const headers = options.headers as Headers

    expect(headers.get("X-CSRF-Token")).toBe("csrf-token-123")
    expect(headers.get("Authorization")).toBeNull()
  })

  it("refreshes the csrf cookie and retries one state-changing request when the session is stale", async () => {
    fetchMock
      .mockResolvedValueOnce({
        ok: false,
        status: 403,
        statusText: "Forbidden",
        json: async () => ({ error: "csrf token required" }),
      })
      .mockImplementationOnce(async () => {
        document.cookie = "thism_csrf=fresh-csrf-token; path=/"
        return {
          ok: true,
          json: async () => ({ role: "admin" }),
        }
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({ ok: true }),
      })

    const { api } = await import("./api")
    await api.renameNode("node-1", "edge")

    expect(fetchMock).toHaveBeenCalledTimes(3)
    const [refreshPath] = fetchMock.mock.calls[1] as [string, RequestInit]
    const [retryPath, retryOptions] = fetchMock.mock.calls[2] as [string, RequestInit]
    const retryHeaders = retryOptions.headers as Headers

    expect(refreshPath).toBe("/api/auth/session")
    expect(retryPath).toBe("/api/nodes/node-1")
    expect(retryHeaders.get("X-CSRF-Token")).toBe("fresh-csrf-token")
  })
})
