import { createContext, useContext, useMemo, type CSSProperties, type ReactNode } from "react"

import { validateThemeRegistry, type ThemeMode, type ThemeRegistry } from "@thism/theme-sdk"
import { useThemeMode } from "../theme/mode"
import { DEFAULT_SHADCN_PLUGIN } from "./default-shadcn"
import type { ThemePluginModule } from "./types"

export type { ThemePluginModule } from "./types"

const ThemePluginContext = createContext<ThemePluginModule>(DEFAULT_SHADCN_PLUGIN)

type Props = {
  plugin: ThemePluginModule
  children: ReactNode
}

function pluginTokenStyle(registry: ThemeRegistry, mode: ThemeMode) {
  return Object.fromEntries(
    Object.entries(registry.tokens[mode]).map(([name, value]) => [`--thism-plugin-${name}`, value]),
  ) as CSSProperties
}

export function ThemePluginRuntimeProvider({ plugin, children }: Props) {
  const { mode } = useThemeMode()
  const validation = useMemo(() => validateThemeRegistry(plugin.registry), [plugin.registry])

  if (!validation.ok) {
    const details = validation.issues.map((issue) => `${issue.path}: ${issue.message}`).join("; ")
    throw new Error(`Invalid theme plugin ${plugin.id}@${plugin.version}: ${details}`)
  }

  return (
    <ThemePluginContext.Provider value={plugin}>
      <div
        data-testid="theme-plugin-root"
        data-thism-theme-root=""
        data-thism-theme-id={plugin.id}
        data-thism-theme-version={plugin.version}
        data-thism-theme-mode={mode}
        style={pluginTokenStyle(plugin.registry, mode)}
      >
        {children}
      </div>
    </ThemePluginContext.Provider>
  )
}

export function useThemePlugin() {
  return useContext(ThemePluginContext)
}

export function useThemeRegistry() {
  return useThemePlugin().registry
}
