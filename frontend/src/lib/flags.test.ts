import { describe, expect, it } from "vitest"
import { countryCodeToFlagAssetPath, countryCodeToFlagEmoji, countryCodeToFlagLabel, countryFlagEmojiToCode } from "./flags"

describe("flag helpers", () => {
  it("keeps emoji conversion for platforms that render regional indicators as flags", () => {
    expect(countryCodeToFlagEmoji("HK")).toBe("🇭🇰")
    expect(countryCodeToFlagEmoji(" nl ")).toBe("🇳🇱")
    expect(countryCodeToFlagEmoji("")).toBe("")
    expect(countryCodeToFlagEmoji("HKG")).toBe("")
  })

  it("returns a stable text label for PC browsers that do not render flag emoji", () => {
    expect(countryCodeToFlagLabel("HK")).toBe("HK")
    expect(countryCodeToFlagLabel(" nl ")).toBe("NL")
    expect(countryCodeToFlagLabel("🇭🇰")).toBe("HK")
    expect(countryCodeToFlagLabel(null)).toBe("")
    expect(countryCodeToFlagLabel("HKG")).toBe("")
  })

  it("resolves regional-indicator emoji to country codes", () => {
    expect(countryFlagEmojiToCode("🇭🇰")).toBe("HK")
    expect(countryFlagEmojiToCode("🇳🇱")).toBe("NL")
    expect(countryFlagEmojiToCode("🌐")).toBe("")
  })

  it("builds local SVG asset paths", () => {
    expect(countryCodeToFlagAssetPath("HK")).toBe("/assets/flags/HK.svg")
    expect(countryCodeToFlagAssetPath("🇭🇰")).toBe("/assets/flags/HK.svg")
    expect(countryCodeToFlagAssetPath("HKG")).toBe("")
  })
})
