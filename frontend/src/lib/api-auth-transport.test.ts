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
    const headers = options.headers as Record<string, string>

    expect(headers.Authorization).toBeUndefined()
    expect(headers["Accept-Language"] ?? headers["accept-language"]).toBeTruthy()
  })
})
