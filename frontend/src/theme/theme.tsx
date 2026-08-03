import { useCallback, useEffect, useMemo, useRef, useState } from "react"

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
}

function normalizeServerThemeSettings(settings: ThemeSettings): NormalizedThemeSettings {
  const customThemes = (settings.custom_themes ?? []).flatMap((themePackage) => {
    try { return [definitionFromThemePackage(themePackage)] } catch { return [] }
  })
  const themes = [...appThemes, ...customThemes]
  return {
    theme: isAppThemeName(settings.theme, themes) ? settings.theme : "classic",
    customThemes,
  }
}

function hasLocalThemePreference(theme: AppThemeName, customThemes: readonly ImportedThemeDefinition[]) {
  return theme !== "classic" || customThemes.length > 0
}

function serializeThemeSettings(theme: AppThemeName, customThemes: readonly ImportedThemeDefinition[]) {
  return JSON.stringify({ theme, custom_themes: customThemes.map((option) => option.package) })
}

export function AppThemeProvider({ children }: Props) {
  const { mode } = useThemeMode()
  const [customThemes, setCustomThemes] = useState<ImportedThemeDefinition[]>(getInitialCustomThemes)
  const themes = useMemo<readonly AppThemeDefinition[]>(() => [...appThemes, ...customThemes], [customThemes])
  const [theme, setThemeState] = useState<AppThemeName>(() => getInitialTheme(themes))
  const [serverSyncReady, setServerSyncReady] = useState(false)
  const [serverSyncEnabled, setServerSyncEnabled] = useState(true)
  const initialTheme = useRef(theme)
  const initialCustomThemes = useRef(customThemes)
  const localThemeChanged = useRef(false)
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
        if (settings.configured || (!localThemeChanged.current && !hasLocalThemePreference(initialTheme.current, initialCustomThemes.current))) {
          setCustomThemes(remote.customThemes)
          setThemeState(remote.theme)
        }
        lastSyncedSettings.current = serializeThemeSettings(remote.theme, remote.customThemes)
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
    const serialized = serializeThemeSettings(effectiveTheme, customThemes)
    if (serialized === lastSyncedSettings.current) return
    lastSyncedSettings.current = serialized
    api.updateThemeSettings({
      theme: effectiveTheme,
      custom_themes: customThemes.map((option) => option.package),
    }).catch(() => { lastSyncedSettings.current = null })
  }, [customThemes, effectiveTheme, serverSyncEnabled, serverSyncReady])

  const importThemePackage = useCallback((source: string) => {
    const imported = parseThemePackage(source)
    localThemeChanged.current = true
    setCustomThemes((current) => [...current.filter((option) => option.name !== imported.name), imported])
    setThemeState(imported.name)
    return imported
  }, [])
  const setTheme = useCallback((nextTheme: AppThemeName) => {
    if (!themes.some((option) => option.name === nextTheme)) return
    localThemeChanged.current = true
    setThemeState(nextTheme)
  }, [themes])
  const value = useMemo(() => ({ theme: effectiveTheme, themes, importThemePackage, setTheme }),
    [effectiveTheme, importThemePackage, setTheme, themes])
  return <AppThemeContext.Provider value={value}>{children}</AppThemeContext.Provider>
}
