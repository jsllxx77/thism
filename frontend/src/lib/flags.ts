export function countryCodeToFlagEmoji(countryCode?: string | null): string {
  const normalized = (countryCode ?? "").trim().toUpperCase()
  if (!/^[A-Z]{2}$/.test(normalized)) {
    return ""
  }

  return String.fromCodePoint(...Array.from(normalized).map((char) => 0x1f1e6 + char.charCodeAt(0) - 65))
}
