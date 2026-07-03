export function normalizeCountryCode(countryCode?: string | null): string {
  const normalized = (countryCode ?? "").trim().toUpperCase()
  return /^[A-Z]{2}$/.test(normalized) ? normalized : ""
}

export function countryFlagEmojiToCode(flag?: string | null): string {
  const chars = Array.from((flag ?? "").trim())
  if (chars.length !== 2) {
    return ""
  }

  const codePoints = chars.map((char) => char.codePointAt(0) ?? 0)
  if (codePoints.some((codePoint) => codePoint < 0x1f1e6 || codePoint > 0x1f1ff)) {
    return ""
  }

  return String.fromCodePoint(...codePoints.map((codePoint) => codePoint - 0x1f1e6 + 65))
}

export function resolveCountryFlagCode(countryCode?: string | null): string {
  return normalizeCountryCode(countryCode) || countryFlagEmojiToCode(countryCode)
}

export function countryCodeToFlagLabel(countryCode?: string | null): string {
  return resolveCountryFlagCode(countryCode)
}

export function countryCodeToFlagEmoji(countryCode?: string | null): string {
  const normalized = resolveCountryFlagCode(countryCode)
  if (!normalized) {
    return ""
  }

  return String.fromCodePoint(...Array.from(normalized).map((char) => 0x1f1e6 + char.charCodeAt(0) - 65))
}

export function countryCodeToFlagAssetPath(countryCode?: string | null): string {
  const normalized = resolveCountryFlagCode(countryCode)
  return normalized ? `/assets/flags/${normalized}.svg` : ""
}
