export function normalizeCountryCode(countryCode?: string | null): string {
  const normalized = (countryCode ?? "").trim().toUpperCase()
  return /^[A-Z]{2}$/.test(normalized) ? normalized : ""
}

export function countryCodeToFlagLabel(countryCode?: string | null): string {
  return normalizeCountryCode(countryCode)
}

export function countryCodeToFlagEmoji(countryCode?: string | null): string {
  const normalized = normalizeCountryCode(countryCode)
  if (!normalized) {
    return ""
  }

  return String.fromCodePoint(...Array.from(normalized).map((char) => 0x1f1e6 + char.charCodeAt(0) - 65))
}
