import { describe, expect, it, vi } from "vitest"

import { loadThemePackageFromGitHub } from "./theme-repository"

const themePackage = JSON.stringify({
  type: "thism-theme",
  version: 1,
  id: "repo-command",
  name: "Repo Command",
  accent: "#0ea5e9",
  tokens: {
    light: {
      background: "204 90% 98%",
      foreground: "222 44% 12%",
      card: "0 0% 100%",
      "card-foreground": "222 44% 12%",
      primary: "199 89% 48%",
      "primary-foreground": "0 0% 100%",
      border: "204 32% 84%",
      input: "204 32% 82%",
      ring: "199 89% 48%",
    },
    dark: {
      background: "222 44% 7%",
      foreground: "204 90% 96%",
      card: "222 34% 11%",
      "card-foreground": "204 90% 96%",
      primary: "199 89% 62%",
      "primary-foreground": "222 44% 7%",
      border: "222 24% 24%",
      input: "222 24% 22%",
      ring: "199 89% 62%",
    },
  },
  appearance: {
    radius: "1rem",
    surface: "command",
    background: "grid",
    density: "compact",
  },
})

function response(body: unknown, ok = true, status = ok ? 200 : 404) {
  return {
    ok,
    status,
    json: async () => body,
    text: async () => (typeof body === "string" ? body : JSON.stringify(body)),
  } as Response
}

describe("GitHub theme repositories", () => {
  it("loads the first thism theme package from a repository latest release", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url === "https://api.github.com/repos/acme/thism-themes/releases/latest") {
        return response({
          assets: [
            { name: "preview.png", url: "https://api.github.com/assets/1", browser_download_url: "https://github.com/acme/preview.png" },
            { name: "repo-command.thism-theme.json", url: "https://api.github.com/assets/2", browser_download_url: "https://github.com/acme/theme.json" },
          ],
        })
      }
      if (url === "https://api.github.com/assets/2") {
        return response(themePackage)
      }
      throw new Error(`unexpected ${url}`)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/thism-themes", fetchMock)).resolves.toBe(themePackage)
    expect(fetchMock).toHaveBeenCalledWith("https://api.github.com/assets/2", expect.objectContaining({ headers: expect.objectContaining({ Accept: "application/octet-stream" }) }))
  })

  it("loads a theme package from a GitHub blob URL", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url === "https://raw.githubusercontent.com/acme/thism-themes/main/themes/repo-command.thism-theme.json") {
        return response(themePackage)
      }
      throw new Error(`unexpected ${url}`)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/thism-themes/blob/main/themes/repo-command.thism-theme.json", fetchMock)).resolves.toBe(themePackage)
  })

  it("falls back to repository contents when release asset downloads are blocked", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url === "https://api.github.com/repos/acme/thism-themes/releases/latest") {
        return response({
          assets: [
            {
              name: "repo-command.thism-theme.json",
              url: "https://api.github.com/assets/2",
              browser_download_url: "https://github.com/acme/theme.json",
            },
          ],
        })
      }
      if (url === "https://api.github.com/assets/2" || url === "https://github.com/acme/theme.json") {
        throw new TypeError("Failed to fetch")
      }
      if (url === "https://api.github.com/repos/acme/thism-themes/contents/thism-theme.json") {
        return response({ download_url: "https://raw.githubusercontent.com/acme/thism-themes/main/thism-theme.json" })
      }
      if (url === "https://raw.githubusercontent.com/acme/thism-themes/main/thism-theme.json") {
        return response(themePackage)
      }
      return response({ message: "Not Found" }, false, 404)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/thism-themes", fetchMock)).resolves.toBe(themePackage)
  })

  it("fails clearly when a GitHub repository has no thism theme package", async () => {
    const fetchMock = vi.fn(async (url: string) => {
      if (url === "https://api.github.com/repos/acme/html-theme/releases/latest") {
        return response({
          assets: [{ name: "management.html", url: "https://api.github.com/assets/3", browser_download_url: "https://github.com/acme/management.html" }],
        })
      }
      return response({ message: "Not Found" }, false, 404)
    })

    await expect(loadThemePackageFromGitHub("https://github.com/acme/html-theme", fetchMock)).rejects.toThrow("No thisM theme package found")
  })
})
