import type { ComponentType, ReactNode, Ref } from "react"

export const THEME_API_VERSION = "1.0.0" as const

export const HOST_OWNED_CAPABILITIES = [
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
] as const

export const REQUIRED_PRIMITIVES = [
  "Button",
  "Badge",
  "Card",
  "Input",
  "Select",
  "Checkbox",
  "Switch",
  "Tabs",
  "Dialog",
  "Sheet",
  "DropdownMenu",
  "Tooltip",
  "Popover",
  "Table",
  "Form",
  "Toast",
  "Skeleton",
  "Separator",
  "ScrollArea",
  "ResponsiveNavigation",
] as const

export const REQUIRED_SHELLS = [
  "RootShell",
  "PrimaryNavigation",
  "MobileNavigation",
  "PageContainer",
  "PageHeader",
  "SettingsSection",
  "DashboardGrid",
  "EmptyState",
  "LoadingState",
  "ErrorState",
] as const

export const REQUIRED_SLOTS = [
  "Brand",
  "UserMenu",
  "ThemeControls",
  "NotificationCenter",
] as const

export const REQUIRED_PORTALS = ["DialogPortal", "SheetPortal", "MenuPortal", "ToastPortal"] as const

export type ThemeMode = "light" | "dark"
export type ThemePrimitiveName = (typeof REQUIRED_PRIMITIVES)[number]
export type ThemeShellName = (typeof REQUIRED_SHELLS)[number]
export type ThemeSlotName = (typeof REQUIRED_SLOTS)[number]
export type ThemePortalName = (typeof REQUIRED_PORTALS)[number]

export interface ThemeComponentProps {
  children?: ReactNode
  className?: string
  disabled?: boolean
  loading?: boolean
  ref?: Ref<HTMLElement>
  [property: string]: unknown
}

export type ThemeComponent = ComponentType<ThemeComponentProps>

export interface ThemeTokens {
  light: Readonly<Record<string, string>>
  dark: Readonly<Record<string, string>>
}

interface ThemeSettingBase {
  key: string
  label: string
  description?: string
}

export interface BooleanThemeSetting extends ThemeSettingBase {
  type: "boolean"
  default: boolean
}

export interface EnumThemeSetting extends ThemeSettingBase {
  type: "enum"
  default: string
  options: readonly string[]
}

export interface NumberThemeSetting extends ThemeSettingBase {
  type: "number"
  default: number
  minimum: number
  maximum: number
  step?: number
}

export interface StringThemeSetting extends ThemeSettingBase {
  type: "string"
  default: string
  minimumLength?: number
  maximumLength: number
  pattern?: string
}

export type ThemeSetting = BooleanThemeSetting | EnumThemeSetting | NumberThemeSetting | StringThemeSetting

export interface ThemeRegistry {
  apiVersion: string
  primitives: Record<ThemePrimitiveName, ThemeComponent>
  shells: Record<ThemeShellName, ThemeComponent>
  slots: Record<ThemeSlotName, ThemeComponent>
  portals: Record<ThemePortalName, ThemeComponent>
  tokens: ThemeTokens
  settings: readonly ThemeSetting[]
}

export type ThemeContractIssueCode =
  | "invalid-api-version"
  | "unsupported-api-version"
  | "invalid-export"
  | "missing-export"
  | "invalid-tokens"
  | "invalid-setting"

export interface ThemeContractIssue {
  code: ThemeContractIssueCode
  path: string
  message: string
}

export type ThemeContractResult =
  | { ok: true; issues: [] }
  | { ok: false; issues: ThemeContractIssue[] }

function result(issues: ThemeContractIssue[]): ThemeContractResult {
  return issues.length === 0 ? { ok: true, issues: [] } : { ok: false, issues }
}

function parseVersionMajor(version: string) {
  const match = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$/.exec(version)
  return match ? Number(match[1]) : null
}

export function validateThemeApiCompatibility(version: string): ThemeContractResult {
  const major = parseVersionMajor(version)
  if (major === null) {
    return result([{ code: "invalid-api-version", path: "apiVersion", message: `Theme API version ${version || "<empty>"} is not valid semantic versioning` }])
  }
  if (major !== parseVersionMajor(THEME_API_VERSION)) {
    return result([{ code: "unsupported-api-version", path: "apiVersion", message: `Theme API ${version} is incompatible with host API ${THEME_API_VERSION}` }])
  }
  return result([])
}

function validateExports(
  issues: ThemeContractIssue[],
  groupName: "primitives" | "shells" | "slots" | "portals",
  registry: unknown,
  requiredNames: readonly string[],
) {
  const exports = registry && typeof registry === "object" ? registry as Record<string, unknown> : {}
  for (const name of requiredNames) {
    const path = `${groupName}.${name}`
    if (!(name in exports)) {
      issues.push({ code: "missing-export", path, message: `Required theme export ${path} is missing` })
      continue
    }
    const value = exports[name]
    const reactObject = typeof value === "object" && value !== null && typeof (value as { $$typeof?: unknown }).$$typeof === "symbol"
    if (typeof value !== "function" && !reactObject) {
      issues.push({ code: "invalid-export", path, message: `Theme export ${path} is not a React component` })
    }
  }
}

function validateTokens(issues: ThemeContractIssue[], tokens: unknown) {
  const modes = tokens && typeof tokens === "object" ? tokens as Record<string, unknown> : {}
  for (const mode of ["light", "dark"] as const) {
    const values = modes[mode]
    if (!values || typeof values !== "object" || Array.isArray(values) || Object.keys(values).length === 0) {
      issues.push({ code: "invalid-tokens", path: `tokens.${mode}`, message: `Theme tokens.${mode} must contain semantic token values` })
    }
  }
}

function validateSettings(issues: ThemeContractIssue[], settings: unknown) {
  if (!Array.isArray(settings)) {
    issues.push({ code: "invalid-setting", path: "settings", message: "Theme settings must be an array" })
    return
  }
  const keys = new Set<string>()
  settings.forEach((setting, index) => {
    const path = `settings.${index}`
    if (!setting || typeof setting !== "object") {
      issues.push({ code: "invalid-setting", path, message: "Theme setting must be an object" })
      return
    }
    const candidate = setting as Partial<ThemeSetting>
    if (!candidate.key || keys.has(candidate.key)) {
      issues.push({ code: "invalid-setting", path: `${path}.key`, message: "Theme setting key must be present and unique" })
    } else {
      keys.add(candidate.key)
    }
    if (!candidate.label || !["boolean", "enum", "number", "string"].includes(candidate.type ?? "")) {
      issues.push({ code: "invalid-setting", path, message: "Theme setting must have a label and supported type" })
      return
    }
    if (candidate.type === "boolean" && typeof candidate.default !== "boolean") {
      issues.push({ code: "invalid-setting", path: `${path}.default`, message: "Boolean setting default must be boolean" })
    }
    if (candidate.type === "enum") {
      const options = Array.isArray(candidate.options) ? candidate.options : []
      if (options.length === 0 || options.some((option) => typeof option !== "string") || new Set(options).size !== options.length || !options.includes(candidate.default as string)) {
        issues.push({ code: "invalid-setting", path: `${path}.options`, message: "Enum options must be non-empty, unique strings and include the default" })
      }
    }
    if (candidate.type === "number") {
      const { default: defaultValue, minimum, maximum, step } = candidate
      const validStep = step === undefined || (Number.isFinite(step) && step > 0)
      const aligned = step === undefined || (typeof defaultValue === "number" && typeof minimum === "number" && Math.abs((defaultValue - minimum) / step - Math.round((defaultValue - minimum) / step)) < 1e-9)
      if (![defaultValue, minimum, maximum].every((value) => typeof value === "number" && Number.isFinite(value)) || (minimum as number) > (maximum as number) || (defaultValue as number) < (minimum as number) || (defaultValue as number) > (maximum as number) || !validStep || !aligned) {
        issues.push({ code: "invalid-setting", path, message: "Number setting must have finite ordered bounds and a valid aligned default" })
      }
    }
    if (candidate.type === "string") {
      const minimumLength = candidate.minimumLength ?? 0
      let pattern: RegExp | null = null
      try { pattern = candidate.pattern ? new RegExp(candidate.pattern) : null } catch { pattern = null }
      const patternValid = !candidate.pattern || pattern !== null
      const defaultValue = candidate.default
      const matches = typeof defaultValue === "string" && (!pattern || pattern.test(defaultValue))
      if (!Number.isInteger(minimumLength) || !Number.isInteger(candidate.maximumLength) || minimumLength < 0 || (candidate.maximumLength ?? -1) < minimumLength || typeof defaultValue !== "string" || defaultValue.length < minimumLength || defaultValue.length > (candidate.maximumLength ?? -1) || !patternValid || !matches) {
        issues.push({ code: "invalid-setting", path, message: "String setting must have valid length constraints, pattern, and default" })
      }
    }
  })
}

export function validateThemeRegistry(registry: unknown): ThemeContractResult {
  if (!registry || typeof registry !== "object") {
    return result([{ code: "invalid-export", path: "registry", message: "Theme registry must be an object" }])
  }

  const candidate = registry as Partial<ThemeRegistry>
  const issues = [...validateThemeApiCompatibility(candidate.apiVersion ?? "").issues]
  validateExports(issues, "primitives", candidate.primitives, REQUIRED_PRIMITIVES)
  validateExports(issues, "shells", candidate.shells, REQUIRED_SHELLS)
  validateExports(issues, "slots", candidate.slots, REQUIRED_SLOTS)
  validateExports(issues, "portals", candidate.portals, REQUIRED_PORTALS)
  validateTokens(issues, candidate.tokens)
  validateSettings(issues, candidate.settings)
  return result(issues)
}
