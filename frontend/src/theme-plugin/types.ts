import type { ThemeRegistry } from "@thism/theme-sdk"

export interface ThemePluginModule {
  id: string
  name: string
  version: string
  source: "embedded" | "installed"
  removable: boolean
  activeByDefault?: boolean
  registry: ThemeRegistry
}
