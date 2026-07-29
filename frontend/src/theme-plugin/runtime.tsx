/* eslint-disable react-refresh/only-export-components */
import { createContext, useCallback, useContext, useEffect, useMemo, type CSSProperties, type ReactNode } from "react"

import { migrateThemeSettings, validateThemeRegistry, validateThemeSettingValue, type ThemeMode, type ThemePluginSettingsRecord, type ThemeRegistry, type ThemeSettingValidationResult, type ThemeSettingValue, type ThemeSettingsMigrationIssue } from "@thism/theme-sdk"
import { useThemeMode } from "../theme/mode"
import { DEFAULT_SHADCN_PLUGIN } from "./default-shadcn"
import type { ThemePluginModule } from "./types"
export type { ThemePluginModule } from "./types"

const ThemePluginContext = createContext<ThemePluginModule>(DEFAULT_SHADCN_PLUGIN)
type SettingsContext = { settings: Readonly<Record<string, ThemeSettingValue>>; migrationIssues: readonly ThemeSettingsMigrationIssue[]; updateSetting: (key: string, value: unknown) => ThemeSettingValidationResult }
const defaultSettings = migrateThemeSettings(DEFAULT_SHADCN_PLUGIN.registry.settings, DEFAULT_SHADCN_PLUGIN.version).record.values
const ThemePluginSettingsContext = createContext<SettingsContext>({ settings: defaultSettings, migrationIssues: [], updateSetting: (key) => ({ ok: false, message: `Setting ${key} is unavailable outside the theme runtime` }) })
type Props = { plugin: ThemePluginModule; settings?: ThemePluginSettingsRecord; onSettingsChange?: (record: ThemePluginSettingsRecord) => void; children: ReactNode }
function pluginTokenStyle(registry: ThemeRegistry, mode: ThemeMode, settings: Readonly<Record<string, ThemeSettingValue>>) {
  return { ...Object.fromEntries(Object.entries(registry.tokens[mode]).map(([name, value]) => [`--thism-plugin-${name}`, value])), "--thism-plugin-content-width": `${settings.contentWidth ?? 1280}px` } as CSSProperties
}
export function ThemePluginRuntimeProvider({ plugin, settings: persistedSettings, onSettingsChange, children }: Props) {
  const { mode } = useThemeMode()
  const validation = useMemo(() => validateThemeRegistry(plugin.registry), [plugin.registry])
  const migration = useMemo(() => migrateThemeSettings(plugin.registry.settings, plugin.version, persistedSettings), [persistedSettings, plugin.registry.settings, plugin.version])
  useEffect(() => {
    if (!onSettingsChange) return
    const current = JSON.stringify(persistedSettings ?? null)
    if (current !== JSON.stringify(migration.record)) onSettingsChange(migration.record)
  }, [migration.record, onSettingsChange, persistedSettings])
  const updateSetting = useCallback((key: string, value: unknown): ThemeSettingValidationResult => {
    const declaration = plugin.registry.settings.find((setting) => setting.key === key)
    if (!declaration) return { ok: false, message: `Setting ${key} is not declared by ${plugin.id}` }
    const result = validateThemeSettingValue(declaration, value)
    if (result.ok) onSettingsChange?.({ version: plugin.version, values: { ...migration.record.values, [key]: result.value } })
    return result
  }, [migration.record.values, onSettingsChange, plugin.id, plugin.registry.settings, plugin.version])
  const settingsValue = useMemo(() => ({ settings: migration.record.values, migrationIssues: migration.issues, updateSetting }), [migration, updateSetting])
  if (!validation.ok) {
    const details = validation.issues.map((issue) => `${issue.path}: ${issue.message}`).join("; ")
    throw new Error(`Invalid theme plugin ${plugin.id}@${plugin.version}: ${details}`)
  }
  return <ThemePluginContext.Provider value={plugin}><ThemePluginSettingsContext.Provider value={settingsValue}><div data-testid="theme-plugin-root" data-thism-theme-root="" data-thism-theme-id={plugin.id} data-thism-theme-version={plugin.version} data-thism-theme-mode={mode} data-thism-theme-compact={String(migration.record.values.compact === true)} data-thism-theme-navigation={String(migration.record.values.navigation ?? "sidebar")} data-thism-theme-brand={String(migration.record.values.brandLabel ?? "thisM")} style={pluginTokenStyle(plugin.registry, mode, migration.record.values)}>{children}</div></ThemePluginSettingsContext.Provider></ThemePluginContext.Provider>
}
export function useThemePlugin() { return useContext(ThemePluginContext) }
export function useThemeRegistry() { return useThemePlugin().registry }
export function useThemePluginSettings() { return useContext(ThemePluginSettingsContext) }
