import { createContext, useContext } from "react"

export type BuiltInThemeName = "classic" | "ocean" | "graphite"
export type AppThemeName = BuiltInThemeName | `custom:${string}`
export type ThemeModeName = "light" | "dark"
export type ThemeSource = "built-in" | "custom"

type ThemeTokenName = (typeof THEME_TOKEN_NAMES)[number]
type ThemeAppearanceName = (typeof THEME_APPEARANCE_VARIABLES)[number]["name"]

export type ThemeTokenSet = Partial<Record<ThemeTokenName, string>>

export type ThemeAppearance = Partial<Record<ThemeAppearanceName, string>> & {
  density?: "compact" | "comfortable" | "spacious"
  surface?: "solid" | "glass" | "command"
  background?: "solid" | "grid" | "mesh"
  navigation?: "solid" | "floating" | "transparent"
}

export type AppThemePackage = {
  type: "thism-theme"
  version: 1
  id: string
  name: string
  description?: string
  accent: string
  tokens: {
    light: ThemeTokenSet
    dark: ThemeTokenSet
  }
  appearance: ThemeAppearance
}

export type BuiltInThemeDefinition = {
  name: BuiltInThemeName
  label: string
  labelKey: BuiltInThemeName
  accent: string
  source: "built-in"
}

export type ImportedThemeDefinition = {
  name: `custom:${string}`
  label: string
  accent: string
  description?: string
  source: "custom"
  package: AppThemePackage
}

export type AppThemeDefinition = BuiltInThemeDefinition | ImportedThemeDefinition

export const appThemes = [
  { name: "classic", label: "Classic", labelKey: "classic", accent: "#2859ad", source: "built-in" },
  { name: "ocean", label: "Ocean", labelKey: "ocean", accent: "#0f766e", source: "built-in" },
  { name: "graphite", label: "Graphite", labelKey: "graphite", accent: "#334155", source: "built-in" },
] as const satisfies readonly BuiltInThemeDefinition[]

const DEFAULT_THEME: BuiltInThemeName = "classic"
const STORAGE_KEY = "thism-color-theme"
const CUSTOM_THEMES_STORAGE_KEY = "thism-custom-themes"
const BUILT_IN_THEME_NAMES = new Set<string>(appThemes.map((theme) => theme.name))

const THEME_TOKEN_NAMES = [
  "background",
  "foreground",
  "card",
  "card-foreground",
  "popover",
  "popover-foreground",
  "primary",
  "primary-foreground",
  "secondary",
  "secondary-foreground",
  "muted",
  "muted-foreground",
  "accent",
  "accent-foreground",
  "destructive",
  "destructive-foreground",
  "border",
  "input",
  "ring",
  "chart-1",
  "chart-2",
  "chart-3",
  "chart-4",
  "chart-5",
  "sidebar",
  "sidebar-foreground",
  "sidebar-primary",
  "sidebar-primary-foreground",
  "sidebar-accent",
  "sidebar-accent-foreground",
  "sidebar-border",
  "sidebar-ring",
] as const

const CORE_TOKEN_NAMES = ["background", "foreground", "card", "card-foreground", "primary", "primary-foreground", "border", "input", "ring"] as const

const THEME_APPEARANCE_VARIABLES = [
  { name: "radius", variable: "--radius", kind: "length" },
  { name: "cardRadius", variable: "--theme-card-radius", kind: "length" },
  { name: "panelRadius", variable: "--theme-panel-radius", kind: "length" },
  { name: "controlRadius", variable: "--theme-control-radius", kind: "length" },
  { name: "cardPadding", variable: "--theme-card-padding", kind: "length" },
  { name: "panelPadding", variable: "--theme-panel-padding", kind: "length" },
  { name: "fontFamily", variable: "--app-font-family", kind: "font" },
  { name: "monoFontFamily", variable: "--app-mono-font-family", kind: "font" },
  { name: "shadow", variable: "--panel-shadow", kind: "shadow" },
] as const

const THEME_RUNTIME_VARIABLES = [
  ...THEME_TOKEN_NAMES.map((token) => `--${token}`),
  "--radius",
  "--app-font-family",
  "--app-mono-font-family",
  "--app-surface-start",
  "--app-surface-end",
  "--panel-shadow",
  "--panel-hover-shadow",
  "--enterprise-shadow",
  "--enterprise-hero-shadow",
  "--theme-card-radius",
  "--theme-panel-radius",
  "--theme-control-radius",
  "--theme-density-scale",
  "--theme-card-padding",
  "--theme-panel-padding",
  "--theme-surface-blur",
  "--theme-border-width",
] as const

const THEME_DATASET_KEYS = ["themeSource", "themeSurface", "themeBackground", "themeNavigation"] as const
const HSL_TOKEN_PATTERN = /^-?\d+(?:\.\d+)?\s+\d+(?:\.\d+)?%\s+\d+(?:\.\d+)?%(?:\s*\/\s*(?:0|1|0?\.\d+|\d+(?:\.\d+)?%))?$/
const HEX_COLOR_PATTERN = /^#[0-9a-fA-F]{6}$/
const LENGTH_PATTERN = /^(?:0|(?:\d+(?:\.\d+)?)(?:rem|px|em|%)|9999px)$/
const SAFE_FONT_PATTERN = /^[\w\s"',.-]+$/
const SAFE_SHADOW_PATTERN = /^[\w\s().,#%/-]+$/

export type AppThemeContextValue = {
  theme: AppThemeName
  setTheme: (theme: AppThemeName) => void
  themes: readonly AppThemeDefinition[]
  importThemePackage: (source: string) => ImportedThemeDefinition
  removeTheme: (theme: AppThemeName) => void
}

export const AppThemeContext = createContext<AppThemeContextValue | undefined>(undefined)

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value)
}

function invariant(condition: unknown, message = "Invalid theme file"): asserts condition {
  if (!condition) {
    throw new Error(message)
  }
}

function sanitizeThemeId(value: string) {
  return value.trim().toLowerCase().replace(/[^a-z0-9-]+/g, "-").replace(/-+/g, "-").replace(/^-|-$/g, "")
}

function readOptionalString(source: Record<string, unknown>, key: string, maxLength: number) {
  const value = source[key]
  if (value === undefined) return undefined
  invariant(typeof value === "string" && value.trim().length > 0 && value.trim().length <= maxLength)
  return value.trim()
}

function readRequiredString(source: Record<string, unknown>, key: string, maxLength: number) {
  const value = readOptionalString(source, key, maxLength)
  invariant(value)
  return value
}

function normalizeTokenSet(source: unknown) {
  invariant(isRecord(source))
  const tokens: ThemeTokenSet = {}

  for (const name of THEME_TOKEN_NAMES) {
    const value = source[name]
    if (value === undefined) continue
    invariant(typeof value === "string" && HSL_TOKEN_PATTERN.test(value.trim()))
    tokens[name] = value.trim()
  }

  for (const name of CORE_TOKEN_NAMES) {
    invariant(tokens[name])
  }

  return tokens
}

function normalizeAppearance(source: unknown) {
  invariant(isRecord(source))
  const appearance: ThemeAppearance = {}

  for (const field of THEME_APPEARANCE_VARIABLES) {
    const value = source[field.name]
    if (value === undefined) continue
    invariant(typeof value === "string")
    const normalized = value.trim()
    if (field.kind === "length") {
      invariant(LENGTH_PATTERN.test(normalized))
    } else if (field.kind === "font") {
      invariant(normalized.length <= 160 && SAFE_FONT_PATTERN.test(normalized))
    } else {
      invariant(normalized.length <= 160 && SAFE_SHADOW_PATTERN.test(normalized) && !normalized.toLowerCase().includes("url("))
    }
    appearance[field.name] = normalized
  }

  const density = source.density
  if (density !== undefined) {
    invariant(density === "compact" || density === "comfortable" || density === "spacious")
    appearance.density = density
  }

  const surface = source.surface
  if (surface !== undefined) {
    invariant(surface === "solid" || surface === "glass" || surface === "command")
    appearance.surface = surface
  }

  const background = source.background
  if (background !== undefined) {
    invariant(background === "solid" || background === "grid" || background === "mesh")
    appearance.background = background
  }

  const navigation = source.navigation
  if (navigation !== undefined) {
    invariant(navigation === "solid" || navigation === "floating" || navigation === "transparent")
    appearance.navigation = navigation
  }

  return appearance
}

function normalizeThemePackage(value: unknown): AppThemePackage {
  invariant(isRecord(value))
  invariant(value.type === "thism-theme")
  invariant(value.version === 1)

  const rawId = readRequiredString(value, "id", 48)
  const id = sanitizeThemeId(rawId)
  invariant(id.length >= 2 && !BUILT_IN_THEME_NAMES.has(id))

  const name = readRequiredString(value, "name", 64)
  const description = readOptionalString(value, "description", 180)
  const accent = readRequiredString(value, "accent", 7)
  invariant(HEX_COLOR_PATTERN.test(accent))
  invariant(isRecord(value.tokens))

  const light = normalizeTokenSet(value.tokens.light)
  const dark = normalizeTokenSet(value.tokens.dark)
  const appearance = normalizeAppearance(value.appearance)

  return {
    type: "thism-theme",
    version: 1,
    id,
    name,
    description,
    accent,
    tokens: { light, dark },
    appearance,
  }
}

function definitionFromPackage(themePackage: AppThemePackage): ImportedThemeDefinition {
  return {
    name: `custom:${themePackage.id}`,
    label: themePackage.name,
    accent: themePackage.accent,
    description: themePackage.description,
    source: "custom",
    package: themePackage,
  }
}

export function parseThemePackage(source: string): ImportedThemeDefinition {
  try {
    return definitionFromPackage(normalizeThemePackage(JSON.parse(source)))
  } catch (error) {
    if (error instanceof Error && error.message === "Invalid theme file") {
      throw error
    }
    throw new Error("Invalid theme file")
  }
}

export function isAppThemeName(value: string | null, themes: readonly AppThemeDefinition[] = appThemes): value is AppThemeName {
  return value !== null && themes.some((theme) => theme.name === value)
}

export function getInitialCustomThemes(): ImportedThemeDefinition[] {
  if (typeof window === "undefined") return []
  const stored = window.localStorage.getItem(CUSTOM_THEMES_STORAGE_KEY)
  if (!stored) return []

  try {
    const parsed = JSON.parse(stored)
    invariant(Array.isArray(parsed))
    return parsed.map((item) => definitionFromPackage(normalizeThemePackage(item)))
  } catch {
    return []
  }
}

export function persistCustomThemes(themes: readonly ImportedThemeDefinition[]) {
  if (typeof window === "undefined") return
  window.localStorage.setItem(CUSTOM_THEMES_STORAGE_KEY, JSON.stringify(themes.map((theme) => theme.package)))
}

export function getInitialTheme(themes: readonly AppThemeDefinition[] = appThemes): AppThemeName {
  if (typeof window === "undefined") return DEFAULT_THEME
  const stored = window.localStorage.getItem(STORAGE_KEY)
  return isAppThemeName(stored, themes) ? stored : DEFAULT_THEME
}

function clearThemeOverrides() {
  if (typeof document === "undefined") return
  const root = document.documentElement
  for (const variable of THEME_RUNTIME_VARIABLES) {
    root.style.removeProperty(variable)
  }
  for (const key of THEME_DATASET_KEYS) {
    delete root.dataset[key]
  }
}

function setAppearanceVariable(name: ThemeAppearanceName, value: string) {
  const field = THEME_APPEARANCE_VARIABLES.find((item) => item.name === name)
  if (field) {
    document.documentElement.style.setProperty(field.variable, value)
  }
}

function densityScale(density: ThemeAppearance["density"]) {
  if (density === "compact") return "0.88"
  if (density === "spacious") return "1.14"
  return "1"
}

function surfaceBlur(surface: ThemeAppearance["surface"]) {
  if (surface === "glass") return "blur(18px) saturate(1.18)"
  return "none"
}

function shadowForHover(shadow?: string) {
  if (!shadow) return undefined
  return shadow.replace(/0\.\d+\)/, "0.28)")
}

export function applyThemeDefinition(theme: AppThemeDefinition, mode: ThemeModeName = "light") {
  if (typeof document === "undefined") return
  clearThemeOverrides()
  const root = document.documentElement
  root.dataset.theme = theme.name

  if (theme.source === "built-in") {
    return
  }

  const themePackage = theme.package
  const tokens = themePackage.tokens[mode]
  const appearance = themePackage.appearance
  root.dataset.themeSource = "custom"
  root.dataset.themeSurface = appearance.surface ?? "solid"
  root.dataset.themeBackground = appearance.background ?? "solid"
  root.dataset.themeNavigation = appearance.navigation ?? "solid"

  for (const name of THEME_TOKEN_NAMES) {
    const value = tokens[name]
    if (value) {
      root.style.setProperty(`--${name}`, value)
    }
  }

  root.style.setProperty("--app-surface-start", tokens.background ?? tokens.card ?? "")
  root.style.setProperty("--app-surface-end", tokens.muted ?? tokens.background ?? "")
  root.style.setProperty("--theme-density-scale", densityScale(appearance.density))
  root.style.setProperty("--theme-surface-blur", surfaceBlur(appearance.surface))
  root.style.setProperty("--theme-border-width", appearance.surface === "command" ? "1.5px" : "1px")

  if (appearance.shadow) {
    root.style.setProperty("--panel-shadow", appearance.shadow)
    root.style.setProperty("--enterprise-shadow", appearance.shadow)
    root.style.setProperty("--enterprise-hero-shadow", appearance.shadow)
    root.style.setProperty("--panel-hover-shadow", shadowForHover(appearance.shadow) ?? appearance.shadow)
  }

  for (const field of THEME_APPEARANCE_VARIABLES) {
    const value = appearance[field.name]
    if (value) {
      setAppearanceVariable(field.name, value)
    }
  }
}

export function applyTheme(theme: AppThemeName) {
  const definition = appThemes.find((option) => option.name === theme) ?? appThemes[0]
  applyThemeDefinition(definition)
}

export function persistTheme(theme: AppThemeName) {
  if (typeof window === "undefined") return
  window.localStorage.setItem(STORAGE_KEY, theme)
}

export function useAppTheme() {
  const context = useContext(AppThemeContext)
  if (!context) {
    throw new Error("useAppTheme must be used within AppThemeProvider")
  }
  return context
}
