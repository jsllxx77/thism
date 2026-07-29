import type { ReactNode } from "react"
import { useAppTheme } from "../theme/theme-context"
import { DEFAULT_SHADCN_PLUGIN } from "./default-shadcn"
import { ThemePluginRuntimeProvider } from "./runtime"
export function ConfiguredThemePluginRuntime({ children }: { children: ReactNode }) {
  const { pluginSettings, setPluginSettings } = useAppTheme()
  return <ThemePluginRuntimeProvider plugin={DEFAULT_SHADCN_PLUGIN} settings={pluginSettings[DEFAULT_SHADCN_PLUGIN.id]} onSettingsChange={(record) => setPluginSettings(DEFAULT_SHADCN_PLUGIN.id, record)}>{children}</ThemePluginRuntimeProvider>
}
