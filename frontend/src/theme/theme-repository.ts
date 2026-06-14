type ThemeFetch = (input: string, init?: RequestInit) => Promise<Response>

type ParsedGitHubThemeUrl =
  | { kind: "raw"; rawUrl: string }
  | { kind: "repo"; owner: string; repo: string }

const GITHUB_API_BASE = "https://api.github.com"
const THEME_PACKAGE_CANDIDATE_PATHS = [
  "thism-theme.json",
  ".thism-theme.json",
  "theme.json",
  "themes/thism-theme.json",
  "themes/theme.json",
]

function normalizeRepositoryName(value: string) {
  return value.replace(/\.git$/, "")
}

function isThemePackageAsset(name: string) {
  const lower = name.toLowerCase()
  return lower.endsWith(".thism-theme.json") || lower === "thism-theme.json" || lower === "theme.json"
}

function parseGitHubThemeUrl(input: string): ParsedGitHubThemeUrl {
  const value = input.trim()
  if (!value) {
    throw new Error("Enter a GitHub repository URL.")
  }

  let url: URL
  try {
    url = new URL(value)
  } catch {
    throw new Error("Enter a GitHub repository URL.")
  }

  if (url.protocol !== "https:") {
    throw new Error("Enter a GitHub repository URL.")
  }

  const parts = url.pathname.split("/").filter(Boolean)
  if (url.hostname === "raw.githubusercontent.com") {
    if (parts.length < 4) throw new Error("Enter a GitHub repository URL.")
    return { kind: "raw", rawUrl: url.toString() }
  }

  if (url.hostname !== "github.com" || parts.length < 2) {
    throw new Error("Enter a GitHub repository URL.")
  }

  const [owner, rawRepo, ...rest] = parts
  const repo = normalizeRepositoryName(rawRepo)
  if (!owner || !repo) {
    throw new Error("Enter a GitHub repository URL.")
  }

  if (rest[0] === "blob" && rest.length >= 3) {
    const ref = rest[1]
    const filePath = rest.slice(2).join("/")
    return { kind: "raw", rawUrl: `https://raw.githubusercontent.com/${owner}/${repo}/${ref}/${filePath}` }
  }

  if (rest[0] === "releases" && rest[1] === "download" && rest.length >= 4) {
    return { kind: "raw", rawUrl: url.toString() }
  }

  return { kind: "repo", owner, repo }
}

async function fetchText(fetchImpl: ThemeFetch, url: string, init?: RequestInit) {
  const response = await fetchImpl(url, init)
  if (!response.ok) {
    throw new Error("Unable to download theme package.")
  }
  return response.text()
}

async function fetchJson(fetchImpl: ThemeFetch, url: string, init?: RequestInit) {
  const response = await fetchImpl(url, init)
  if (!response.ok) return null
  return response.json() as Promise<unknown>
}

function decodeBase64Content(content: string) {
  const normalized = content.replace(/\s/g, "")
  if (typeof atob === "function") {
    return atob(normalized)
  }
  return Buffer.from(normalized, "base64").toString("utf8")
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value)
}

async function loadFromLatestRelease(owner: string, repo: string, fetchImpl: ThemeFetch) {
  const release = await fetchJson(fetchImpl, `${GITHUB_API_BASE}/repos/${owner}/${repo}/releases/latest`, {
    headers: { Accept: "application/vnd.github+json" },
  }).catch(() => null)
  if (!isRecord(release) || !Array.isArray(release.assets)) {
    return null
  }

  const asset = release.assets
    .filter(isRecord)
    .find((item) => typeof item.name === "string" && isThemePackageAsset(item.name))

  if (!asset) {
    return null
  }

  if (typeof asset.url === "string") {
    try {
      return await fetchText(fetchImpl, asset.url, {
        headers: { Accept: "application/octet-stream" },
      })
    } catch {
      // Fall through to browser_download_url. GitHub's asset API may redirect
      // differently across environments.
    }
  }

  if (typeof asset.browser_download_url === "string") {
    try {
      return await fetchText(fetchImpl, asset.browser_download_url)
    } catch {
      return null
    }
  }

  return null
}

async function loadFromRepositoryContents(owner: string, repo: string, fetchImpl: ThemeFetch) {
  for (const path of THEME_PACKAGE_CANDIDATE_PATHS) {
    const content = await fetchJson(fetchImpl, `${GITHUB_API_BASE}/repos/${owner}/${repo}/contents/${path}`, {
      headers: { Accept: "application/vnd.github+json" },
    })
    if (!isRecord(content)) continue

    if (typeof content.download_url === "string") {
      return fetchText(fetchImpl, content.download_url)
    }

    if (content.encoding === "base64" && typeof content.content === "string") {
      return decodeBase64Content(content.content)
    }
  }

  return null
}

export async function loadThemePackageFromGitHub(input: string, fetchImpl: ThemeFetch = globalThis.fetch.bind(globalThis)) {
  const parsed = parseGitHubThemeUrl(input)
  if (parsed.kind === "raw") {
    return fetchText(fetchImpl, parsed.rawUrl)
  }

  const releaseTheme = await loadFromLatestRelease(parsed.owner, parsed.repo, fetchImpl)
  if (releaseTheme) {
    return releaseTheme
  }

  const repositoryTheme = await loadFromRepositoryContents(parsed.owner, parsed.repo, fetchImpl)
  if (repositoryTheme) {
    return repositoryTheme
  }

  throw new Error("No thisM theme package found in that GitHub repository.")
}
