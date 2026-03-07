import { createContext, useContext, useEffect, useMemo, useState } from "react"

type ThemeMode = "light" | "dark"

type ThemeModeContextValue = {
  mode: ThemeMode
  toggleMode: () => void
}

const STORAGE_KEY = "thism-theme"

function getSystemPrefersDark(): boolean {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return false
  }
  return window.matchMedia("(prefers-color-scheme: dark)").matches
}

function getInitialMode(): ThemeMode {
  if (typeof window === "undefined") return "light"
  const stored = window.localStorage.getItem(STORAGE_KEY)
  if (stored === "dark" || stored === "light") {
    return stored
  }
  return getSystemPrefersDark() ? "dark" : "light"
}

function applyMode(mode: ThemeMode) {
  if (typeof document === "undefined") return
  document.documentElement.classList.toggle("dark", mode === "dark")
}

const ThemeModeContext = createContext<ThemeModeContextValue | undefined>(undefined)

type Props = {
  children: React.ReactNode
}

export function ThemeModeProvider({ children }: Props) {
  const [mode, setMode] = useState<ThemeMode>(getInitialMode)

  useEffect(() => {
    applyMode(mode)
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, mode)
    }
  }, [mode])

  const value = useMemo(
    () => ({
      mode,
      toggleMode: () => setMode((prev) => (prev === "dark" ? "light" : "dark")),
    }),
    [mode]
  )

  return <ThemeModeContext.Provider value={value}>{children}</ThemeModeContext.Provider>
}

export function useThemeMode() {
  const context = useContext(ThemeModeContext)
  if (!context) {
    throw new Error("useThemeMode must be used within ThemeModeProvider")
  }
  return context
}

