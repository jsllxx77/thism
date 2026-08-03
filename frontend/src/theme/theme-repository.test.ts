import { describe, expect, it, vi } from "vitest"

import { loadThemePackageFromGitHub } from "./theme-repository"

function response(body: unknown, ok = true, status = ok ? 200 : 404) {
  return {
    ok,
    status,
    json: async () => body,
    arrayBuffer: async () => body instanceof Uint8Array ? body.slice().buffer : new TextEncoder().encode(String(body)).buffer,
  } as Response
}

describe("GitHub theme repositories", () => {
  it("loads a zip archive from the latest release asset", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url === "https://api.github.com/repos/acme/thism-themes/releases/latest") {
        return response({
          assets: [
            { name: "preview.txt", url: "https://api.github.com/assets/preview" },
            { name: "repo-command.thism-theme.zip", url: "https://api.github.com/assets/1", browser_download_url: "https://github.com/acme/theme.zip" },
          ],
        })
      }
      if (url === "https://api.github.com/assets/1") return response(new Uint8Array([1, 2, 3]))
      throw new Error(`unexpected ${url}`)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/thism-themes", fetchMock)).resolves.toEqual({
      filename: "repo-command.thism-theme.zip",
      data: "AQID",
    })
    expect(fetchMock).toHaveBeenCalledWith("https://api.github.com/assets/1", expect.objectContaining({ headers: expect.objectContaining({ Accept: "application/octet-stream" }) }))
  })

  it("prefers a theme archive when a release has multiple zip assets", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url.endsWith("/releases/latest")) return response({ assets: [
        { name: "source.zip", url: "https://api.github.com/assets/source" },
        { name: "repo-command.thism-theme.zip", url: "https://api.github.com/assets/theme" },
      ] })
      if (url.endsWith("/assets/theme")) return response(new Uint8Array([4]))
      throw new Error(`unexpected ${url}`)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/thism-themes", fetchMock)).resolves.toEqual({
      filename: "repo-command.thism-theme.zip",
      data: "BA==",
    })
  })

  it("falls back to the browser download URL when the GitHub asset API fails", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url.endsWith("/releases/latest")) return response({ assets: [{ name: "theme.zip", url: "https://api.github.com/assets/theme", browser_download_url: "https://github.com/acme/theme.zip" }] })
      if (url.endsWith("/assets/theme")) throw new TypeError("Failed to fetch")
      if (url === "https://github.com/acme/theme.zip") return response(new Uint8Array([5, 6]))
      throw new Error(`unexpected ${url}`)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/thism-themes", fetchMock)).resolves.toEqual({
      filename: "theme.zip",
      data: "BQY=",
    })
  })

  it("fails when the latest release has no zip theme archive", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url.endsWith("/releases/latest")) return response({ assets: [{ name: "theme.json", url: "https://api.github.com/assets/theme" }] })
      throw new Error(`unexpected ${url}`)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/html-theme", fetchMock)).rejects.toThrow("No theme zip archive")
  })
})
