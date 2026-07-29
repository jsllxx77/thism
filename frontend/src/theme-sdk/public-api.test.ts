import { describe, expect, it } from "vitest"

import { THEME_API_VERSION, validateThemeRegistry } from "@thism/theme-sdk"
import { THEME_CONFORMANCE_COVERAGE } from "@thism/theme-sdk/testing"

describe("Theme SDK public aliases", () => {
  it("exposes runtime validation and the testing entry", () => {
    expect(THEME_API_VERSION).toBe("1.0.0")
    expect(validateThemeRegistry(null).ok).toBe(false)
    expect(THEME_CONFORMANCE_COVERAGE).toContain("keyboard")
  })
})
