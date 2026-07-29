import { useCallback, useEffect, useMemo, useRef, useState } from "react"

import type { ThemePluginSettingsRecord } from "@thism/theme-sdk"
import { api, type ThemeSettings } from "../lib/api"
import {
  AppThemeContext, appThemes, applyThemeDefinition, definitionFromThemePackage,
  getInitialCustomThemes, getInitialTheme, isAppThemeName, parseThemePackage,
  persistCustomThemes, persistTheme, type AppThemeDefinition, type AppThemeName,
  type ImportedThemeDefinition,
} from "./theme-context"
import { useThemeMode } from "./mode"

type Props = { children: React.ReactNode }
type NormalizedThemeSettings = {
  theme: AppThemeName
  customThemes: ImportedThemeDefinition[]
  pluginSettings: Record<string, ThemePluginSettingsRecord>
}

function normalizeServerThemeSettings(settings: ThemeSettings): NormalizedThemeSettings {
  const customThemes = (settings.custom_themes ?? []).flatMap((themePackage) => {
    try { return [definitionFromThemePackage(themePackage)] } catch { return [] }
  })
  const themes = [...appThemes, ...customThemes]
  return {
    theme: isAppThemeName(settings.theme, themes) ? settings.theme : "classic",
    customThemes,
    pluginSettings: settings.plugin_settings ?? {},
  }
}

function hasLocalThemePreference(theme: AppThemeName, customThemes: readonly ImportedThemeDefinition[]) {
  return theme !== "classic" || customThemes.length > 0
}

function serializeThemeSettings(theme: AppThemeName, customThemes: readonly ImportedThemeDefinition[], pluginSettings: Record<string, ThemePluginSettingsRecord>) {
  return JSON.stringify({ theme, custom_themes: customThemes.map((option) => option.package), plugin_settings: pluginSettings })
}

export function AppThemeProvider({ children }: Props) {
  const { mode } = useThemeMode()
  const [customThemes, setCustomThemes] = useState<ImportedThemeDefinition[]>(getInitialCustomThemes)
  const themes = useMemo<readonly AppThemeDefinition[]>(() => [...appThemes, ...customThemes], [customThemes])
  const [theme, setTheme] = useState<AppThemeName>(() => getInitialTheme(themes))
  const [pluginSettings, setAllPluginSettings] = useState<Record<string, ThemePluginSettingsRecord>>({})
  const [serverSyncReady, setServerSyncReady] = useState(false)
  const [serverSyncEnabled, setServerSyncEnabled] = useState(true)
  const initialTheme = useRef(theme)
  const initialCustomThemes = useRef(customThemes)
  const lastSyncedSettings = useRef<string | null>(null)
  const effectiveTheme = themes.some((option) => option.name === theme) ? theme : "classic"

  useEffect(() => {
    let cancelled = false
    async function loadServerThemeSettings() {
      if (typeof api.themeSettings !== "function" || typeof api.updateThemeSettings !== "function") { setServerSyncEnabled(false); return }
      try {
        const settings = await api.themeSettings()
        if (cancelled) return
        const remote = normalizeServerThemeSettings(settings)
        setAllPluginSettings(remote.pluginSettings)
        if (settings.configured || !hasLocalThemePreference(initialTheme.current, initialCustomThemes.current)) {
          setCustomThemes(remote.customThemes)
          setTheme(remote.theme)
        }
        lastSyncedSettings.current = serializeThemeSettings(remote.theme, remote.customThemes, remote.pluginSettings)
        setServerSyncReady(true)
      } catch { if (!cancelled) setServerSyncEnabled(false) }
    }
    loadServerThemeSettings()
    return () => { cancelled = true }
  }, [])

  useEffect(() => {
    const definition = themes.find((option) => option.name === effectiveTheme) ?? appThemes[0]
    applyThemeDefinition(definition, mode)
    persistTheme(effectiveTheme)
  }, [effectiveTheme, mode, themes])
  useEffect(() => { persistCustomThemes(customThemes) }, [customThemes])
  useEffect(() => {
    if (!serverSyncEnabled || !serverSyncReady) return
    const serialized = serializeThemeSettings(effectiveTheme, customThemes, pluginSettings)
    if (serialized === lastSyncedSettings.current) return
    lastSyncedSettings.current = serialized
    api.updateThemeSettings({
      theme: effectiveTheme,
      custom_themes: customThemes.map((option) => option.package),
      ...(Object.keys(pluginSettings).length > 0 ? { plugin_settings: pluginSettings } : {}),
    }).catch(() => { lastSyncedSettings.current = null })
  }, [customThemes, effectiveTheme, pluginSettings, serverSyncEnabled, serverSyncReady])

  const importThemePackage = useCallback((source: string) => {
    const imported = parseThemePackage(source)
    setCustomThemes((current) => [...current.filter((option) => option.name !== imported.name), imported])
    setTheme(imported.name)
    return imported
  }, [])
  const removeTheme = useCallback((themeName: AppThemeName) => {
    setCustomThemes((current) => current.filter((option) => option.name !== themeName))
    if (themeName === effectiveTheme) setTheme("classic")
  }, [effectiveTheme])
  const setPluginSettings = useCallback((pluginID: string, record: ThemePluginSettingsRecord) => {
    setAllPluginSettings((current) => ({ ...current, [pluginID]: record }))
  }, [])
  const value = useMemo(() => ({ theme: effectiveTheme, setTheme, themes, importThemePackage, removeTheme, pluginSettings, setPluginSettings }),
    [effectiveTheme, importThemePackage, pluginSettings, removeTheme, setPluginSettings, themes])
  return <AppThemeContext.Provider value={value}>{children}</AppThemeContext.Provider>
}
