import { createContext, useContext, useEffect, useMemo, useState } from "react"

export type ThemeMode = "light" | "dark"
export type ThemeModePreference = ThemeMode | "system"
type ThemeModeContextValue = { preference: ThemeModePreference; setPreference: (value: ThemeModePreference) => void; mode: ThemeMode; toggleMode: () => void }
const STORAGE_KEY = "thism-theme"
const MEDIA_QUERY = "(prefers-color-scheme: dark)"
function systemMode(): ThemeMode { return typeof window !== "undefined" && typeof window.matchMedia === "function" && window.matchMedia(MEDIA_QUERY).matches ? "dark" : "light" }
function initialPreference(): ThemeModePreference {
  if (typeof window === "undefined") return "system"
  const stored = window.localStorage.getItem(STORAGE_KEY)
  return stored === "light" || stored === "dark" || stored === "system" ? stored : "system"
}
function applyMode(mode: ThemeMode) { if (typeof document !== "undefined") document.documentElement.classList.toggle("dark", mode === "dark") }
const ThemeModeContext = createContext<ThemeModeContextValue | undefined>(undefined)
export function ThemeModeProvider({ children }: { children: React.ReactNode }) {
  const [preference, setPreference] = useState<ThemeModePreference>(initialPreference)
  const [systemPreference, setSystemPreference] = useState<ThemeMode>(systemMode)
  const mode = preference === "system" ? systemPreference : preference
  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") return
    const media = window.matchMedia(MEDIA_QUERY)
    const update = () => setSystemPreference(media.matches ? "dark" : "light")
    update(); media.addEventListener?.("change", update)
    return () => media.removeEventListener?.("change", update)
  }, [])
  useEffect(() => { applyMode(mode); if (typeof window !== "undefined") window.localStorage.setItem(STORAGE_KEY, preference) }, [mode, preference])
  const value = useMemo(() => ({ preference, setPreference, mode, toggleMode: () => setPreference(mode === "dark" ? "light" : "dark") }), [mode, preference])
  return <ThemeModeContext.Provider value={value}>{children}</ThemeModeContext.Provider>
}
export function useThemeMode() { const context = useContext(ThemeModeContext); if (!context) throw new Error("useThemeMode must be used within ThemeModeProvider"); return context }
