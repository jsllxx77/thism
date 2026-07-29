import { describe, expect, it } from "vitest"

import {
  migrateThemeSettings,
  validateThemeSettingValue,
  type ThemePluginSettingsRecord,
} from "./settings"
import type { ThemeSetting } from "./contract"

const declarations: ThemeSetting[] = [
  { key: "compact", label: "Compact density", type: "boolean", default: false },
  { key: "navigation", label: "Navigation", type: "enum", default: "sidebar", options: ["sidebar", "topbar"] },
  { key: "contentWidth", label: "Content width", type: "number", default: 1280, minimum: 960, maximum: 1600, step: 40 },
  { key: "brandLabel", label: "Brand label", type: "string", default: "thisM", minimumLength: 2, maximumLength: 24, pattern: "^[A-Za-z0-9 -]+$" },
]

describe("declarative theme settings", () => {
  it("creates a version-bound record from declaration defaults", () => {
    expect(migrateThemeSettings(declarations, "1.2.0")).toEqual({
      record: {
        version: "1.2.0",
        values: { compact: false, navigation: "sidebar", contentWidth: 1280, brandLabel: "thisM" },
      },
      issues: [],
    })
  })

  it("preserves compatible values and defaults invalid or removed values during migration", () => {
    const stored: ThemePluginSettingsRecord = {
      version: "1.0.0",
      values: {
        compact: true,
        navigation: "rail",
        contentWidth: 1320,
        brandLabel: "x",
        removedSetting: "legacy",
      },
    }

    expect(migrateThemeSettings(declarations, "1.2.0", stored)).toEqual({
      record: {
        version: "1.2.0",
        values: { compact: true, navigation: "sidebar", contentWidth: 1320, brandLabel: "thisM" },
      },
      issues: [
        { code: "defaulted-invalid", key: "navigation", message: "Setting navigation is incompatible and was reset to its default" },
        { code: "defaulted-invalid", key: "brandLabel", message: "Setting brandLabel is incompatible and was reset to its default" },
        { code: "dropped-removed", key: "removedSetting", message: "Setting removedSetting is no longer declared and was removed" },
      ],
    })
  })

  it("rejects values outside boolean, enum, numeric, and string constraints", () => {
    expect(validateThemeSettingValue(declarations[0], "true").ok).toBe(false)
    expect(validateThemeSettingValue(declarations[1], "rail").ok).toBe(false)
    expect(validateThemeSettingValue(declarations[2], 1300).ok).toBe(false)
    expect(validateThemeSettingValue(declarations[3], "x").ok).toBe(false)
    expect(validateThemeSettingValue(declarations[3], "thisM ops")).toEqual({ ok: true, value: "thisM ops" })
  })
})
