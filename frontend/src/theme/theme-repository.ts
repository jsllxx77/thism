type ThemeFetch = (input: string, init?: RequestInit) => Promise<Response>

export type ThemeArchiveSource = {
  filename: string
  data: string
}

const GITHUB_API_BASE = "https://api.github.com"

function normalizeRepositoryName(value: string) {
  return value.replace(/\.git$/, "")
}

function parseRepositoryUrl(input: string) {
  const value = input.trim()
  if (!value) {
    throw new Error("Enter a GitHub theme repository URL.")
  }

  let url: URL
  try {
    url = new URL(value)
  } catch {
    throw new Error("Enter a GitHub theme repository URL.")
  }

  if (url.protocol !== "https:" || url.hostname !== "github.com") {
    throw new Error("Enter a GitHub theme repository URL.")
  }

  const parts = url.pathname.split("/").filter(Boolean)
  if (parts.length !== 2) {
    throw new Error("Enter a GitHub theme repository URL.")
  }

  const [owner, rawRepo] = parts
  const repo = normalizeRepositoryName(rawRepo)
  if (!owner || !repo) {
    throw new Error("Enter a GitHub theme repository URL.")
  }

  return { owner, repo }
}

function isThemeArchiveAsset(name: string) {
  return name.toLowerCase().endsWith(".zip")
}

function archivePriority(name: string) {
  const lower = name.toLowerCase()
  if (lower.endsWith(".thism-theme.zip")) return 0
  if (lower.endsWith(".theme.zip")) return 1
  return 2
}

function encodeBase64(bytes: Uint8Array) {
  let binary = ""
  for (let index = 0; index < bytes.length; index += 0x8000) {
    binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000))
  }
  return btoa(binary)
}

async function fetchArchive(fetchImpl: ThemeFetch, url: string, init?: RequestInit) {
  const response = await fetchImpl(url, init)
  if (!response.ok) {
    throw new Error("Unable to download theme release archive.")
  }
  return new Uint8Array(await response.arrayBuffer())
}

export async function loadThemePackageFromGitHub(
  input: string,
  fetchImpl: ThemeFetch = globalThis.fetch.bind(globalThis),
): Promise<ThemeArchiveSource> {
  const { owner, repo } = parseRepositoryUrl(input)
  const releaseResponse = await fetchImpl(`${GITHUB_API_BASE}/repos/${owner}/${repo}/releases/latest`, {
    headers: { Accept: "application/vnd.github+json" },
  })
  if (!releaseResponse.ok) {
    throw new Error("Unable to load the latest GitHub theme release.")
  }

  const release = await releaseResponse.json() as { assets?: Array<{ name?: unknown; url?: unknown; browser_download_url?: unknown }> }
  const assets = (release.assets ?? [])
    .filter((asset) => typeof asset.name === "string" && isThemeArchiveAsset(asset.name))
    .sort((left, right) => archivePriority(left.name as string) - archivePriority(right.name as string))

  const asset = assets[0]
  if (!asset || typeof asset.name !== "string") {
    throw new Error("No theme zip archive found in the latest GitHub release.")
  }

  let bytes: Uint8Array | null = null
  if (typeof asset.url === "string") {
    try {
      bytes = await fetchArchive(fetchImpl, asset.url, {
        headers: { Accept: "application/octet-stream" },
      })
    } catch {
      bytes = null
    }
  }
  if (!bytes && typeof asset.browser_download_url === "string") {
    bytes = await fetchArchive(fetchImpl, asset.browser_download_url)
  }
  if (!bytes) {
    throw new Error("Unable to download the latest GitHub theme release archive.")
  }

  return { filename: asset.name, data: encodeBase64(bytes) }
}
