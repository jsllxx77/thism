import { useState } from "react"
import type { ThemeSetting, ThemeSettingValue } from "@thism/theme-sdk"
import { useLanguage } from "../../i18n/language"
import { useThemeMode, type ThemeModePreference } from "../../theme/mode"
import { useThemePlugin, useThemePluginSettings } from "../../theme-plugin/runtime"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../ui/card"
import { Input } from "../ui/input"

function fieldID(key: string) { return `theme-plugin-setting-${key}` }
type FieldsProps = { declarations: readonly ThemeSetting[]; settings: Readonly<Record<string, ThemeSettingValue>>; updateSetting: (key: string, value: unknown) => { ok: boolean; message?: string }; stepLabel: string }
function SettingsFields({ declarations, settings, updateSetting, stepLabel }: FieldsProps) {
  const [drafts, setDrafts] = useState(() => Object.fromEntries(Object.entries(settings).map(([key, value]) => [key, String(value)])))
  const [errors, setErrors] = useState<Record<string, string>>({})
  const commit = (setting: ThemeSetting, value: unknown) => {
    const result = updateSetting(setting.key, value)
    setErrors((current) => ({ ...current, [setting.key]: result.ok ? "" : result.message ?? "Invalid value" }))
  }
  return <div className="grid gap-4 border-t border-border pt-5 md:grid-cols-2">{declarations.map((setting) => {
    const id = fieldID(setting.key)
    const error = errors[setting.key]
    return <div key={setting.key} className="rounded-xl border border-border/80 bg-background/60 p-4">
      <div className="mb-3"><label className="font-medium" htmlFor={id}>{setting.label}</label>{setting.description ? <p id={`${id}-hint`} className="mt-1 text-sm text-muted-foreground">{setting.description}</p> : null}</div>
      {setting.type === "boolean" ? <input id={id} type="checkbox" className="h-11 w-11 cursor-pointer accent-primary focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" checked={settings[setting.key] === true} onChange={(event) => commit(setting, event.target.checked)} /> : null}
      {setting.type === "enum" ? <select id={id} className="h-11 w-full rounded-xl border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" value={String(settings[setting.key])} onChange={(event) => commit(setting, event.target.value)}>{setting.options.map((option) => <option key={option} value={option}>{option}</option>)}</select> : null}
      {setting.type === "number" ? <><Input id={id} className="h-11 rounded-xl" type="number" min={setting.minimum} max={setting.maximum} step={setting.step} value={drafts[setting.key] ?? ""} aria-invalid={Boolean(error)} aria-describedby={`${id}-range${error ? ` ${id}-error` : ""}`} onChange={(event) => { const raw = event.target.value; setDrafts((current) => ({ ...current, [setting.key]: raw })); if (raw !== "") commit(setting, Number(raw)) }} /><p id={`${id}-range`} className="mt-2 text-xs text-muted-foreground">{setting.minimum}–{setting.maximum}{setting.step ? ` · ${stepLabel} ${setting.step}` : ""}</p></> : null}
      {setting.type === "string" ? <Input id={id} className="h-11 rounded-xl" value={drafts[setting.key] ?? ""} maxLength={setting.maximumLength} aria-invalid={Boolean(error)} aria-describedby={error ? `${id}-error` : undefined} onChange={(event) => { const raw = event.target.value; setDrafts((current) => ({ ...current, [setting.key]: raw })); commit(setting, raw) }} /> : null}
      {error ? <p id={`${id}-error`} role="alert" className="mt-2 text-sm text-destructive">{error}</p> : null}
    </div>
  })}</div>
}

export function ThemePluginSettingsCard() {
  const { t } = useLanguage()
  const plugin = useThemePlugin()
  const { settings, migrationIssues, updateSetting } = useThemePluginSettings()
  const { preference, setPreference } = useThemeMode()
  return <Card className="enterprise-surface overflow-hidden rounded-2xl">
    <CardHeader className="gap-2"><div className="flex flex-wrap items-center justify-between gap-3"><div><CardTitle>{t("settingsPage.themePluginTitle")}</CardTitle><CardDescription>{t("settingsPage.themePluginDescription")}</CardDescription></div><span className="enterprise-chip rounded-full px-3 py-1 text-xs font-medium">{t("settingsPage.themePluginManaged")}</span></div><p className="text-xs text-muted-foreground">{plugin.name} · v{plugin.version}</p></CardHeader>
    <CardContent className="grid gap-5">
      <div className="grid gap-2 sm:grid-cols-[minmax(0,1fr)_minmax(12rem,18rem)] sm:items-center"><div><label className="font-medium" htmlFor="theme-mode-preference">{t("settingsPage.themeModeLabel")}</label><p className="text-sm text-muted-foreground">{t("settingsPage.themeModeDescription")}</p></div><select id="theme-mode-preference" className="h-11 w-full rounded-xl border border-input bg-background px-3 text-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring" value={preference} onChange={(event) => setPreference(event.target.value as ThemeModePreference)}><option value="system">{t("settingsPage.themeModeSystem")}</option><option value="light">{t("settingsPage.themeModeLight")}</option><option value="dark">{t("settingsPage.themeModeDark")}</option></select></div>
      <SettingsFields key={JSON.stringify(settings)} declarations={plugin.registry.settings} settings={settings} updateSetting={updateSetting} stepLabel={t("settingsPage.themePluginStep")} />
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground"><span>{t("settingsPage.themePluginSaved")}</span>{migrationIssues.length ? <span>{t("settingsPage.themePluginMigrated", { count: migrationIssues.length })}</span> : null}</div>
    </CardContent>
  </Card>
}
