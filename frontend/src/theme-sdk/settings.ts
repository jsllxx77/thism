import type { ThemeSetting } from "./contract"

export type ThemeSettingValue = boolean | number | string
export type ThemeSettingValues = Record<string, ThemeSettingValue>

export interface ThemePluginSettingsRecord {
  version: string
  values: ThemeSettingValues
}

export type ThemeSettingValidationResult =
  | { ok: true; value: ThemeSettingValue }
  | { ok: false; message: string }

export type ThemeSettingsMigrationIssueCode = "defaulted-invalid" | "dropped-removed"

export interface ThemeSettingsMigrationIssue {
  code: ThemeSettingsMigrationIssueCode
  key: string
  message: string
}

export interface ThemeSettingsMigrationResult {
  record: ThemePluginSettingsRecord
  issues: ThemeSettingsMigrationIssue[]
}

function matchesStep(value: number, minimum: number, step?: number) {
  if (step === undefined) return true
  if (!Number.isFinite(step) || step <= 0) return false
  const steps = (value - minimum) / step
  return Math.abs(steps - Math.round(steps)) < 1e-9
}

export function validateThemeSettingValue(setting: ThemeSetting, value: unknown): ThemeSettingValidationResult {
  if (setting.type === "boolean") {
    return typeof value === "boolean"
      ? { ok: true, value }
      : { ok: false, message: `Setting ${setting.key} must be boolean` }
  }

  if (setting.type === "enum") {
    return typeof value === "string" && setting.options.includes(value)
      ? { ok: true, value }
      : { ok: false, message: `Setting ${setting.key} must be one of: ${setting.options.join(", ")}` }
  }

  if (setting.type === "number") {
    const valid = typeof value === "number"
      && Number.isFinite(value)
      && value >= setting.minimum
      && value <= setting.maximum
      && matchesStep(value, setting.minimum, setting.step)
    return valid
      ? { ok: true, value }
      : { ok: false, message: `Setting ${setting.key} must be between ${setting.minimum} and ${setting.maximum}` }
  }

  if (typeof value !== "string") {
    return { ok: false, message: `Setting ${setting.key} must be a string` }
  }
  if (setting.minimumLength !== undefined && value.length < setting.minimumLength) {
    return { ok: false, message: `Setting ${setting.key} must be at least ${setting.minimumLength} characters` }
  }
  if (value.length > setting.maximumLength) {
    return { ok: false, message: `Setting ${setting.key} must be at most ${setting.maximumLength} characters` }
  }
  if (setting.pattern) {
    try {
      if (!new RegExp(setting.pattern).test(value)) {
        return { ok: false, message: `Setting ${setting.key} does not match its required format` }
      }
    } catch {
      return { ok: false, message: `Setting ${setting.key} has an invalid declaration pattern` }
    }
  }
  return { ok: true, value }
}

export function migrateThemeSettings(
  declarations: readonly ThemeSetting[],
  targetVersion: string,
  stored?: ThemePluginSettingsRecord,
): ThemeSettingsMigrationResult {
  const issues: ThemeSettingsMigrationIssue[] = []
  const values: ThemeSettingValues = {}
  const declaredKeys = new Set(declarations.map((setting) => setting.key))

  for (const setting of declarations) {
    const persisted = stored?.values?.[setting.key]
    if (persisted === undefined) {
      values[setting.key] = setting.default
      continue
    }
    const validation = validateThemeSettingValue(setting, persisted)
    if (validation.ok) {
      values[setting.key] = validation.value
    } else {
      values[setting.key] = setting.default
      issues.push({
        code: "defaulted-invalid",
        key: setting.key,
        message: `Setting ${setting.key} is incompatible and was reset to its default`,
      })
    }
  }

  for (const key of Object.keys(stored?.values ?? {})) {
    if (!declaredKeys.has(key)) {
      issues.push({
        code: "dropped-removed",
        key,
        message: `Setting ${key} is no longer declared and was removed`,
      })
    }
  }

  return { record: { version: targetVersion, values }, issues }
}
