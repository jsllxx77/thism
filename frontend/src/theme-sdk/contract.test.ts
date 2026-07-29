import { describe, expect, it } from "vitest"

import {
  HOST_OWNED_CAPABILITIES,
  REQUIRED_PORTALS,
  REQUIRED_PRIMITIVES,
  REQUIRED_SHELLS,
  REQUIRED_SLOTS,
  THEME_API_VERSION,
  validateThemeApiCompatibility,
  validateThemeRegistry,
  type ThemeRegistry,
} from "./contract"

const component = () => null

function completeRegistry(): ThemeRegistry {
  return {
    apiVersion: THEME_API_VERSION,
    primitives: Object.fromEntries(REQUIRED_PRIMITIVES.map((name) => [name, component])) as unknown as ThemeRegistry["primitives"],
    shells: Object.fromEntries(REQUIRED_SHELLS.map((name) => [name, component])) as unknown as ThemeRegistry["shells"],
    slots: Object.fromEntries(REQUIRED_SLOTS.map((name) => [name, component])) as unknown as ThemeRegistry["slots"],
    portals: Object.fromEntries(REQUIRED_PORTALS.map((name) => [name, component])) as unknown as ThemeRegistry["portals"],
    tokens: {
      light: { background: "0 0% 100%", foreground: "222 47% 11%" },
      dark: { background: "222 47% 11%", foreground: "210 40% 98%" },
    },
    settings: [],
  }
}

describe("Theme API v1 registry contract", () => {
  it("accepts a complete presentation-only registry", () => {
    expect(validateThemeRegistry(completeRegistry())).toEqual({ ok: true, issues: [] })
    expect(HOST_OWNED_CAPABILITIES).toEqual([
      "react-root",
      "routing",
      "authentication",
      "authorization",
      "localization",
      "api-transport",
      "data-loading",
      "mutations",
      "commands",
      "error-semantics",
    ])
  })

  it("rejects a partial registry instead of mixing fallback components", () => {
    const registry = completeRegistry()
    delete (registry.primitives as Partial<ThemeRegistry["primitives"]>).Dialog
    delete (registry.shells as Partial<ThemeRegistry["shells"]>).MobileNavigation

    expect(validateThemeRegistry(registry)).toEqual({
      ok: false,
      issues: [
        { code: "missing-export", path: "primitives.Dialog", message: "Required theme export primitives.Dialog is missing" },
        { code: "missing-export", path: "shells.MobileNavigation", message: "Required theme export shells.MobileNavigation is missing" },
      ],
    })
  })

  it("rejects values that are present but are not React components", () => {
    const registry = completeRegistry()
    registry.primitives.Button = {} as never

    expect(validateThemeRegistry(registry)).toEqual({
      ok: false,
      issues: [{ code: "invalid-export", path: "primitives.Button", message: "Theme export primitives.Button is not a React component" }],
    })
  })

  it("accepts supported v1 releases and rejects other major versions", () => {
    expect(validateThemeApiCompatibility("1.0.0")).toEqual({ ok: true, issues: [] })
    expect(validateThemeApiCompatibility("1.8.4+build.12")).toEqual({ ok: true, issues: [] })
    expect(validateThemeApiCompatibility("2.0.0")).toEqual({
      ok: false,
      issues: [{ code: "unsupported-api-version", path: "apiVersion", message: "Theme API 2.0.0 is incompatible with host API 1.0.0" }],
    })
  })
})
