import { describe, expect, it } from "vitest"
import { countryCodeToFlagEmoji, countryCodeToFlagLabel } from "./flags"

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
    expect(countryCodeToFlagLabel(null)).toBe("")
    expect(countryCodeToFlagLabel("HKG")).toBe("")
  })
})
