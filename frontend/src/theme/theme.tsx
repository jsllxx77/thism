import { useCallback, useEffect, useMemo, useState } from "react"

import {
  AppThemeContext,
  appThemes,
  applyThemeDefinition,
  getInitialCustomThemes,
  getInitialTheme,
  parseThemePackage,
  persistCustomThemes,
  persistTheme,
  type AppThemeDefinition,
  type AppThemeName,
  type ImportedThemeDefinition,
} from "./theme-context"
import { useThemeMode } from "./mode"

type Props = {
  children: React.ReactNode
}

export function AppThemeProvider({ children }: Props) {
  const { mode } = useThemeMode()
  const [customThemes, setCustomThemes] = useState<ImportedThemeDefinition[]>(getInitialCustomThemes)
  const themes = useMemo<readonly AppThemeDefinition[]>(() => [...appThemes, ...customThemes], [customThemes])
  const [theme, setTheme] = useState<AppThemeName>(() => getInitialTheme(themes))
  const effectiveTheme = themes.some((option) => option.name === theme) ? theme : "classic"

  useEffect(() => {
    const definition = themes.find((option) => option.name === effectiveTheme) ?? appThemes[0]
    applyThemeDefinition(definition, mode)
    persistTheme(effectiveTheme)
  }, [effectiveTheme, mode, themes])

  useEffect(() => {
    persistCustomThemes(customThemes)
  }, [customThemes])

  const importThemePackage = useCallback((source: string) => {
    const imported = parseThemePackage(source)
    setCustomThemes((current) => [
      ...current.filter((option) => option.name !== imported.name),
      imported,
    ])
    setTheme(imported.name)
    return imported
  }, [])

  const removeTheme = useCallback((themeName: AppThemeName) => {
    setCustomThemes((current) => current.filter((option) => option.name !== themeName))
    if (themeName === effectiveTheme) {
      setTheme("classic")
    }
  }, [effectiveTheme])

  const value = useMemo(
    () => ({
      theme: effectiveTheme,
      setTheme,
      themes,
      importThemePackage,
      removeTheme,
    }),
    [effectiveTheme, importThemePackage, removeTheme, themes],
  )

  return <AppThemeContext.Provider value={value}>{children}</AppThemeContext.Provider>
}
