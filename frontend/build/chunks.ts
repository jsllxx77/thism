export function getManualChunkName(id: string): string | undefined {
  const normalized = id.replace(/\\/g, "/")

  if (normalized.includes("/node_modules/recharts/")) {
    return "vendor-recharts"
  }

  return undefined
}
